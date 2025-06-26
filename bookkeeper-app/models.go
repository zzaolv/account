// bookkeeper-app/models.go
package main

import (
	"database/sql"
	"log/slog"

	"github.com/golang-jwt/jwt/v5"
)

// DBHandler 结构体，新增 Logger
type DBHandler struct {
	DB     *sql.DB
	Logger *slog.Logger
}

// === 新增：用户和认证模型 ===

// User 数据库中的用户模型
type User struct {
	ID                  int64  `json:"id"`
	Username            string `json:"username"`
	PasswordHash        string `json:"-"` // 不应在JSON中暴露
	IsAdmin             bool   `json:"is_admin"`
	MustChangePassword  bool   `json:"must_change_password"`
	CreatedAt           string `json:"created_at"`
	FailedLoginAttempts int    `json:"-"`
	LockoutUntil        string `json:"-"`
}

// LoginRequest 登录请求体
type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"rememberMe"`
}

// RegisterRequest 注册请求体 (管理员使用)
type RegisterRequest struct {
	Username string `json:"username" binding:"required,alphanum,min=4,max=20"`
	Password string `json:"password" binding:"required,min=6,max=50"`
}

// UpdatePasswordRequest 修改密码请求体
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password"` // 首次强制修改时可为空
	NewPassword string `json:"new_password" binding:"required,min=6,max=50"`
}

// Claims JWT 中携带的数据
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// === 更新现有模型 ===

// Category 相关结构体，新增 UserID
type Category struct {
	ID         string `json:"id"`
	UserID     int64  `json:"-"` // 不暴露
	Name       string `json:"name"`
	Type       string `json:"type"`
	Icon       string `json:"icon"`
	CreatedAt  string `json:"created_at"`
	IsShared   bool   `json:"is_shared"`
	IsEditable bool   `json:"is_editable"`
}
type CreateCategoryRequest struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required,oneof=income expense internal"`
	Icon string `json:"icon" binding:"required"`
}
type UpdateCategoryRequest struct {
	Name string `json:"name" binding:"required"`
	Icon string `json:"icon" binding:"required"`
}

// Transaction 相关结构体 (已重构)
type Transaction struct {
	ID              int64   `json:"id"`
	UserID          int64   `json:"-"`
	Type            string  `json:"type"`
	Amount          float64 `json:"amount"`
	TransactionDate string  `json:"transaction_date"`
	Description     string  `json:"description"`
	RelatedLoanID   *int64  `json:"related_loan_id,omitempty"`
	CategoryID      *string `json:"category_id,omitempty"`
	CategoryName    *string `json:"category_name,omitempty"`
	CreatedAt       string  `json:"created_at"`
	FromAccountID   *int64  `json:"from_account_id,omitempty"`
	FromAccountName *string `json:"from_account_name,omitempty"`
	ToAccountID     *int64  `json:"to_account_id,omitempty"`
	ToAccountName   *string `json:"to_account_name,omitempty"`
}
type CreateTransactionRequest struct {
	Type            string  `json:"type" binding:"required,oneof=income expense repayment transfer settlement"`
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	TransactionDate string  `json:"transaction_date" binding:"required"`
	Description     string  `json:"description"`
	CategoryID      *string `json:"category_id"`
	RelatedLoanID   *int64  `json:"related_loan_id"`
	FromAccountID   *int64  `json:"from_account_id"`
	ToAccountID     *int64  `json:"to_account_id"`
}
type GetTransactionsResponse struct {
	Transactions []Transaction    `json:"transactions"`
	Summary      FinancialSummary `json:"summary"`
}
type FinancialSummary struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	NetBalance   float64 `json:"net_balance"`
}

// Loan 相关模型，新增 UserID
type Loan struct {
	ID            int64   `json:"id"`
	UserID        int64   `json:"-"`
	Principal     float64 `json:"principal"`
	InterestRate  float64 `json:"interest_rate"`
	LoanDate      string  `json:"loan_date"`
	RepaymentDate *string `json:"repayment_date,omitempty"`
	Description   *string `json:"description,omitempty"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
}
type UpdateLoanRequest struct {
	Principal     float64  `json:"principal" binding:"required,gt=0"`
	InterestRate  *float64 `json:"interest_rate" binding:"required,gte=0"`
	LoanDate      string   `json:"loan_date" binding:"required"`
	RepaymentDate *string  `json:"repayment_date,omitempty"`
	Description   *string  `json:"description,omitempty"`
}
type LoanResponse struct {
	Loan
	TotalRepaid        float64 `json:"total_repaid"`
	OutstandingBalance float64 `json:"outstanding_balance"`
}
type SettleLoanRequest struct {
	FromAccountID int64  `json:"from_account_id" binding:"required"`
	RepaymentDate string `json:"repayment_date" binding:"required"`
	Description   string `json:"description"`
}

// Budget 相关模型，新增 UserID
type Budget struct {
	ID           int64   `json:"id"`
	UserID       int64   `json:"-"`
	CategoryID   *string `json:"category_id"`
	Amount       float64 `json:"amount"`
	Period       string  `json:"period"`
	CategoryName *string `json:"category_name,omitempty"`
	Spent        float64 `json:"spent"`
	Remaining    float64 `json:"remaining"`
	Progress     float64 `json:"progress"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
}

// 【修改】修正 CreateOrUpdateBudgetRequest 结构体
type CreateOrUpdateBudgetRequest struct {
	CategoryID *string `json:"category_id"`
	Amount     float64 `json:"amount" binding:"required,gt=0"`
	Period     string  `json:"period" binding:"required,oneof=monthly yearly"`
	Year       int     `json:"year"`  // 对于年度预算是必须的
	Month      int     `json:"month"` // 对于月度预算是必须的
}

// Account 相关模型，新增 UserID
type Account struct {
	ID        int64   `json:"id"`
	UserID    int64   `json:"-"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Balance   float64 `json:"balance"`
	Icon      string  `json:"icon"`
	IsPrimary bool    `json:"is_primary"`
	CreatedAt string  `json:"created_at"`
}
type CreateAccountRequest struct {
	Name    string  `json:"name" binding:"required"`
	Type    string  `json:"type" binding:"required,oneof=wechat alipay card other"`
	Balance float64 `json:"balance" binding:"gte=0"`
	Icon    string  `json:"icon"`
}
type UpdateAccountRequest struct {
	Name string `json:"name" binding:"required"`
	Icon string `json:"icon"`
}
type TransferRequest struct {
	FromAccountID int64   `json:"from_account_id" binding:"required"`
	ToAccountID   int64   `json:"to_account_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Date          string  `json:"date" binding:"required"`
	Description   string  `json:"description"`
}

// Dashboard & Analytics 相关模型 (这些是聚合数据，不需要 UserID)
type DashboardCard struct {
	Title     string  `json:"title"`
	Value     float64 `json:"value"`
	PrevValue float64 `json:"prev_value"`
	Icon      string  `json:"icon"`
	Meta      any     `json:"meta,omitempty"`
}
type ChartDataPoint struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}
type AnalyticsChartsResponse struct {
	ExpenseTrend    []ChartDataPoint `json:"expense_trend"`
	CategoryExpense []ChartDataPoint `json:"category_expense"`
}
type DashboardBudgetSummary struct {
	Period   string  `json:"period"`
	Amount   float64 `json:"amount"`
	Spent    float64 `json:"spent"`
	Progress float64 `json:"progress"`
	IsSet    bool    `json:"is_set"`
}
type DashboardLoanInfo struct {
	ID                      int64   `json:"id"`
	Description             string  `json:"description"`
	OutstandingBalance      float64 `json:"outstanding_balance"`
	Principal               float64 `json:"principal"`
	RepaymentAmountProgress float64 `json:"repayment_amount_progress"`
	LoanDate                string  `json:"loan_date"`
	RepaymentDate           *string `json:"repayment_date,omitempty"`
}
type DashboardWidgetsResponse struct {
	Budgets []DashboardBudgetSummary `json:"budgets"`
	Loans   []DashboardLoanInfo      `json:"loans"`
}

// getDefaultCategories 抽离出来，方便复用
func getDefaultCategories() []Category {
	return []Category{
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
}
