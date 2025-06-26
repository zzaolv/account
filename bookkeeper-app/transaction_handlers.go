// bookkeeper-app/transaction_handlers.go
package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateTransaction (已重构)
func (h *DBHandler) CreateTransaction(c *gin.Context) {
	userID, _ := c.Get("userID")
	logger := h.Logger.With(slog.Int64("userID", userID.(int64)))

	var req CreateTransactionRequest
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
	defer tx.Rollback() // 确保在出错时回滚

	// --- 核心逻辑：根据流水类型处理账户余额 ---
	switch req.Type {
	case "income":
		if req.ToAccountID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "收入流水必须指定收款账户 (to_account_id)"})
			return
		}
		if !isOwner(tx, userID.(int64), "accounts", *req.ToAccountID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作收款账户"})
			return
		}
		// 增加收款账户余额
		_, err = tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", req.Amount, *req.ToAccountID)
		if err != nil {
			logger.Error("更新收款账户余额失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新收款账户余额失败"})
			return
		}

	case "expense", "repayment":
		if req.FromAccountID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "支出或还款流水必须指定付款账户 (from_account_id)"})
			return
		}
		if !isOwner(tx, userID.(int64), "accounts", *req.FromAccountID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作付款账户"})
			return
		}
		// 检查余额是否充足
		var balance float64
		err := tx.QueryRow("SELECT balance FROM accounts WHERE id = ?", *req.FromAccountID).Scan(&balance)
		if err != nil {
			logger.Error("查询付款账户余额失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询付款账户余额失败"})
			return
		}
		if balance < req.Amount {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("账户余额不足 (当前: %.2f, 需要: %.2f)", balance, req.Amount)})
			return
		}
		// 扣减付款账户余额
		_, err = tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", req.Amount, *req.FromAccountID)
		if err != nil {
			logger.Error("更新付款账户余额失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新付款账户余额失败"})
			return
		}
		// 如果是还款，需要额外验证关联贷款的归属权
		if req.Type == "repayment" {
			if req.RelatedLoanID == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "还款流水必须指定关联贷款 (related_loan_id)"})
				return
			}
			if !isOwner(tx, userID.(int64), "loans", *req.RelatedLoanID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "无权操作关联贷款"})
				return
			}
		}

	case "transfer":
		if req.FromAccountID == nil || req.ToAccountID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "转账流水必须同时指定转出和转入账户"})
			return
		}
		// 验证两个账户都属于当前用户
		var count int
		err = tx.QueryRow("SELECT COUNT(*) FROM accounts WHERE id IN (?, ?) AND user_id = ?", req.FromAccountID, req.ToAccountID, userID).Scan(&count)
		if err != nil || count != 2 {
			c.JSON(http.StatusForbidden, gin.H{"error": "账户不存在或无权操作"})
			return
		}
		// 检查转出账户余额
		var fromBalance float64
		err = tx.QueryRow("SELECT balance FROM accounts WHERE id = ?", *req.FromAccountID).Scan(&fromBalance)
		if err != nil {
			logger.Error("查询转出账户余额失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询转出账户余额失败"})
			return
		}
		if fromBalance < req.Amount {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("转出账户余额不足 (当前: %.2f, 需要: %.2f)", fromBalance, req.Amount)})
			return
		}
		// 更新账户余额
		_, err = tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", req.Amount, *req.FromAccountID)
		if err != nil {
			logger.Error("更新转出账户余额失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新转出账户余额失败"})
			return
		}
		_, err = tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", req.Amount, *req.ToAccountID)
		if err != nil {
			logger.Error("更新转入账户余额失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新转入账户余额失败"})
			return
		}

	// settlement 类型不直接处理账户，它由月度结算功能独立处理
	case "settlement":
		// no account action needed here
		break
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的流水类型"})
		return
	}

	createdAt := time.Now().Format(time.RFC3339)
	_, err = tx.Exec(
		"INSERT INTO transactions(user_id, type, amount, transaction_date, description, category_id, related_loan_id, from_account_id, to_account_id, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userID, req.Type, req.Amount, req.TransactionDate, req.Description, req.CategoryID, req.RelatedLoanID, req.FromAccountID, req.ToAccountID, createdAt,
	)
	if err != nil {
		logger.Error("创建流水记录失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建流水记录失败"})
		return
	}

	if err := tx.Commit(); err != nil {
		logger.Error("提交事务失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "流水记录创建成功"})
}

// isOwner 是一个辅助函数，用于检查某个资源是否属于当前用户
func isOwner(tx *sql.Tx, userID int64, tableName string, resourceID int64) bool {
	var count int
	// 使用 Sprintf 时要极其小心SQL注入，这里 tableName 是由我们硬编码控制的，所以是安全的。
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id = ? AND user_id = ?", tableName)
	err := tx.QueryRow(query, resourceID, userID).Scan(&count)
	return err == nil && count > 0
}

// GetTransactions (已修复查询逻辑)
func (h *DBHandler) GetTransactions(c *gin.Context) {
	userID, _ := c.Get("userID")
	logger := h.Logger.With(slog.Int64("userID", userID.(int64)))

	year := c.Query("year")
	month := c.Query("month")

	var queryBuilder strings.Builder
	// 【修复问题三】使用子查询和 COALESCE 统一获取分类名
	queryBuilder.WriteString(`
        WITH UserCategories AS (
            SELECT id, name FROM shared_categories
            UNION ALL
            SELECT id, name FROM categories WHERE user_id = ?
        )
        SELECT 
            t.id, t.type, t.amount, t.transaction_date, t.description, 
            t.related_loan_id, t.category_id, uc.name as category_name, t.created_at,
            t.from_account_id, fa.name as from_account_name,
            t.to_account_id, ta.name as to_account_name
        FROM transactions t
        LEFT JOIN UserCategories uc ON t.category_id = uc.id
        LEFT JOIN accounts fa ON t.from_account_id = fa.id
        LEFT JOIN accounts ta ON t.to_account_id = ta.id
    `)

	var conditions []string
	var args []interface{}
	// 子查询的 userID
	args = append(args, userID)
	// 主查询的 userID
	conditions = append(conditions, "t.user_id = ?")
	args = append(args, userID)

	if year != "" {
		conditions = append(conditions, "strftime('%Y', t.transaction_date) = ?")
		args = append(args, year)
	}
	if month != "" {
		monthFormatted := fmt.Sprintf("%02s", month)
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
		logger.Error("查询流水失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询流水失败"})
		return
	}
	defer rows.Close()

	transactions := []Transaction{}
	var totalIncome, totalExpense float64
	for rows.Next() {
		var t Transaction
		var description, categoryID, categoryName, fromAccountName, toAccountName sql.NullString
		var relatedLoanID, fromAccountID, toAccountID sql.NullInt64
		if err := rows.Scan(
			&t.ID, &t.Type, &t.Amount, &t.TransactionDate, &description,
			&relatedLoanID, &categoryID, &categoryName, &t.CreatedAt,
			&fromAccountID, &fromAccountName, &toAccountID, &toAccountName,
		); err != nil {
			logger.Error("扫描流水数据失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "扫描流水数据失败"})
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
		if fromAccountID.Valid {
			t.FromAccountID = &fromAccountID.Int64
		}
		if fromAccountName.Valid {
			t.FromAccountName = &fromAccountName.String
		}
		if toAccountID.Valid {
			t.ToAccountID = &toAccountID.Int64
		}
		if toAccountName.Valid {
			t.ToAccountName = &toAccountName.String
		}

		if t.Type == "income" {
			totalIncome += t.Amount
		} else if t.Type == "expense" || t.Type == "repayment" { // 【修复问题二】将还款计入总支出
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

// DeleteTransaction (已重构)
func (h *DBHandler) DeleteTransaction(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")
	logger := h.Logger.With(slog.Int64("userID", userID.(int64)), "transactionID", id)

	tx, err := h.DB.Begin()
	if err != nil {
		logger.Error("开启事务失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "开启事务失败"})
		return
	}
	defer tx.Rollback()

	// 1. 获取要删除的流水信息
	var t Transaction
	err = tx.QueryRow(
		"SELECT type, amount, from_account_id, to_account_id FROM transactions WHERE id = ? AND user_id = ?",
		id, userID,
	).Scan(&t.Type, &t.Amount, &t.FromAccountID, &t.ToAccountID)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的流水"})
		} else {
			logger.Error("查询待删除流水失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		}
		return
	}

	// 2. 执行反向操作，恢复账户余额
	switch t.Type {
	case "income":
		if t.ToAccountID != nil {
			_, err = tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", t.Amount, t.ToAccountID)
		}
	case "expense", "repayment":
		if t.FromAccountID != nil {
			_, err = tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", t.Amount, t.FromAccountID)
		}
	case "transfer":
		if t.FromAccountID != nil && t.ToAccountID != nil {
			_, err = tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", t.Amount, t.FromAccountID)
			if err == nil {
				_, err = tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", t.Amount, t.ToAccountID)
			}
		}
	}
	if err != nil {
		logger.Error("恢复账户余额失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除流水时恢复账户余额失败"})
		return
	}

	// 3. 删除流水记录
	res, err := tx.Exec("DELETE FROM transactions WHERE id = ?", id)
	if err != nil {
		logger.Error("删除流水记录失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除流水记录失败"})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		// 理论上不会发生，因为前面已经查询过了
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的流水"})
		return
	}

	if err := tx.Commit(); err != nil {
		logger.Error("提交事务失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "流水删除成功，相关账户余额已恢复"})
}
