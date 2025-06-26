// bookkeeper-app/account_handlers_test.go
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGetAccounts_DataIsolation 测试数据隔离
func TestGetAccounts_DataIsolation(t *testing.T) {
	// 1. 设置
	db := setupTestDB(t)
	defer db.Close()

	handler := &DBHandler{DB: db, Logger: slog.New(slog.NewJSONHandler(io.Discard, nil))}
	router := setupRouter(handler)

	// 创建两个用户
	user1ID := createTestUser(t, db, "user1", "password123")
	user2ID := createTestUser(t, db, "user2", "password123")

	// 为 user1 创建2个账户
	db.Exec("INSERT INTO accounts (user_id, name, type, balance, icon, created_at) VALUES (?, ?, ?, ?, ?, ?)", user1ID, "User1-Checking", "card", 1000, "CreditCard", time.Now().Format(time.RFC3339))
	db.Exec("INSERT INTO accounts (user_id, name, type, balance, icon, created_at) VALUES (?, ?, ?, ?, ?, ?)", user1ID, "User1-Savings", "card", 5000, "PiggyBank", time.Now().Format(time.RFC3339))
	// 为 user2 创建1个账户
	db.Exec("INSERT INTO accounts (user_id, name, type, balance, icon, created_at) VALUES (?, ?, ?, ?, ?, ?)", user2ID, "User2-Wallet", "wechat", 200, "Wallet", time.Now().Format(time.RFC3339))

	// 获取 user1 的token
	user1Token := getTestAuthToken(t, user1ID, "user1", false)

	// 2. 执行
	w := performRequest(router, "GET", "/api/v1/accounts", nil, user1Token)

	// 3. 断言
	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody []Account
	err := json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.NoError(t, err)

	// 断言 user1 只能看到自己的2个账户
	assert.Len(t, responseBody, 2)
	assert.Equal(t, "User1-Checking", responseBody[0].Name)
	assert.Equal(t, "User1-Savings", responseBody[1].Name)
}

// TestCreateAccount_Success 测试成功创建账户
func TestCreateAccount_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	handler := &DBHandler{DB: db, Logger: slog.New(slog.NewJSONHandler(io.Discard, nil))}
	router := setupRouter(handler)

	userID := createTestUser(t, db, "testuser", "password")
	token := getTestAuthToken(t, userID, "testuser", false)

	accountPayload := CreateAccountRequest{
		Name:    "My New Test Account",
		Type:    "alipay",
		Balance: 150.50,
		Icon:    "Briefcase",
	}
	body, _ := json.Marshal(accountPayload)

	w := performRequest(router, "POST", "/api/v1/accounts", bytes.NewBuffer(body), token)

	assert.Equal(t, http.StatusCreated, w.Code)

	var responseMessage map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseMessage)
	assert.Equal(t, "账户创建成功", responseMessage["message"])

	// 验证数据库中是否真的创建了
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM accounts WHERE user_id = ? AND name = ?", userID, "My New Test Account").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestDeleteAccount_Forbidden 测试删除不属于自己的账户
func TestDeleteAccount_Forbidden(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	handler := &DBHandler{DB: db, Logger: slog.New(slog.NewJSONHandler(io.Discard, nil))}
	router := setupRouter(handler)

	user1ID := createTestUser(t, db, "user1", "password")
	user2ID := createTestUser(t, db, "user2", "password")

	// user2 创建一个账户
	res, _ := db.Exec("INSERT INTO accounts (user_id, name, type, balance, icon, created_at) VALUES (?, ?, ?, ?, ?, ?)", user2ID, "User2-Account", "card", 0, "CreditCard", time.Now().Format(time.RFC3339))
	accountID, _ := res.LastInsertId()

	// user1 尝试删除 user2 的账户
	user1Token := getTestAuthToken(t, user1ID, "user1", false)

	w := performRequest(router, "DELETE", fmt.Sprintf("/api/v1/accounts/%d", accountID), nil, user1Token)

	// 因为我们是在handler层面检查，对于不属于自己的ID，查询会失败，所以返回404 Not Found
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// 【新增测试用例】测试转账时余额不足的情况
func TestTransferFunds_InsufficientBalance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	handler := &DBHandler{DB: db, Logger: slog.New(slog.NewJSONHandler(io.Discard, nil))}
	router := setupRouter(handler)

	userID := createTestUser(t, db, "testuser", "password")
	token := getTestAuthToken(t, userID, "testuser", false)

	// 创建两个账户，一个余额100，一个为0
	fromAccountID := createTestAccount(t, db, userID, "From", 100.0)
	toAccountID := createTestAccount(t, db, userID, "To", 0.0)

	// 尝试转账 100.01
	transferPayload := TransferRequest{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        100.01,
		Date:          time.Now().Format("2006-01-02"),
		Description:   "Test transfer",
	}
	body, _ := json.Marshal(transferPayload)

	w := performRequest(router, "POST", "/api/v1/accounts/transfer", bytes.NewBuffer(body), token)

	// 断言会返回 409 Conflict，并提示余额不足
	assert.Equal(t, http.StatusConflict, w.Code)
	var errResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Contains(t, errResp["error"], "账户余额不足")

	// 验证数据库中的余额没有发生变化
	var fromBalance, toBalance float64
	db.QueryRow("SELECT balance FROM accounts WHERE id = ?", fromAccountID).Scan(&fromBalance)
	db.QueryRow("SELECT balance FROM accounts WHERE id = ?", toAccountID).Scan(&toBalance)
	assert.Equal(t, 100.0, fromBalance)
	assert.Equal(t, 0.0, toBalance)
}

// 辅助函数，用于在测试中快速创建账户
func createTestAccount(t *testing.T, db *sql.DB, userID int64, name string, balance float64) int64 {
	res, err := db.Exec(
		"INSERT INTO accounts (user_id, name, type, balance, icon, is_primary, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		userID, name, "card", balance, "Wallet", 0, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("创建测试账户 '%s' 失败: %v", name, err)
	}
	id, _ := res.LastInsertId()
	return id
}

// 【新增测试用例】测试删除流水时账户余额是否恢复
func TestDeleteTransaction_RevertsBalance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	handler := &DBHandler{DB: db, Logger: slog.New(slog.NewJSONHandler(io.Discard, nil))}
	router := setupRouter(handler)

	userID := createTestUser(t, db, "testuser", "password")
	token := getTestAuthToken(t, userID, "testuser", false)
	accountID := createTestAccount(t, db, userID, "Test Account", 1000.0)

	// 创建一笔50元的支出流水
	createReq := CreateTransactionRequest{
		Type:            "expense",
		Amount:          50.0,
		TransactionDate: time.Now().Format("2006-01-02"),
		CategoryID:      func() *string { s := "food_dining"; return &s }(),
		FromAccountID:   &accountID,
	}
	body, _ := json.Marshal(createReq)
	w := performRequest(router, "POST", "/api/v1/transactions", bytes.NewBuffer(body), token)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 确认账户余额被扣减
	var balanceAfterCreate float64
	db.QueryRow("SELECT balance FROM accounts WHERE id = ?", accountID).Scan(&balanceAfterCreate)
	assert.Equal(t, 950.0, balanceAfterCreate)

	// 获取这笔流水的ID
	var transactionID int64
	db.QueryRow("SELECT id FROM transactions WHERE user_id = ? ORDER BY id DESC LIMIT 1", userID).Scan(&transactionID)

	// 删除这笔流水
	wDelete := performRequest(router, "DELETE", fmt.Sprintf("/api/v1/transactions/%d", transactionID), nil, token)
	assert.Equal(t, http.StatusOK, wDelete.Code)

	// 确认账户余额已恢复
	var balanceAfterDelete float64
	db.QueryRow("SELECT balance FROM accounts WHERE id = ?", accountID).Scan(&balanceAfterDelete)
	assert.Equal(t, 1000.0, balanceAfterDelete, "删除流水后，账户余额应该恢复到原始值")
}
