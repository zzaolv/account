// bookkeeper-app/budget_handlers.go
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mattn/go-sqlite3"
)

// CreateOrUpdateBudget (【修正版】)
func (h *DBHandler) CreateOrUpdateBudget(c *gin.Context) {
	userID, _ := c.Get("userID")
	logger := h.Logger.With(slog.Int64("userID", userID.(int64)))

	var req CreateOrUpdateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	if req.Period == "monthly" && (req.Month < 1 || req.Month > 12) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "月度预算必须提供有效的月份 (1-12)"})
		return
	}

	createdAt := time.Now().Format(time.RFC3339)

	// 使用事务确保操作的原子性
	tx, err := h.DB.Begin()
	if err != nil {
		logger.Error("开启事务失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库操作失败"})
		return
	}
	defer tx.Rollback()

	// 先尝试删除可能存在的旧预算记录，因为 ON CONFLICT 对 NULL 的处理在某些 SQLite 版本中有问题
	var deleteQuery string
	var deleteArgs []interface{}

	deleteQuery = "DELETE FROM budgets WHERE user_id = ? AND period = ? AND year = ?"
	deleteArgs = append(deleteArgs, userID, req.Period, req.Year)

	if req.Period == "monthly" {
		deleteQuery += " AND month = ?"
		deleteArgs = append(deleteArgs, req.Month)
	}

	if req.CategoryID == nil || *req.CategoryID == "" {
		deleteQuery += " AND category_id IS NULL"
	} else {
		deleteQuery += " AND category_id = ?"
		deleteArgs = append(deleteArgs, *req.CategoryID)
	}

	if _, err := tx.Exec(deleteQuery, deleteArgs...); err != nil {
		logger.Error("删除旧预算失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存预算失败"})
		return
	}

	// 插入新记录
	var insertQuery string
	var insertArgs []interface{}

	insertQuery = "INSERT INTO budgets (user_id, period, year, month, category_id, amount, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)"

	var categoryIDForInsert interface{} // 使用 interface{} 来处理 NULL
	if req.CategoryID != nil && *req.CategoryID != "" {
		categoryIDForInsert = *req.CategoryID
	} else {
		categoryIDForInsert = nil
	}

	if req.Period == "monthly" {
		insertArgs = append(insertArgs, userID, req.Period, req.Year, req.Month, categoryIDForInsert, req.Amount, createdAt)
	} else {
		insertArgs = append(insertArgs, userID, req.Period, req.Year, nil, categoryIDForInsert, req.Amount, createdAt)
	}

	_, err = tx.Exec(insertQuery, insertArgs...)
	if err != nil {
		logger.Error("创建或更新预算失败", "error", err)
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			c.JSON(http.StatusConflict, gin.H{"error": "该周期的预算已存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建或更新预算失败"})
		}
		return
	}

	if err := tx.Commit(); err != nil {
		logger.Error("提交预算事务失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交预算事务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "预算保存成功"})
}

// GetBudgets (【修正版】)
func (h *DBHandler) GetBudgets(c *gin.Context) {
	userID, _ := c.Get("userID")
	logger := h.Logger.With(slog.Int64("userID", userID.(int64)))

	yearStr := c.DefaultQuery("year", fmt.Sprintf("%d", time.Now().Year()))
	monthStr := c.DefaultQuery("month", fmt.Sprintf("%d", time.Now().Month()))

	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)

	// 1. 先查询出符合当前时间周期的所有预算定义
	query := `
        WITH UserCategories AS (
            SELECT id, name FROM shared_categories
            UNION ALL
            SELECT id, name FROM categories WHERE user_id = ?
        )
        SELECT 
            b.id, b.category_id, b.amount, b.period, b.year, b.month,
            uc.name as category_name
        FROM budgets b
        LEFT JOIN UserCategories uc ON b.category_id = uc.id
        WHERE b.user_id = ?
          AND b.year = ?
          AND (b.period = 'yearly' OR (b.period = 'monthly' AND b.month = ?))
        ORDER BY b.period, category_name;
    `
	rows, err := h.DB.Query(query, userID, userID, year, month)
	if err != nil {
		logger.Error("查询预算列表失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询预算失败"})
		return
	}
	defer rows.Close()

	var budgets []Budget
	for rows.Next() {
		var b Budget
		var categoryID, categoryName sql.NullString
		var bYear, bMonth sql.NullInt64
		if err := rows.Scan(&b.ID, &categoryID, &b.Amount, &b.Period, &bYear, &bMonth, &categoryName); err != nil {
			logger.Error("扫描预算数据失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "扫描预算数据失败"})
			return
		}
		if categoryID.Valid {
			b.CategoryID = &categoryID.String
		}
		if categoryName.Valid {
			b.CategoryName = &categoryName.String
		}
		if bYear.Valid {
			b.Year = int(bYear.Int64)
		}
		if bMonth.Valid {
			b.Month = int(bMonth.Int64)
		}

		// 2. 为每个预算单独计算其已用金额
		var spent float64
		var spentQueryBuilder strings.Builder
		spentQueryBuilder.WriteString("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE user_id = ? AND type IN ('expense', 'repayment')")

		args := []interface{}{userID}

		if b.Period == "monthly" {
			spentQueryBuilder.WriteString(" AND strftime('%Y', transaction_date) = ? AND strftime('%m', transaction_date) = ?")
			monthFormatted := fmt.Sprintf("%02d", b.Month)
			args = append(args, strconv.Itoa(b.Year), monthFormatted)
		} else if b.Period == "yearly" {
			spentQueryBuilder.WriteString(" AND strftime('%Y', transaction_date) = ?")
			args = append(args, strconv.Itoa(b.Year))
		}

		if b.CategoryID != nil {
			spentQueryBuilder.WriteString(" AND category_id = ?")
			args = append(args, *b.CategoryID)
		} else {
			// 如果是全局预算，统计所有支出，不加 category_id 条件
		}

		h.DB.QueryRow(spentQueryBuilder.String(), args...).Scan(&spent)

		b.Spent = spent
		b.Remaining = b.Amount - spent
		if b.Amount > 0 {
			b.Progress = spent / b.Amount
		} else {
			b.Progress = 0
		}

		budgets = append(budgets, b)
	}

	if err = rows.Err(); err != nil {
		logger.Error("遍历预算结果集时出错", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理预算数据失败"})
		return
	}

	c.JSON(http.StatusOK, budgets)
}

// DeleteBudget (无修改, 保持原样)
func (h *DBHandler) DeleteBudget(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")
	res, err := h.DB.Exec("DELETE FROM budgets WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		h.Logger.Error("删除预算失败", "error", err, "budgetID", id, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除预算失败"})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的预算"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "预算删除成功"})
}
