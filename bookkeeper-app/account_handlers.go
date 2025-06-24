// bookkeeper-app/account_handlers.go
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mattn/go-sqlite3"
)

// calculateNetIncomeForPreviousMonth 保持不变
func calculateNetIncomeForPreviousMonth(db *sql.DB) (float64, string, string, error) {
	now := time.Now()
	firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDayOfPreviousMonth := firstDayOfCurrentMonth.Add(-time.Second)
	year := fmt.Sprintf("%d", lastDayOfPreviousMonth.Year())
	month := fmt.Sprintf("%02d", lastDayOfPreviousMonth.Month())
	var income, expense float64
	query := `
        SELECT 
            COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0),
            COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)
        FROM transactions
        WHERE strftime('%Y', transaction_date) = ? 
          AND strftime('%m', transaction_date) = ?
          AND type IN ('income', 'expense')
    `
	err := db.QueryRow(query, year, month).Scan(&income, &expense)
	if err != nil && err != sql.ErrNoRows {
		return 0, "", "", err
	}
	return income - expense, year, month, nil
}

// ExecuteMonthlyTransfer 执行月度结算转存 (【最终修复版】)
func (h *DBHandler) ExecuteMonthlyTransfer(c *gin.Context) {
	// 1. 计算上个月的年份和月份
	now := time.Now()
	firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDayOfPreviousMonth := firstDayOfCurrentMonth.Add(-time.Second)
	year := fmt.Sprintf("%d", lastDayOfPreviousMonth.Year())
	month := fmt.Sprintf("%02d", lastDayOfPreviousMonth.Month())
	settlementMonth := fmt.Sprintf("%s-%s", year, month)

	// 2. 【幂等性检查】检查该月份的结算记录是否已存在
	var count int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE settlement_month = ?", settlementMonth).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查结算状态失败: " + err.Error()})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("%s 已经结算过了，请勿重复操作。", settlementMonth)})
		return
	}

	// 3. 查找主账户
	var primaryAccountID int64
	err = h.DB.QueryRow("SELECT id FROM accounts WHERE is_primary = 1").Scan(&primaryAccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusPreconditionFailed, gin.H{"error": "未设置主账户，无法执行月度结算"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查找主账户失败: " + err.Error()})
		}
		return
	}

	// 4. 计算上月净收入
	netIncome, _, _, err := calculateNetIncomeForPreviousMonth(h.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "计算上月收支失败: " + err.Error()})
		return
	}
	if netIncome == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("%s 收支平衡，无需转存", settlementMonth),
			"details": fmt.Sprintf("为 %s 结算，净收入为 0.00", settlementMonth),
		})
		return
	}

	// 5. 开始事务
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "开启事务失败: " + err.Error()})
		return
	}
	defer tx.Rollback()

	// 6. 更新主账户余额
	_, err = tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", netIncome, primaryAccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新主账户余额失败: " + err.Error()})
		return
	}

	// 7. 记录一条 'settlement' 类型的系统流水，并填充 settlement_month 字段
	var description string
	amount := math.Abs(netIncome)
	if netIncome > 0 {
		description = fmt.Sprintf("%s 结余，自动转入主账户", settlementMonth)
	} else {
		description = fmt.Sprintf("%s 亏空，自动从主账户扣除", settlementMonth)
	}

	_, err = tx.Exec(
		"INSERT INTO transactions (type, amount, transaction_date, description, category_id, to_account_id, created_at, settlement_month) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		"settlement",
		amount,
		now.Format("2006-01-02"),
		description,
		"settlement", // 使用 'settlement' 分类ID
		primaryAccountID,
		now.Format(time.RFC3339),
		settlementMonth, // 填充新的字段
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("%s 已经结算过了（数据库约束），请勿重复操作。", settlementMonth)})
		} else {
			log.Printf("创建月度结算流水失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建月度结算流水失败: " + err.Error()})
		}
		return
	}

	// 8. 提交事务
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "月度结算成功",
		"details": fmt.Sprintf("为 %s 结算，净收入 %.2f 已同步到主账户", settlementMonth, netIncome),
	})
}

// --- 其余函数保持不变 ---

func (h *DBHandler) GetAccounts(c *gin.Context) {
	rows, err := h.DB.Query("SELECT id, name, type, balance, icon, is_primary, created_at FROM accounts ORDER BY is_primary DESC, created_at ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取账户列表失败: " + err.Error()})
		return
	}
	defer rows.Close()
	var accounts []Account
	for rows.Next() {
		var acc Account
		var isPrimaryInt int
		if err := rows.Scan(&acc.ID, &acc.Name, &acc.Type, &acc.Balance, &acc.Icon, &isPrimaryInt, &acc.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "扫描账户数据失败: " + err.Error()})
			return
		}
		acc.IsPrimary = isPrimaryInt == 1
		accounts = append(accounts, acc)
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *DBHandler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}
	createdAt := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec("INSERT INTO accounts (name, type, balance, icon, created_at) VALUES (?, ?, ?, ?, ?)", req.Name, req.Type, req.Balance, req.Icon, createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建账户失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "账户创建成功"})
}

func (h *DBHandler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}
	res, err := h.DB.Exec("UPDATE accounts SET name = ?, icon = ? WHERE id = ?", req.Name, req.Icon, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新账户失败: " + err.Error()})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的账户"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "账户更新成功"})
}

func (h *DBHandler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	var balance float64
	var isPrimaryInt int
	err := h.DB.QueryRow("SELECT balance, is_primary FROM accounts WHERE id = ?", id).Scan(&balance, &isPrimaryInt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的账户"})
		return
	}
	if balance != 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "无法删除：账户余额不为零。请先将余额转出。"})
		return
	}
	if isPrimaryInt == 1 {
		c.JSON(http.StatusConflict, gin.H{"error": "无法删除主账户。请先设置其他账户为主账户。"})
		return
	}
	res, err := h.DB.Exec("DELETE FROM accounts WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除账户失败: " + err.Error()})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的账户"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "账户删除成功"})
}

func (h *DBHandler) SetPrimaryAccount(c *gin.Context) {
	id := c.Param("id")
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "开启事务失败: " + err.Error()})
		return
	}
	if _, err := tx.Exec("UPDATE accounts SET is_primary = 0"); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置主账户失败: " + err.Error()})
		return
	}
	res, err := tx.Exec("UPDATE accounts SET is_primary = 1 WHERE id = ?", id)
	if err != nil {
		tx.Rollback()
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			c.JSON(http.StatusConflict, gin.H{"error": "设置主账户失败，可能存在数据库约束问题。"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "设置主账户失败: " + err.Error()})
		}
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的账户"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "主账户设置成功"})
}

func (h *DBHandler) TransferFunds(c *gin.Context) {
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}
	if req.FromAccountID == req.ToAccountID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "转出和转入账户不能相同"})
		return
	}
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "开启事务失败: " + err.Error()})
		return
	}
	_, err = tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", req.Amount, req.FromAccountID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新转出账户失败: " + err.Error()})
		return
	}
	_, err = tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", req.Amount, req.ToAccountID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新转入账户失败: " + err.Error()})
		return
	}
	createdAt := time.Now().Format(time.RFC3339)
	transferCategoryID := "transfer"
	_, err = tx.Exec(
		"INSERT INTO transactions (type, amount, transaction_date, description, category_id, from_account_id, to_account_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		"transfer", req.Amount, req.Date, req.Description, transferCategoryID, req.FromAccountID, req.ToAccountID, createdAt,
	)
	if err != nil {
		tx.Rollback()
		log.Printf("创建转账流水失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建转账流水失败: " + err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "转账成功"})
}
