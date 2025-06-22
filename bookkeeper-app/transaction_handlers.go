// bookkeeper-app/transaction_handlers.go
package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateTransaction 创建一条新的交易流水
func (h *DBHandler) CreateTransaction(c *gin.Context) {
	var req CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "开启事务失败: " + err.Error()})
		return
	}

	stmt, err := tx.Prepare("INSERT INTO transactions(type, amount, transaction_date, description, category_id, related_loan_id, created_at) VALUES(?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库准备语句失败: " + err.Error()})
		return
	}
	defer stmt.Close()

	createdAt := time.Now().Format(time.RFC3339)
	_, err = stmt.Exec(req.Type, req.Amount, req.TransactionDate, req.Description, req.CategoryID, req.RelatedLoanID, createdAt)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建流水失败: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "流水记录创建成功"})
}

// GetTransactions 获取交易流水列表，支持按年月筛选
func (h *DBHandler) GetTransactions(c *gin.Context) {
	year := c.Query("year")
	month := c.Query("month")

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
        SELECT t.id, t.type, t.amount, t.transaction_date, t.description, t.related_loan_id, t.category_id, cat.name as category_name, t.created_at
        FROM transactions t
        LEFT JOIN categories cat ON t.category_id = cat.id
    `)

	var conditions []string
	var args []interface{}
	if year != "" {
		conditions = append(conditions, "strftime('%Y', t.transaction_date) = ?")
		args = append(args, year)
	}
	if month != "" {
		monthFormatted := fmt.Sprintf("%02s", month) // 保证月份是两位数
		conditions = append(conditions, "strftime('%m', t.transaction_date) = ?")
		args = append(args, monthFormatted)
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY t.transaction_date DESC, t.created_at DESC")

	rows, err := h.DB.Query(queryBuilder.String(), args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询流水失败: " + err.Error()})
		return
	}
	defer rows.Close()

	transactions := []Transaction{}
	var totalIncome, totalExpense float64
	for rows.Next() {
		var t Transaction
		var description, categoryID, categoryName sql.NullString
		var relatedLoanID sql.NullInt64
		if err := rows.Scan(&t.ID, &t.Type, &t.Amount, &t.TransactionDate, &description, &relatedLoanID, &categoryID, &categoryName, &t.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "扫描流水数据失败: " + err.Error()})
			return
		}
		t.Description = description.String
		if relatedLoanID.Valid {
			t.RelatedLoanID = &relatedLoanID.Int64
		}
		if categoryID.Valid {
			t.CategoryID = &categoryID.String
		}
		if categoryName.Valid {
			t.CategoryName = &categoryName.String
		}

		if t.Type == "income" {
			totalIncome += t.Amount
		} else if t.Type == "expense" {
			totalExpense += t.Amount
		}
		transactions = append(transactions, t)
	}

	response := GetTransactionsResponse{
		Transactions: transactions,
		Summary: FinancialSummary{
			TotalIncome:  totalIncome,
			TotalExpense: totalExpense,
			NetBalance:   totalIncome - totalExpense,
		},
	}
	c.JSON(http.StatusOK, response)
}

// DeleteTransaction 删除一条交易流水
func (h *DBHandler) DeleteTransaction(c *gin.Context) {
	id := c.Param("id")
	res, err := h.DB.Exec("DELETE FROM transactions WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除流水失败: " + err.Error()})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的流水"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "流水删除成功"})
}
