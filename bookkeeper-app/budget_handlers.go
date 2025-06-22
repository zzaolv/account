// bookkeeper-app/budget_handlers.go
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mattn/go-sqlite3"
)

// CreateOrUpdateBudget (完整的，无省略)
func (h *DBHandler) CreateOrUpdateBudget(c *gin.Context) {
	var req CreateOrUpdateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	var query string
	var args []interface{}
	if req.CategoryID == nil || *req.CategoryID == "" {
		query = "INSERT INTO budgets (period, category_id, amount, created_at) VALUES (?, NULL, ?, ?) ON CONFLICT(period, category_id) DO UPDATE SET amount=excluded.amount"
		args = append(args, req.Period, req.Amount, time.Now().Format(time.RFC3339))
	} else {
		query = "INSERT INTO budgets (period, category_id, amount, created_at) VALUES (?, ?, ?, ?) ON CONFLICT(period, category_id) DO UPDATE SET amount=excluded.amount"
		args = append(args, req.Period, *req.CategoryID, req.Amount, time.Now().Format(time.RFC3339))
	}

	_, err := h.DB.Exec(query, args...)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			c.JSON(http.StatusConflict, gin.H{"error": "该周期的预算已存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建或更新预算失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "预算保存成功"})
}

// GetBudgets 【重大修改】支持按年月筛选，并返回对应的年月信息
func (h *DBHandler) GetBudgets(c *gin.Context) {
	// 从查询参数获取年份和月份，如果未提供，则使用当前年月
	yearStr := c.DefaultQuery("year", fmt.Sprintf("%d", time.Now().Year()))
	monthStr := c.DefaultQuery("month", fmt.Sprintf("%d", time.Now().Month()))

	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)

	rows, err := h.DB.Query(`
        SELECT b.id, b.category_id, b.amount, b.period, c.name as category_name
        FROM budgets b
        LEFT JOIN categories c ON b.category_id = c.id
        ORDER BY b.period, category_name
    `)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询预算失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var budgets []Budget
	for rows.Next() {
		var b Budget
		var categoryID, categoryName sql.NullString
		if err := rows.Scan(&b.ID, &categoryID, &b.Amount, &b.Period, &categoryName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "扫描预算数据失败"})
			return
		}
		if categoryID.Valid {
			b.CategoryID = &categoryID.String
		}
		if categoryName.Valid {
			b.CategoryName = &categoryName.String
		}

		// 根据预算周期和传入的筛选参数，计算已用金额
		var spent float64
		var queryBuilder strings.Builder
		queryBuilder.WriteString("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'expense'")

		var args []interface{}
		var currentYear, currentMonth int
		if b.Period == "monthly" {
			queryBuilder.WriteString(" AND strftime('%Y', transaction_date) = ? AND strftime('%m', transaction_date) = ?")
			monthFormatted := fmt.Sprintf("%02d", month)
			args = append(args, yearStr, monthFormatted)
			currentYear = year
			currentMonth = month
		} else if b.Period == "yearly" {
			queryBuilder.WriteString(" AND strftime('%Y', transaction_date) = ?")
			args = append(args, yearStr)
			currentYear = year
			currentMonth = 0 // 0 代表全年
		}

		if b.CategoryID != nil {
			queryBuilder.WriteString(" AND category_id = ?")
			args = append(args, *b.CategoryID)
		}

		h.DB.QueryRow(queryBuilder.String(), args...).Scan(&spent)

		b.Spent = spent
		b.Remaining = b.Amount - spent
		if b.Amount > 0 {
			b.Progress = spent / b.Amount
		}
		b.Year = currentYear
		b.Month = currentMonth

		budgets = append(budgets, b)
	}
	c.JSON(http.StatusOK, budgets)
}

// DeleteBudget (完整的，无省略)
func (h *DBHandler) DeleteBudget(c *gin.Context) {
	id := c.Param("id")
	res, err := h.DB.Exec("DELETE FROM budgets WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除预算失败: " + err.Error()})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的预算"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "预算删除成功"})
}
