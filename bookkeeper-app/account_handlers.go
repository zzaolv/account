// bookkeeper-app/account_handlers.go
package main

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	// "github.com/mattn/go-sqlite3" //不再需要
)

// GetAccounts (无修改)
func (h *DBHandler) GetAccounts(c *gin.Context) {
	userID, _ := c.Get("userID")
	rows, err := h.DB.Query("SELECT id, name, type, balance, icon, is_primary, created_at FROM accounts WHERE user_id = ? ORDER BY is_primary DESC, created_at ASC", userID)
	if err != nil {
		h.Logger.Error("获取账户列表失败", "error", err, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取账户列表失败"})
		return
	}
	defer rows.Close()
	var accounts []Account
	for rows.Next() {
		var acc Account
		var isPrimaryInt int
		if err := rows.Scan(&acc.ID, &acc.Name, &acc.Type, &acc.Balance, &acc.Icon, &isPrimaryInt, &acc.CreatedAt); err != nil {
			h.Logger.Error("扫描账户数据失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "扫描账户数据失败"})
			return
		}
		acc.IsPrimary = isPrimaryInt == 1
		accounts = append(accounts, acc)
	}
	c.JSON(http.StatusOK, accounts)
}

// CreateAccount (无修改)
func (h *DBHandler) CreateAccount(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}
	createdAt := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec("INSERT INTO accounts (user_id, name, type, balance, icon, created_at) VALUES (?, ?, ?, ?, ?, ?)", userID, req.Name, req.Type, req.Balance, req.Icon, createdAt)
	if err != nil {
		h.Logger.Error("创建账户失败", "error", err, slog.Int64("userID", userID.(int64)))
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusConflict, gin.H{"error": "该账户名称已存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建账户失败"})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "账户创建成功"})
}

// UpdateAccount (无修改)
func (h *DBHandler) UpdateAccount(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")
	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}
	res, err := h.DB.Exec("UPDATE accounts SET name = ?, icon = ? WHERE id = ? AND user_id = ?", req.Name, req.Icon, id, userID)
	if err != nil {
		h.Logger.Error("更新账户失败", "error", err, "accountID", id, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新账户失败"})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的账户"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "账户更新成功"})
}

// DeleteAccount (无修改)
func (h *DBHandler) DeleteAccount(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")
	var balance float64
	var isPrimaryInt int
	err := h.DB.QueryRow("SELECT balance, is_primary FROM accounts WHERE id = ? AND user_id = ?", id, userID).Scan(&balance, &isPrimaryInt)
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
	res, err := h.DB.Exec("DELETE FROM accounts WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		h.Logger.Error("删除账户失败", "error", err, "accountID", id, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除账户失败"})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的账户"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "账户删除成功"})
}

// SetPrimaryAccount (无修改)
func (h *DBHandler) SetPrimaryAccount(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")
	tx, err := h.DB.Begin()
	if err != nil {
		h.Logger.Error("开启事务失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "开启事务失败"})
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE accounts SET is_primary = 0 WHERE user_id = ?", userID); err != nil {
		h.Logger.Error("重置主账户失败", "error", err, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置主账户失败"})
		return
	}
	res, err := tx.Exec("UPDATE accounts SET is_primary = 1 WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		h.Logger.Error("设置主账户失败", "error", err, "accountID", id, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置主账户失败"})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的账户"})
		return
	}
	if err := tx.Commit(); err != nil {
		h.Logger.Error("提交事务失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "主账户设置成功"})
}

// TransferFunds (修改) - 仅保留用于兼容，实际逻辑在 transaction_handlers
func (h *DBHandler) TransferFunds(c *gin.Context) {
	// 实际的转账逻辑已统一到 POST /transactions 接口 (type=transfer)
	// 这个接口为了保持旧前端（如果有）的兼容性可以保留，但新前端应该直接调用 /transactions
	// 这里我们直接返回一个提示信息，鼓励使用新接口
	c.JSON(http.StatusGone, gin.H{"error": "此接口已废弃，请使用 POST /api/v1/transactions 并设置 type 为 'transfer' 进行转账。"})
}
