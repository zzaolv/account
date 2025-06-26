// bookkeeper-app/loan_handlers.go
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mattn/go-sqlite3"
)

// CreateLoan (已修改)
func (h *DBHandler) CreateLoan(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req UpdateLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的贷款数据: " + err.Error()})
		return
	}

	status := "active"
	createdAt := time.Now().Format(time.RFC3339)

	_, err := h.DB.Exec(
		"INSERT INTO loans(user_id, principal, interest_rate, loan_date, repayment_date, description, status, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?)",
		userID, req.Principal, *req.InterestRate, req.LoanDate, req.RepaymentDate, req.Description, status, createdAt,
	)
	if err != nil {
		h.Logger.Error("创建贷款失败", "error", err, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建贷款失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "贷款创建成功"})
}

// UpdateLoan (已修改)
func (h *DBHandler) UpdateLoan(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")
	var req UpdateLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	result, err := h.DB.Exec(
		"UPDATE loans SET principal=?, interest_rate=?, loan_date=?, repayment_date=?, description=? WHERE id=? AND user_id=?",
		req.Principal, *req.InterestRate, req.LoanDate, req.RepaymentDate, req.Description, id, userID,
	)
	if err != nil {
		h.Logger.Error("更新贷款失败", "error", err, "loanID", id, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新贷款失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的贷款"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "贷款更新成功"})
}

// GetLoans (已修改)
func (h *DBHandler) GetLoans(c *gin.Context) {
	userID, _ := c.Get("userID")
	logger := h.Logger.With(slog.Int64("userID", userID.(int64)))

	rows, err := h.DB.Query("SELECT id, principal, interest_rate, loan_date, repayment_date, description, status, created_at FROM loans WHERE user_id = ? ORDER BY status ASC, loan_date DESC", userID)
	if err != nil {
		logger.Error("查询贷款失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询贷款失败"})
		return
	}
	defer rows.Close()

	var loans []LoanResponse
	for rows.Next() {
		var l Loan
		var repaymentDate, description sql.NullString
		if err := rows.Scan(&l.ID, &l.Principal, &l.InterestRate, &l.LoanDate, &repaymentDate, &description, &l.Status, &l.CreatedAt); err != nil {
			logger.Error("扫描贷款数据失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "扫描贷款数据失败"})
			return
		}
		if repaymentDate.Valid {
			l.RepaymentDate = &repaymentDate.String
		}
		if description.Valid {
			l.Description = &description.String
		}

		var totalRepaid float64
		err := h.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE user_id = ? AND type = 'repayment' AND related_loan_id = ?", userID, l.ID).Scan(&totalRepaid)
		if err != nil {
			logger.Error("计算已还款额失败", "error", err, "loanID", l.ID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "计算已还款额失败"})
			return
		}

		lr := LoanResponse{
			Loan:               l,
			TotalRepaid:        totalRepaid,
			OutstandingBalance: l.Principal - totalRepaid,
		}
		loans = append(loans, lr)
	}
	c.JSON(http.StatusOK, loans)
}

// SettleLoan (已修复)
func (h *DBHandler) SettleLoan(c *gin.Context) {
	userID, _ := c.Get("userID")
	logger := h.Logger.With(slog.Int64("userID", userID.(int64)))

	loanIDStr := c.Param("id")
	loanID, err := strconv.ParseInt(loanIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的贷款ID"})
		return
	}

	var req SettleLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		logger.Error("开启事务失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "开启事务失败"})
		return
	}
	defer tx.Rollback()

	// 1. 获取贷款信息并验证归属权
	var principal float64
	var loanDesc sql.NullString
	err = tx.QueryRow("SELECT principal, description FROM loans WHERE id = ? AND user_id = ?", loanID, userID).Scan(&principal, &loanDesc)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到指定的贷款"})
		} else {
			logger.Error("查询贷款信息失败", "error", err, "loanID", loanID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		}
		return
	}

	var totalRepaid float64
	tx.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE user_id = ? AND type = 'repayment' AND related_loan_id = ?", userID, loanID).Scan(&totalRepaid)
	outstandingBalance := principal - totalRepaid
	if outstandingBalance <= 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该贷款已还清或无需还款"})
		return
	}

	// 2. 从指定账户扣款 (先验证账户归属和余额)
	var fromAccountBalance float64
	err = tx.QueryRow("SELECT balance FROM accounts WHERE id = ? AND user_id = ?", req.FromAccountID, userID).Scan(&fromAccountBalance)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到指定的扣款账户或无权操作"})
		} else {
			logger.Error("查询扣款账户余额失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询扣款账户余额失败"})
		}
		return
	}

	if fromAccountBalance < outstandingBalance {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("扣款账户余额不足 (当前: %.2f, 需要: %.2f)", fromAccountBalance, outstandingBalance)})
		return
	}

	_, err = tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", outstandingBalance, req.FromAccountID)
	if err != nil {
		logger.Error("更新扣款账户余额失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新扣款账户余额失败"})
		return
	}

	// 3. 创建还款流水
	description := req.Description
	if description == "" {
		description = fmt.Sprintf("还清贷款: %s", loanDesc.String)
	}
	createdAt := time.Now().Format(time.RFC3339)
	loanRepaymentCategoryID := "loan_repayment"
	_, err = tx.Exec(
		"INSERT INTO transactions (user_id, type, amount, transaction_date, description, category_id, related_loan_id, from_account_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userID, "repayment", outstandingBalance, req.RepaymentDate, description, loanRepaymentCategoryID, loanID, req.FromAccountID, createdAt,
	)
	if err != nil {
		logger.Error("创建还款流水失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建还款流水失败"})
		return
	}

	// 4. 更新贷款状态
	_, err = tx.Exec("UPDATE loans SET status = ?, repayment_date = ? WHERE id = ? AND user_id = ?", "paid", req.RepaymentDate, loanID, userID)
	if err != nil {
		logger.Error("更新贷款状态失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新贷款状态失败"})
		return
	}

	if err := tx.Commit(); err != nil {
		logger.Error("提交事务失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "贷款已成功还清"})
}

// UpdateLoanStatus (已修改)
func (h *DBHandler) UpdateLoanStatus(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")
	var payload struct {
		Status string `json:"status" binding:"required,oneof=active"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的状态，此接口只接受 'active' 用于恢复贷款。"})
		return
	}

	query := "UPDATE loans SET status = ?, repayment_date = NULL WHERE id = ? AND user_id = ?"
	res, err := h.DB.Exec(query, payload.Status, id, userID)
	if err != nil {
		h.Logger.Error("恢复贷款状态失败", "error", err, "loanID", id, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复贷款状态失败"})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的贷款"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "贷款状态已恢复为 'active'"})
}

// DeleteLoan (已修改)
func (h *DBHandler) DeleteLoan(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	var count int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE related_loan_id = ? AND user_id = ?", id, userID).Scan(&count)
	if err != nil {
		h.Logger.Error("检查贷款使用情况失败", "error", err, "loanID", id, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查贷款使用情况失败"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("无法删除贷款，已有 %d 条还款记录与此关联", count)})
		return
	}

	res, err := h.DB.Exec("DELETE FROM loans WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		h.Logger.Error("删除贷款失败", "error", err, "loanID", id, slog.Int64("userID", userID.(int64)))
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			c.JSON(http.StatusConflict, gin.H{"error": "由于外键约束，无法删除此贷款"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除贷款失败"})
		}
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的贷款"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "贷款删除成功"})
}
