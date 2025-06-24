// bookkeeper-app/main.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3" // 导入驱动
)

// dbFile 定义数据库文件路径
// const dbFile = "./simple_ledger.db"
// 为了在 Docker 中使用持久化存储，修改为相对路径
const dbFile = "/data/simple_ledger.db"

// initializeDB 初始化数据库连接并创建表
func initializeDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFile+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	baseTables := []string{
		`CREATE TABLE IF NOT EXISTS categories (
            "id" TEXT NOT NULL PRIMARY KEY,
            "name" TEXT NOT NULL UNIQUE,
            "type" TEXT NOT NULL,
            "icon" TEXT,
            "created_at" TEXT NOT NULL
        );`,
		`CREATE TABLE IF NOT EXISTS loans (
            "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
            "principal" REAL NOT NULL,
            "interest_rate" REAL NOT NULL,
            "loan_date" TEXT NOT NULL,
            "repayment_date" TEXT,
            "description" TEXT,
            "status" TEXT NOT NULL,
            "created_at" TEXT NOT NULL
        );`,
		`CREATE TABLE IF NOT EXISTS accounts (
            "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
            "name" TEXT NOT NULL,
            "type" TEXT NOT NULL,
            "balance" REAL NOT NULL DEFAULT 0,
            "icon" TEXT,
            "is_primary" INTEGER NOT NULL DEFAULT 0,
            "created_at" TEXT NOT NULL
        );`,
		`CREATE UNIQUE INDEX IF NOT EXISTS one_primary_account_idx ON accounts (is_primary) WHERE is_primary = 1;`,
	}
	for _, sql := range baseTables {
		if _, err := db.Exec(sql); err != nil {
			return nil, fmt.Errorf("创建基础表失败: %w\nSQL: %s", err, sql)
		}
	}

	transactionsTableSQL := `
    CREATE TABLE IF NOT EXISTS transactions (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "type" TEXT NOT NULL,
        "amount" REAL NOT NULL,
        "transaction_date" TEXT NOT NULL,
        "description" TEXT,
        "created_at" TEXT NOT NULL,
        "category_id" TEXT,
        "related_loan_id" INTEGER,
        FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE RESTRICT,
        FOREIGN KEY(related_loan_id) REFERENCES loans(id) ON DELETE SET NULL
    );`
	if _, err := db.Exec(transactionsTableSQL); err != nil {
		return nil, fmt.Errorf("创建 transactions 表失败: %w", err)
	}

	// 使用 ALTER TABLE 安全地添加列
	if !isColumnExists(db, "transactions", "from_account_id") {
		alterSQL := `ALTER TABLE transactions ADD COLUMN from_account_id INTEGER REFERENCES accounts(id) ON DELETE SET NULL;`
		if _, err := db.Exec(alterSQL); err != nil {
			return nil, fmt.Errorf("为 transactions 表添加 from_account_id 列失败: %w", err)
		}
	}
	if !isColumnExists(db, "transactions", "to_account_id") {
		alterSQL := `ALTER TABLE transactions ADD COLUMN to_account_id INTEGER REFERENCES accounts(id) ON DELETE SET NULL;`
		if _, err := db.Exec(alterSQL); err != nil {
			return nil, fmt.Errorf("为 transactions 表添加 to_account_id 列失败: %w", err)
		}
	}
	// 【新增】为月度结算添加专用字段和唯一索引
	if !isColumnExists(db, "transactions", "settlement_month") {
		alterSQL := `ALTER TABLE transactions ADD COLUMN settlement_month TEXT;`
		if _, err := db.Exec(alterSQL); err != nil {
			return nil, fmt.Errorf("为 transactions 表添加 settlement_month 列失败: %w", err)
		}
	}
	// 创建唯一索引以确保幂等性
	uniqueSettlementIndexSQL := `CREATE UNIQUE INDEX IF NOT EXISTS one_settlement_per_month_idx ON transactions (settlement_month) WHERE settlement_month IS NOT NULL;`
	if _, err := db.Exec(uniqueSettlementIndexSQL); err != nil {
		return nil, fmt.Errorf("为 settlement_month 创建唯一索引失败: %w", err)
	}

	budgetsTableSQL := `
    CREATE TABLE IF NOT EXISTS budgets (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "category_id" TEXT,
        "amount" REAL NOT NULL,
        "period" TEXT NOT NULL,
        "created_at" TEXT NOT NULL,
        FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE,
        UNIQUE(period, category_id)
    );`
	if _, err := db.Exec(budgetsTableSQL); err != nil {
		return nil, fmt.Errorf("创建 budgets 表失败: %w", err)
	}

	seedCategories(db)
	fmt.Println("✅ 数据库检查/初始化成功!")
	return db, nil
}

func isColumnExists(db *sql.DB, tableName, columnName string) bool {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		log.Printf("检查列存在性失败 (PRAGMA): %v", err)
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typeName string
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typeName, new(int), &dfltValue, &pk); err != nil {
			log.Printf("检查列存在性失败 (scan): %v", err)
			return false
		}
		if name == columnName {
			return true
		}
	}
	return false
}

func seedCategories(db *sql.DB) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		log.Printf("检查分类数量失败: %v", err)
		return
	}
	if count > 0 {
		return
	}
	fmt.Println("数据库为空，正在插入预设分类...")
	tx, err := db.Begin()
	if err != nil {
		log.Printf("开始事务失败: %v", err)
		return
	}
	stmt, err := tx.Prepare("INSERT INTO categories (id, name, type, icon, created_at) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		log.Printf("准备预设分类语句失败: %v", err)
		tx.Rollback()
		return
	}
	defer stmt.Close()

	defaultCategories := []Category{
		{ID: "salary", Name: "工资", Type: "income", Icon: "Landmark"},
		{ID: "investments", Name: "投资", Type: "income", Icon: "TrendingUp"},
		{ID: "freelance", Name: "兼职", Type: "income", Icon: "Briefcase"},
		{ID: "rent_mortgage", Name: "房租房贷", Type: "expense", Icon: "Home"},
		{ID: "food_dining", Name: "餐饮", Type: "expense", Icon: "Utensils"},
		{ID: "transportation", Name: "交通", Type: "expense", Icon: "Car"},
		{ID: "shopping", Name: "购物", Type: "expense", Icon: "ShoppingBag"},
		{ID: "utilities", Name: "生活缴费", Type: "expense", Icon: "Zap"},
		{ID: "entertainment", Name: "娱乐", Type: "expense", Icon: "Film"},
		{ID: "health_wellness", Name: "健康", Type: "expense", Icon: "HeartPulse"},
		{ID: "loan_repayment", Name: "还贷", Type: "expense", Icon: "ReceiptText"},
		{ID: "interest_expense", Name: "利息支出", Type: "expense", Icon: "Percent"},
		{ID: "other", Name: "其他", Type: "expense", Icon: "Archive"},
		{ID: "transfer", Name: "账户互转", Type: "internal", Icon: "ArrowRightLeft"},
		{ID: "settlement", Name: "月度结算", Type: "internal", Icon: "BookCheck"},
	}

	createdAt := time.Now().Format(time.RFC3339)
	for _, cat := range defaultCategories {
		_, err := stmt.Exec(cat.ID, cat.Name, cat.Type, cat.Icon, createdAt)
		if err != nil {
			log.Printf("插入预设分类 '%s' 失败: %v", cat.Name, err)
			tx.Rollback()
			return
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("提交预设分类事务失败: %v", err)
		return
	}
	fmt.Println("✅ 预设分类插入完成!")
}

func main() {
	db, err := initializeDB()
	if err != nil {
		log.Fatalf("数据库初始化错误: %v", err)
	}
	defer db.Close()
	handler := &DBHandler{DB: db}
	router := setupRouter(handler)
	fmt.Println("🚀 服务器启动于 http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
