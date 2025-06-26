// bookkeeper-app/main.go
package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3" // 导入驱动
)

var jwtKey = []byte(os.Getenv("JWT_SECRET_KEY"))

func getDBPath() string {
	if path := os.Getenv("DB_PATH"); path != "" {
		return path
	}
	return "./simple_ledger.db"
}

// initializeDB 初始化数据库连接并创建表 (【最终修正版】)
func initializeDB(logger *slog.Logger) (*sql.DB, error) {
	dbPath := getDBPath()
	logger.Info("正在连接数据库", "path", dbPath)

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// === 使用事务来确保所有表结构创建的原子性 ===
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启数据库事务失败: %w", err)
	}
	defer tx.Rollback() // 如果中间出错，回滚所有操作

	// === 1. 创建所有基础表结构 (按顺序执行，并检查每一步) ===

	// 用户表
	if _, err := tx.Exec(`
    CREATE TABLE IF NOT EXISTS users (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "username" TEXT NOT NULL UNIQUE,
        "password_hash" TEXT NOT NULL,
        "is_admin" INTEGER NOT NULL DEFAULT 0,
        "must_change_password" INTEGER NOT NULL DEFAULT 0,
        "created_at" TEXT NOT NULL,
        "failed_login_attempts" INTEGER NOT NULL DEFAULT 0,
        "lockout_until" TEXT
    );`); err != nil {
		return nil, fmt.Errorf("创建 users 表失败: %w", err)
	}

	// 共享分类表
	if _, err := tx.Exec(`
    CREATE TABLE IF NOT EXISTS shared_categories (
        "id" TEXT NOT NULL PRIMARY KEY,
        "name" TEXT NOT NULL UNIQUE,
        "type" TEXT NOT NULL,
        "icon" TEXT,
        "is_editable" INTEGER NOT NULL DEFAULT 1,
        "created_at" TEXT NOT NULL
    );`); err != nil {
		return nil, fmt.Errorf("创建 shared_categories 表失败: %w", err)
	}

	// 用户私有分类表
	if _, err := tx.Exec(`
    CREATE TABLE IF NOT EXISTS categories (
        "id" TEXT NOT NULL,
        "user_id" INTEGER NOT NULL,
        "name" TEXT NOT NULL,
        "type" TEXT NOT NULL,
        "icon" TEXT,
        "created_at" TEXT NOT NULL,
        PRIMARY KEY("id", "user_id"),
        FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
    );`); err != nil {
		return nil, fmt.Errorf("创建 categories 表失败: %w", err)
	}
	if _, err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_user_name ON categories (user_id, name);`); err != nil {
		return nil, fmt.Errorf("为 categories 创建唯一索引失败: %w", err)
	}

	// 账户表
	if _, err := tx.Exec(`
    CREATE TABLE IF NOT EXISTS accounts (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "user_id" INTEGER NOT NULL,
        "name" TEXT NOT NULL,
        "type" TEXT NOT NULL,
        "balance" REAL NOT NULL DEFAULT 0,
        "icon" TEXT,
        "is_primary" INTEGER NOT NULL DEFAULT 0,
        "created_at" TEXT NOT NULL,
        FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
        UNIQUE(user_id, name)
    );`); err != nil {
		return nil, fmt.Errorf("创建 accounts 表失败: %w", err)
	}
	if _, err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS one_primary_account_per_user_idx ON accounts (user_id) WHERE is_primary = 1;`); err != nil {
		// 注意: 旧版 SQLite 不支持部分索引。如果这里出错，可以考虑移除这个索引，或者升级 SQLite。
		// 为了兼容性，我们可以先忽略这个索引的创建错误。
		logger.Warn("创建 accounts 的部分唯一索引失败，可能是 SQLite 版本过低，但不影响核心功能", "error", err)
	}

	// 借贷表
	if _, err := tx.Exec(`
    CREATE TABLE IF NOT EXISTS loans (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "user_id" INTEGER NOT NULL,
        "principal" REAL NOT NULL,
        "interest_rate" REAL NOT NULL,
        "loan_date" TEXT NOT NULL,
        "repayment_date" TEXT,
        "description" TEXT,
        "status" TEXT NOT NULL,
        "created_at" TEXT NOT NULL,
        FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
    );`); err != nil {
		return nil, fmt.Errorf("创建 loans 表失败: %w", err)
	}

	// 预算表 (【关键】)
	if _, err := tx.Exec(`
    CREATE TABLE IF NOT EXISTS budgets (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "user_id" INTEGER NOT NULL,
        "category_id" TEXT,
        "amount" REAL NOT NULL,
        "period" TEXT NOT NULL,
        "year" INTEGER,
        "month" INTEGER,
        "created_at" TEXT NOT NULL,
        FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
        UNIQUE(user_id, period, year, month, category_id)
    );`); err != nil {
		return nil, fmt.Errorf("创建 budgets 表失败: %w", err)
	}

	// 流水表 (依赖其他表，最后创建)
	if _, err := tx.Exec(`
    CREATE TABLE IF NOT EXISTS transactions (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "user_id" INTEGER NOT NULL,
        "type" TEXT NOT NULL,
        "amount" REAL NOT NULL,
        "transaction_date" TEXT NOT NULL,
        "description" TEXT,
        "created_at" TEXT NOT NULL,
        "category_id" TEXT,
        "related_loan_id" INTEGER,
        "from_account_id" INTEGER,
        "to_account_id" INTEGER,
        "settlement_month" TEXT,
        FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
        FOREIGN KEY(related_loan_id) REFERENCES loans(id) ON DELETE SET NULL,
        FOREIGN KEY(from_account_id) REFERENCES accounts(id) ON DELETE SET NULL,
        FOREIGN KEY(to_account_id) REFERENCES accounts(id) ON DELETE SET NULL
    );`); err != nil {
		return nil, fmt.Errorf("创建 transactions 表失败: %w", err)
	}

	// 登录历史表
	if _, err := tx.Exec(`
    CREATE TABLE IF NOT EXISTS login_history (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "user_id" INTEGER,
        "username_attempt" TEXT NOT NULL,
        "ip_address" TEXT,
        "user_agent" TEXT,
        "status" TEXT NOT NULL, -- 'success' or 'failure'
        "created_at" TEXT NOT NULL
    );`); err != nil {
		return nil, fmt.Errorf("创建 login_history 表失败: %w", err)
	}

	// Refresh Tokens 表
	if _, err := tx.Exec(`
    CREATE TABLE IF NOT EXISTS refresh_tokens (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "user_id" INTEGER NOT NULL,
        "token_hash" TEXT NOT NULL UNIQUE,
        "expires_at" TEXT NOT NULL,
        "created_at" TEXT NOT NULL,
        FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
    );`); err != nil {
		return nil, fmt.Errorf("创建 refresh_tokens 表失败: %w", err)
	}

	// 提交事务，完成所有表的创建
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交数据库结构创建事务失败: %w", err)
	}

	// === 2. 种子数据 (在表结构创建成功后执行) ===
	seedSharedCategories(db, logger)
	seedAdminUser(db, logger)

	logger.Info("✅ 数据库检查/初始化成功!")
	return db, nil
}

// ... (省略 hashPassword, seedSharedCategories, seedAdminUser, main 函数，它们不需要修改)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func seedSharedCategories(db *sql.DB, logger *slog.Logger) {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM shared_categories").Scan(&count)
	if count > 0 {
		return
	}
	logger.Info("共享分类为空，正在插入预设分类...")

	tx, err := db.Begin()
	if err != nil {
		logger.Error("开始填充共享分类事务失败", "error", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO shared_categories (id, name, type, icon, is_editable, created_at) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		logger.Error("准备共享分类插入语句失败", "error", err)
		return
	}
	defer stmt.Close()

	defaultCategories := getDefaultCategories()
	createdAt := time.Now().Format(time.RFC3339)

	for _, cat := range defaultCategories {
		isEditable := 1
		if cat.ID == "transfer" || cat.ID == "loan_repayment" || cat.ID == "settlement" {
			isEditable = 0
		}
		_, err := stmt.Exec(cat.ID, cat.Name, cat.Type, cat.Icon, isEditable, createdAt)
		if err != nil {
			logger.Error("插入共享分类失败", "category", cat.Name, "error", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		logger.Error("提交共享分类事务失败", "error", err)
		return
	}
	logger.Info("✅ 共享分类插入完成!")
}

func seedAdminUser(db *sql.DB, logger *slog.Logger) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&count)
	if err != nil {
		logger.Error("检查 admin 用户是否存在失败", "error", err)
		return
	}
	if count > 0 {
		return
	}

	logger.Info("数据库中未找到 admin 用户，正在创建...")
	hashedPassword, err := hashPassword("admin")
	if err != nil {
		logger.Error("为 admin 用户哈希初始密码失败", "error", err)
		return
	}
	createdAt := time.Now().Format(time.RFC3339)
	_, err = db.Exec(
		"INSERT INTO users (username, password_hash, is_admin, must_change_password, created_at) VALUES (?, ?, ?, ?, ?)",
		"admin", hashedPassword, 1, 1, createdAt,
	)
	if err != nil {
		logger.Error("插入 admin 用户失败", "error", err)
	} else {
		logger.Info("✅ 默认 admin 用户创建成功 (密码: admin)，首次登录需修改密码。")
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if len(jwtKey) == 0 {
		logger.Error("关键错误: 环境变量 JWT_SECRET_KEY 未设置。服务器无法启动。请设置一个足够长的随机字符串。")
		os.Exit(1)
	}

	db, err := initializeDB(logger)
	if err != nil {
		logger.Error("数据库初始化错误", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	handler := &DBHandler{DB: db, Logger: logger}
	router := setupRouter(handler)

	logger.Info("🚀 服务器启动于 http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		logger.Error("服务器启动失败", "error", err)
		os.Exit(1)
	}
}
