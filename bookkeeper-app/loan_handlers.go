// bookkeeper-app/loan_handlers.go
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mattn/go-sqlite3"
)

// CreateLoan (无修改)
func (h *DBHandler) CreateLoan(c *gin.Context) {
	var req UpdateLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的贷款数据: " + err.Error()})
		return
	}

	status := "active"
	createdAt := time.Now().Format(time.RFC3339)

	stmt, err := h.DB.Prepare("INSERT INTO loans(principal, interest_rate, loan_date, repayment_date, description, status, created_at) VALUES(?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库准备语句失败: " + err.Error()})
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(req.Principal, *req.InterestRate, req.LoanDate, req.RepaymentDate, req.Description, status, createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建贷款失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "贷款创建成功"})
}

// UpdateLoan (无修改)
func (h *DBHandler) UpdateLoan(c *gin.Context) {
	id := c.Param("id")
	var req UpdateLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	stmt, err := h.DB.Prepare("UPDATE loans SET principal=?, interest_rate=?, loan_date=?, repayment_date=?, description=? WHERE id=?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库准备更新语句失败: " + err.Error()})
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(req.Principal, *req.InterestRate, req.LoanDate, req.RepaymentDate, req.Description, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新贷款失败: " + err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的贷款"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "贷款更新成功"})
}

// GetLoans (无修改)
func (h *DBHandler) GetLoans(c *gin.Context) {
	rows, err := h.DB.Query("SELECT id, principal, interest_rate, loan_date, repayment_date, description, status, created_at FROM loans ORDER BY status ASC, loan_date DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询贷款失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var loans []LoanResponse
	for rows.Next() {
		var l Loan
		var repaymentDate, description sql.NullString
		if err := rows.Scan(&l.ID, &l.Principal, &l.InterestRate, &l.LoanDate, &repaymentDate, &description, &l.Status, &l.CreatedAt); err != nil {
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
		err := h.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'repayment' AND related_loan_id = ?", l.ID).Scan(&totalRepaid)
		if err != nil {
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

// 【新增】SettleLoan 一键还清贷款，包含资金流动
func (h *DBHandler) SettleLoan(c *gin.Context) {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "开启事务失败: " + err.Error()})
		return
	}
	defer tx.Rollback()

	// 1. 获取贷款当前信息和待还金额
	var principal float64
	var loanDesc sql.NullString
	err = tx.QueryRow("SELECT principal, description FROM loans WHERE id = ?", loanID).Scan(&principal, &loanDesc)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到指定的贷款"})
		return
	}
	var totalRepaid float64
	tx.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'repayment' AND related_loan_id = ?", loanID).Scan(&totalRepaid)

	outstandingBalance := principal - totalRepaid
	if outstandingBalance <= 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该贷款已还清或无需还款"})
		return
	}

	// 2. 从指定账户扣款
	res, err := tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", outstandingBalance, req.FromAccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新扣款账户余额失败: " + err.Error()})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到指定的扣款账户"})
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
		"INSERT INTO transactions (type, amount, transaction_date, description, category_id, related_loan_id, from_account_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		"repayment", outstandingBalance, req.RepaymentDate, description, loanRepaymentCategoryID, loanID, req.FromAccountID, createdAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建还款流水失败: " + err.Error()})
		return
	}

	// 4. 更新贷款状态为 'paid'
	_, err = tx.Exec("UPDATE loans SET status = ?, repayment_date = ? WHERE id = ?", "paid", req.RepaymentDate, loanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新贷款状态失败: " + err.Error()})
		return
	}

	// 5. 提交事务
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "贷款已成功还清"})
}

// UpdateLoanStatus 【修改】此接口现在只用于将 'paid' 状态改回 'active'，恢复操作
func (h *DBHandler) UpdateLoanStatus(c *gin.Context) {
	id := c.Param("id")
	var payload struct {
		Status string `json:"status" binding:"required,oneof=active"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的状态，此接口只接受 'active' 用于恢复贷款。"})
		return
	}

	// 只处理恢复为 active 的情况
	query := "UPDATE loans SET status = ?, repayment_date = NULL WHERE id = ?"
	res, err := h.DB.Exec(query, payload.Status, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复贷款状态失败: " + err.Error()})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的贷款"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "贷款状态已恢复为 'active'"})
}

// DeleteLoan (无修改)
func (h *DBHandler) DeleteLoan(c *gin.Context) {
	id := c.Param("id")

	var count int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE related_loan_id = ?", id).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查贷款使用情况失败: " + err.Error()})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("无法删除贷款，已有 %d 条还款记录与此关联", count)})
		return
	}

	res, err := h.DB.Exec("DELETE FROM loans WHERE id = ?", id)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			c.JSON(http.StatusConflict, gin.H{"error": "由于外键约束，无法删除此贷款"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除贷款失败: " + err.Error()})
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
