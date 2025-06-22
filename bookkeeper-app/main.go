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
const dbFile = "./simple_ledger.db"

// initializeDB 初始化数据库连接并创建表
func initializeDB() (*sql.DB, error) {
	// 【重要】在连接字符串中添加 `_foreign_keys=on` 以强制启用外键约束
	db, err := sql.Open("sqlite3", dbFile+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 【重大修改】更新所有表的结构以支持新的分类ID和图标，并增强外键约束
	createTablesSQL := `
    CREATE TABLE IF NOT EXISTS categories (
        "id" TEXT NOT NULL PRIMARY KEY,
        "name" TEXT NOT NULL UNIQUE,
        "type" TEXT NOT NULL,
        "icon" TEXT,
        "created_at" TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS loans (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "principal" REAL NOT NULL,
        "interest_rate" REAL NOT NULL,
        "loan_date" TEXT NOT NULL,
        "repayment_date" TEXT,
        "description" TEXT,
        "status" TEXT NOT NULL,
        "created_at" TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS transactions (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "type" TEXT NOT NULL,
        "amount" REAL NOT NULL,
        "transaction_date" TEXT NOT NULL,
        "description" TEXT,
        "related_loan_id" INTEGER,
        "category_id" TEXT, -- 修改: 外键类型与 categories.id 保持一致
        "created_at" TEXT NOT NULL,
        FOREIGN KEY(related_loan_id) REFERENCES loans(id) ON DELETE SET NULL,
        FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE RESTRICT -- 修改: 当分类被使用时，限制删除
    );

    CREATE TABLE IF NOT EXISTS budgets (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "category_id" TEXT, -- 修改: 外键类型与 categories.id 保持一致
        "amount" REAL NOT NULL,
        "period" TEXT NOT NULL,
        "created_at" TEXT NOT NULL,
        FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE, -- 如果分类被删除，关联的预算也删除
        UNIQUE(period, category_id)
    );
    `
	if _, err := db.Exec(createTablesSQL); err != nil {
		return nil, fmt.Errorf("创建表结构失败: %w", err)
	}

	// 【新增】调用函数来插入预设分类
	seedCategories(db)

	fmt.Println("✅ 数据库检查/初始化成功!")
	return db, nil
}

// seedCategories 插入预设分类数据，仅当表为空时执行
func seedCategories(db *sql.DB) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		log.Printf("检查分类数量失败: %v", err)
		return
	}
	if count > 0 {
		return // 如果已经有分类了，就直接返回
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

	// 你提供的预设分类列表
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
	err = tx.Commit()
	if err != nil {
		log.Printf("提交预设分类事务失败: %v", err)
		return
	}
	fmt.Println("✅ 预设分类插入完成!")
}

// main 是应用程序的入口点
func main() {
	db, err := initializeDB()
	if err != nil {
		log.Fatalf("数据库初始化错误: %v", err)
	}
	defer db.Close()

	// 创建一个包含数据库连接的处理器实例
	handler := &DBHandler{DB: db}

	// 设置路由
	router := setupRouter(handler)

	fmt.Println("🚀 服务器启动于 http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
