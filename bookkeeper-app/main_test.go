// bookkeeper-app/main_test.go
package main

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// setupTestDB 创建一个用于测试的内存数据库
func setupTestDB(t *testing.T) *sql.DB {
	// 使用内存数据库
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("无法打开内存数据库: %v", err)
	}

	// 必须手动执行初始化逻辑，因为 initializeDB 是针对文件数据库的
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	// 创建所有表
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS users ( "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "username" TEXT NOT NULL UNIQUE, "password_hash" TEXT NOT NULL, "is_admin" INTEGER NOT NULL DEFAULT 0, "must_change_password" INTEGER NOT NULL DEFAULT 0, "created_at" TEXT NOT NULL, "failed_login_attempts" INTEGER NOT NULL DEFAULT 0, "lockout_until" TEXT );`,
		`CREATE TABLE IF NOT EXISTS shared_categories ( "id" TEXT NOT NULL PRIMARY KEY, "name" TEXT NOT NULL UNIQUE, "type" TEXT NOT NULL, "icon" TEXT, "is_editable" INTEGER NOT NULL DEFAULT 1, "created_at" TEXT NOT NULL );`,
		`CREATE TABLE IF NOT EXISTS categories ( "id" TEXT NOT NULL, "user_id" INTEGER NOT NULL, "name" TEXT NOT NULL, "type" TEXT NOT NULL, "icon" TEXT, "created_at" TEXT NOT NULL, PRIMARY KEY("id", "user_id"), FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE );`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_user_name ON categories (user_id, name);`,
		`CREATE TABLE IF NOT EXISTS loans ( "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "user_id" INTEGER NOT NULL, "principal" REAL NOT NULL, "interest_rate" REAL NOT NULL, "loan_date" TEXT NOT NULL, "repayment_date" TEXT, "description" TEXT, "status" TEXT NOT NULL, "created_at" TEXT NOT NULL, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE );`,
		`CREATE TABLE IF NOT EXISTS accounts ( "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "user_id" INTEGER NOT NULL, "name" TEXT NOT NULL, "type" TEXT NOT NULL, "balance" REAL NOT NULL DEFAULT 0, "icon" TEXT, "is_primary" INTEGER NOT NULL DEFAULT 0, "created_at" TEXT NOT NULL, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE, UNIQUE(user_id, name) );`,
		`CREATE UNIQUE INDEX IF NOT EXISTS one_primary_account_per_user_idx ON accounts (user_id, is_primary) WHERE is_primary = 1;`,
		`CREATE TABLE IF NOT EXISTS transactions ( "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "user_id" INTEGER NOT NULL, "type" TEXT NOT NULL, "amount" REAL NOT NULL, "transaction_date" TEXT NOT NULL, "description" TEXT, "created_at" TEXT NOT NULL, "category_id" TEXT, "related_loan_id" INTEGER, "from_account_id" INTEGER, "to_account_id" INTEGER, "settlement_month" TEXT, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE, FOREIGN KEY(related_loan_id) REFERENCES loans(id) ON DELETE SET NULL, FOREIGN KEY(from_account_id) REFERENCES accounts(id) ON DELETE SET NULL, FOREIGN KEY(to_account_id) REFERENCES accounts(id) ON DELETE SET NULL );`,
		`CREATE UNIQUE INDEX IF NOT EXISTS one_settlement_per_month_per_user_idx ON transactions (user_id, settlement_month) WHERE settlement_month IS NOT NULL;`,
		`CREATE TABLE IF NOT EXISTS budgets ( "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "user_id" INTEGER NOT NULL, "category_id" TEXT, "amount" REAL NOT NULL, "period" TEXT NOT NULL, "created_at" TEXT NOT NULL, UNIQUE(user_id, period, category_id) );`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens ( "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "user_id" INTEGER NOT NULL, "token_hash" TEXT NOT NULL UNIQUE, "expires_at" TEXT NOT NULL, "created_at" TEXT NOT NULL, FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE );`,
		`CREATE TABLE IF NOT EXISTS login_history ( "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "user_id" INTEGER, "username_attempt" TEXT NOT NULL, "ip_address" TEXT, "user_agent" TEXT, "status" TEXT NOT NULL, "created_at" TEXT NOT NULL );`,
	}
	for _, schema := range schemas {
		if _, err := db.Exec(schema); err != nil {
			db.Close()
			t.Fatalf("创建测试数据库表失败: %v", err)
		}
	}

	// 为测试数据库也创建 admin 用户和共享分类
	seedSharedCategories(db, logger)
	seedAdminUser(db, logger)

	return db
}

// createTestUser 在测试数据库中创建一个普通用户并返回其ID和密码
func createTestUser(t *testing.T, db *sql.DB, username, password string) int64 {
	hashedPassword, err := hashPassword(password)
	if err != nil {
		t.Fatalf("哈希测试用户密码失败: %v", err)
	}
	createdAt := time.Now().Format(time.RFC3339)
	res, err := db.Exec(
		"INSERT INTO users (username, password_hash, is_admin, must_change_password, created_at) VALUES (?, ?, ?, ?, ?)",
		username, hashedPassword, 0, 0, createdAt,
	)
	if err != nil {
		t.Fatalf("创建测试用户 '%s' 失败: %v", username, err)
	}
	id, _ := res.LastInsertId()

	return id
}

// getTestAuthToken 为指定用户生成一个测试用的JWT
func getTestAuthToken(t *testing.T, userID int64, username string, isAdmin bool) string {
	jwtKey = []byte("test_secret_key_for_unit_tests")

	testUser := User{ID: userID, Username: username, IsAdmin: isAdmin}
	// 【修改】增加 duration 参数
	token, err := generateJWT(testUser, 24*time.Hour)
	if err != nil {
		t.Fatalf("生成测试token失败: %v", err)
	}
	return token
}

// performRequest 执行一个HTTP测试请求
func performRequest(r http.Handler, method, path string, body io.Reader, token ...string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, body)
	if len(token) > 0 && token[0] != "" {
		req.Header.Set("Authorization", "Bearer "+token[0])
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
