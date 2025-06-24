// bookkeeper-app/models.go
package main

import "database/sql"

// DBHandler 结构体
type DBHandler struct {
	DB *sql.DB
}

// Category 相关结构体
type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	CreatedAt string `json:"created_at"`
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

// Transaction 相关结构体
type Transaction struct {
	ID              int64   `json:"id"`
	Type            string  `json:"type"`
	Amount          float64 `json:"amount"`
	TransactionDate string  `json:"transaction_date"`
	Description     string  `json:"description"`
	RelatedLoanID   *int64  `json:"related_loan_id,omitempty"`
	CategoryID      *string `json:"category_id,omitempty"`
	CategoryName    *string `json:"category_name,omitempty"`
	CreatedAt       string  `json:"created_at"`
	FromAccountID   *int64  `json:"from_account_id,omitempty"`
	ToAccountID     *int64  `json:"to_account_id,omitempty"`
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

// GetTransactionsResponse 及 FinancialSummary
type GetTransactionsResponse struct {
	Transactions []Transaction    `json:"transactions"`
	Summary      FinancialSummary `json:"summary"`
}
type FinancialSummary struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	NetBalance   float64 `json:"net_balance"`
}

// Loan 相关模型
type Loan struct {
	ID            int64   `json:"id"`
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

// 【新增】还清贷款的请求体
type SettleLoanRequest struct {
	FromAccountID int64  `json:"from_account_id" binding:"required"`
	RepaymentDate string `json:"repayment_date" binding:"required"`
	Description   string `json:"description"`
}

// Budget 相关模型
type Budget struct {
	ID           int64   `json:"id"`
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
type CreateOrUpdateBudgetRequest struct {
	CategoryID *string `json:"category_id"`
	Amount     float64 `json:"amount" binding:"required,gt=0"`
	Period     string  `json:"period" binding:"required,oneof=monthly yearly"`
}

// Dashboard & Analytics 相关模型
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

// Account 相关模型
type Account struct {
	ID        int64   `json:"id"`
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
