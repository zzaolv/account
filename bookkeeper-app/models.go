// bookkeeper-app/models.go
package main

import "database/sql"

// DBHandler (完整的，无省略)
type DBHandler struct {
	DB *sql.DB
}

// Category (完整的，无省略)
type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	CreatedAt string `json:"created_at"`
}

// CreateCategoryRequest (完整的，无省略)
type CreateCategoryRequest struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required,oneof=income expense"`
	Icon string `json:"icon" binding:"required"`
}

// UpdateCategoryRequest (完整的，无省略)
type UpdateCategoryRequest struct {
	Name string `json:"name" binding:"required"`
	Icon string `json:"icon" binding:"required"`
}

// Transaction (完整的，无省略)
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
}

// CreateTransactionRequest (完整的，无省略)
type CreateTransactionRequest struct {
	Type            string  `json:"type" binding:"required,oneof=income expense repayment"`
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	TransactionDate string  `json:"transaction_date" binding:"required"`
	Description     string  `json:"description"`
	CategoryID      *string `json:"category_id"`
	RelatedLoanID   *int64  `json:"related_loan_id"`
}

// GetTransactionsResponse (完整的，无省略)
type GetTransactionsResponse struct {
	Transactions []Transaction    `json:"transactions"`
	Summary      FinancialSummary `json:"summary"`
}

// FinancialSummary (完整的，无省略)
type FinancialSummary struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	NetBalance   float64 `json:"net_balance"`
}

// Loan (完整的，无省略)
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

// UpdateLoanRequest (完整的，无省略)
type UpdateLoanRequest struct {
	Principal float64 `json:"principal" binding:"required,gt=0"`
	// ======================= 核心修改点 (1) =======================
	// 使用指针类型 *float64 来区分 “未提供” (nil) 和 “值为0”
	InterestRate *float64 `json:"interest_rate" binding:"required,gte=0"`
	// ============================================================
	LoanDate      string  `json:"loan_date" binding:"required"`
	RepaymentDate *string `json:"repayment_date,omitempty"`
	Description   *string `json:"description,omitempty"`
}

// LoanResponse (完整的，无省略)
type LoanResponse struct {
	Loan
	TotalRepaid        float64 `json:"total_repaid"`
	OutstandingBalance float64 `json:"outstanding_balance"`
}

// Budget (完整的，无省略)
type Budget struct {
	ID           int64   `json:"id"`
	CategoryID   *string `json:"category_id"`
	Amount       float64 `json:"amount"`
	Period       string  `json:"period"` // 'monthly' 或 'yearly'
	CategoryName *string `json:"category_name,omitempty"`
	Spent        float64 `json:"spent"`
	Remaining    float64 `json:"remaining"`
	Progress     float64 `json:"progress"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
}

// CreateOrUpdateBudgetRequest (完整的，无省略)
type CreateOrUpdateBudgetRequest struct {
	CategoryID *string `json:"category_id"`
	Amount     float64 `json:"amount" binding:"required,gt=0"`
	Period     string  `json:"period" binding:"required,oneof=monthly yearly"`
}

// DashboardCard (完整的，无省略)
type DashboardCard struct {
	Title     string  `json:"title"`
	Value     float64 `json:"value"`
	PrevValue float64 `json:"prev_value"`
	Icon      string  `json:"icon"`
}

// ChartDataPoint (完整的，无省略)
type ChartDataPoint struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// AnalyticsChartsResponse (完整的，无省略)
type AnalyticsChartsResponse struct {
	ExpenseTrend    []ChartDataPoint `json:"expense_trend"`
	CategoryExpense []ChartDataPoint `json:"category_expense"`
}

// DashboardBudgetSummary (完整的，无省略)
type DashboardBudgetSummary struct {
	Period   string  `json:"period"`
	Amount   float64 `json:"amount"`
	Spent    float64 `json:"spent"`
	Progress float64 `json:"progress"`
	IsSet    bool    `json:"is_set"`
}

// DashboardLoanInfo (完整的，无省略)
type DashboardLoanInfo struct {
	ID                      int64   `json:"id"`
	Description             string  `json:"description"`
	OutstandingBalance      float64 `json:"outstanding_balance"`
	Principal               float64 `json:"principal"`
	RepaymentAmountProgress float64 `json:"repayment_amount_progress"`
	LoanDate                string  `json:"loan_date"`
	RepaymentDate           *string `json:"repayment_date,omitempty"`
}

// DashboardWidgetsResponse (完整的，无省略)
type DashboardWidgetsResponse struct {
	Budgets []DashboardBudgetSummary `json:"budgets"`
	Loans   []DashboardLoanInfo      `json:"loans"`
}
