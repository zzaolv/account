// bookkeeper-app/loan_handlers.go
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mattn/go-sqlite3"
)

// CreateLoan (完整的，无省略)
func (h *DBHandler) CreateLoan(c *gin.Context) {
	var req UpdateLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的贷款数据: " + err.Error()})
		return
	}
	// ... 后端日期校验逻辑 (如果已添加) ...

	status := "active"
	createdAt := time.Now().Format(time.RFC3339)

	stmt, err := h.DB.Prepare("INSERT INTO loans(principal, interest_rate, loan_date, repayment_date, description, status, created_at) VALUES(?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库准备语句失败: " + err.Error()})
		return
	}
	defer stmt.Close()

	// ======================= 核心修改点 (2) =======================
	// 使用 *req.InterestRate 来获取指针指向的值
	_, err = stmt.Exec(req.Principal, *req.InterestRate, req.LoanDate, req.RepaymentDate, req.Description, status, createdAt)
	// ============================================================
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建贷款失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "贷款创建成功"})
}

// 【新增】UpdateLoan 编辑一笔现有的贷款
func (h *DBHandler) UpdateLoan(c *gin.Context) {
	id := c.Param("id")
	var req UpdateLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}
	// ... 后端日期校验逻辑 (如果已添加) ...

	stmt, err := h.DB.Prepare("UPDATE loans SET principal=?, interest_rate=?, loan_date=?, repayment_date=?, description=? WHERE id=?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库准备更新语句失败: " + err.Error()})
		return
	}
	defer stmt.Close()

	// ======================= 核心修改点 (3) =======================
	// 使用 *req.InterestRate 来获取指针指向的值
	result, err := stmt.Exec(req.Principal, *req.InterestRate, req.LoanDate, req.RepaymentDate, req.Description, id)
	// ============================================================
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

// 【新增】UpdateLoan 编辑一笔现有的贷款
//func (h *DBHandler) UpdateLoan(c *gin.Context) {
//	id := c.Param("id")
//	var req UpdateLoanRequest
//	if err := c.ShouldBindJSON(&req); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
//		return
//	}
//
//	stmt, err := h.DB.Prepare("UPDATE loans SET principal=?, interest_rate=?, loan_date=?, repayment_date=?, description=? WHERE id=?")
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库准备更新语句失败: " + err.Error()})
//		return
//	}
//	defer stmt.Close()
//
//	result, err := stmt.Exec(req.Principal, req.InterestRate, req.LoanDate, req.RepaymentDate, req.Description, id)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新贷款失败: " + err.Error()})
//		return
//	}
//
//	rowsAffected, _ := result.RowsAffected()
//	if rowsAffected == 0 {
//		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的贷款"})
//		return
//	}
//
//	c.JSON(http.StatusOK, gin.H{"message": "贷款更新成功"})
//}

// GetLoans (完整的，无省略)
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

// UpdateLoanStatus (完整的，无省略)
func (h *DBHandler) UpdateLoanStatus(c *gin.Context) {
	id := c.Param("id")
	var payload struct {
		Status string `json:"status" binding:"required,oneof=active paid"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的状态: " + err.Error()})
		return
	}

	var query string
	var args []interface{}
	if payload.Status == "paid" {
		query = "UPDATE loans SET status = ?, repayment_date = ? WHERE id = ?"
		args = append(args, payload.Status, time.Now().Format("2006-01-02"), id)
	} else {
		query = "UPDATE loans SET status = ?, repayment_date = NULL WHERE id = ?"
		args = append(args, payload.Status, id)
	}

	res, err := h.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新贷款状态失败: " + err.Error()})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的贷款"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "贷款状态更新成功"})
}

// DeleteLoan (完整的，无省略)
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
