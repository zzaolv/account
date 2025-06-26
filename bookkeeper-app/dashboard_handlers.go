// bookkeeper-app/dashboard_handlers.go
package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// getTotalsForPeriod (无修改)
func getTotalsForPeriod(db *sql.DB, userID int64, year, month string) (float64, float64, error) {
	var income, expense sql.NullFloat64
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "user_id = ?")
	args = append(args, userID)
	conditions = append(conditions, "type IN ('income', 'expense', 'repayment')")

	if year != "" {
		conditions = append(conditions, "strftime('%Y', transaction_date) = ?")
		args = append(args, year)
	}
	if month != "" {
		monthFormatted := fmt.Sprintf("%02s", month)
		conditions = append(conditions, "strftime('%m', transaction_date) = ?")
		args = append(args, monthFormatted)
	}

	query := `
        SELECT 
            COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0),
            COALESCE(SUM(CASE WHEN type IN ('expense', 'repayment') THEN amount ELSE 0 END), 0)
        FROM transactions
    `
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	err := db.QueryRow(query, args...).Scan(&income, &expense)
	if err != nil {
		return 0, 0, err
	}
	return income.Float64, expense.Float64, nil
}

// GetDashboardCards (无修改)
func (h *DBHandler) GetDashboardCards(c *gin.Context) {
	userID, _ := c.Get("userID")
	logger := h.Logger.With(slog.Int64("userID", userID.(int64)))

	year := c.Query("year")
	month := c.Query("month")
	currentIncome, currentExpense, err := getTotalsForPeriod(h.DB, userID.(int64), year, month)
	if err != nil {
		logger.Error("获取当前周期数据失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取当前周期数据失败: " + err.Error()})
		return
	}
	var prevYear, prevMonth string
	if year != "" {
		y, err := strconv.Atoi(year)
		if err == nil {
			if month != "" {
				m, err := strconv.Atoi(month)
				if err == nil {
					prevTime := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
					prevYear, prevMonth = fmt.Sprintf("%d", prevTime.Year()), fmt.Sprintf("%d", prevTime.Month())
				}
			} else {
				prevYear = fmt.Sprintf("%d", y-1)
			}
		}
	}
	var prevIncome, prevExpense float64
	if prevYear != "" || prevMonth != "" {
		prevIncome, prevExpense, err = getTotalsForPeriod(h.DB, userID.(int64), prevYear, prevMonth)
		if err != nil {
			logger.Warn("获取上一周期数据失败", "error", err)
		}
	}

	var totalDeposits float64
	var accountCount int
	err = h.DB.QueryRow("SELECT COALESCE(SUM(balance), 0), COUNT(id) FROM accounts WHERE user_id = ?", userID).Scan(&totalDeposits, &accountCount)
	if err != nil {
		logger.Error("获取总存款数据失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取总存款数据失败"})
		return
	}

	var totalLoan float64
	h.DB.QueryRow("SELECT COALESCE(SUM(principal), 0) FROM loans WHERE user_id = ? AND status = 'active'", userID).Scan(&totalLoan)

	cards := []DashboardCard{
		{Title: "总收入", Value: currentIncome, PrevValue: prevIncome, Icon: "TrendingUp"},
		{Title: "总支出", Value: currentExpense, PrevValue: prevExpense, Icon: "TrendingDown"},
		{Title: "净结余", Value: currentIncome - currentExpense, PrevValue: prevIncome - prevExpense, Icon: "Scale"},
		{Title: "总存款", Value: totalDeposits, PrevValue: 0, Icon: "PiggyBank", Meta: gin.H{"account_count": accountCount}},
	}
	c.JSON(http.StatusOK, cards)
}

// GetAnalyticsCharts (无修改)
func (h *DBHandler) GetAnalyticsCharts(c *gin.Context) {
	userID, _ := c.Get("userID")
	logger := h.Logger.With(slog.Int64("userID", userID.(int64)))

	year := c.Query("year")
	month := c.Query("month")

	var response AnalyticsChartsResponse
	response.ExpenseTrend = []ChartDataPoint{}
	response.CategoryExpense = []ChartDataPoint{}

	var trendQuery strings.Builder
	var trendArgs []interface{}
	trendQuery.WriteString("SELECT ")
	if year != "" && month == "" {
		trendQuery.WriteString("strftime('%Y-%m', transaction_date) as period, COALESCE(SUM(amount), 0)")
	} else if year != "" && month != "" {
		trendQuery.WriteString("strftime('%d', transaction_date) as period, COALESCE(SUM(amount), 0)")
	} else {
		trendQuery.WriteString("strftime('%Y', transaction_date) as period, COALESCE(SUM(amount), 0)")
	}
	trendQuery.WriteString(" FROM transactions WHERE user_id = ? AND type IN ('expense', 'repayment')")
	trendArgs = append(trendArgs, userID)

	if year != "" {
		trendQuery.WriteString(" AND strftime('%Y', transaction_date) = ?")
		trendArgs = append(trendArgs, year)
	}
	if month != "" {
		monthFormatted := fmt.Sprintf("%02s", month)
		trendQuery.WriteString(" AND strftime('%m', transaction_date) = ?")
		trendArgs = append(trendArgs, monthFormatted)
	}
	trendQuery.WriteString(" GROUP BY period ORDER BY period ASC")

	rows, err := h.DB.Query(trendQuery.String(), trendArgs...)
	if err != nil {
		logger.Error("查询支出趋势失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取支出趋势数据失败"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var point ChartDataPoint
		if err := rows.Scan(&point.Name, &point.Value); err != nil {
			logger.Warn("扫描支出趋势数据失败", "error", err)
			continue
		}
		response.ExpenseTrend = append(response.ExpenseTrend, point)
	}
	rows.Close()

	var catQueryBuilder strings.Builder
	catQueryBuilder.WriteString(`
        WITH UserCategories AS (
            SELECT id, name FROM shared_categories
            UNION ALL
            SELECT id, name FROM categories WHERE user_id = ?
        )
        SELECT COALESCE(uc.name, '未分类') as category_name, COALESCE(SUM(t.amount), 0)
        FROM transactions t
        LEFT JOIN UserCategories uc ON t.category_id = uc.id
        WHERE t.user_id = ? AND t.type IN ('expense', 'repayment')
    `)
	var catArgs []interface{}
	catArgs = append(catArgs, userID, userID)

	if year != "" {
		catQueryBuilder.WriteString(" AND strftime('%Y', t.transaction_date) = ?")
		catArgs = append(catArgs, year)
	}
	if month != "" {
		monthFormatted := fmt.Sprintf("%02s", month)
		catQueryBuilder.WriteString(" AND strftime('%m', t.transaction_date) = ?")
		catArgs = append(catArgs, monthFormatted)
	}
	catQueryBuilder.WriteString(" GROUP BY category_name HAVING SUM(t.amount) > 0 ORDER BY SUM(t.amount) DESC")

	catRows, err := h.DB.Query(catQueryBuilder.String(), catArgs...)
	if err != nil {
		logger.Error("查询分类支出失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取支出分类数据失败"})
		return
	}
	defer catRows.Close()

	for catRows.Next() {
		var point ChartDataPoint
		if err := catRows.Scan(&point.Name, &point.Value); err != nil {
			logger.Warn("扫描分类支出数据失败", "error", err)
			continue
		}
		response.CategoryExpense = append(response.CategoryExpense, point)
	}

	c.JSON(http.StatusOK, response)
}

// GetSystemStats (无修改)
func (h *DBHandler) GetSystemStats(c *gin.Context) {
	var userCount, transactionCount, accountCount int64
	var dbSize int64

	h.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	h.DB.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&transactionCount)
	h.DB.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&accountCount)

	dbPath := getDBPath()
	fileInfo, err := os.Stat(dbPath)
	if err == nil {
		dbSize = fileInfo.Size()
	}

	c.JSON(http.StatusOK, gin.H{
		"user_count":        userCount,
		"transaction_count": transactionCount,
		"account_count":     accountCount,
		"db_size_bytes":     dbSize,
	})
}

// GetDashboardWidgets (【修正版】)
func (h *DBHandler) GetDashboardWidgets(c *gin.Context) {
	userID, _ := c.Get("userID")
	logger := h.Logger.With(slog.Int64("userID", userID.(int64)))

	yearStr := c.DefaultQuery("year", fmt.Sprintf("%d", time.Now().Year()))
	monthStr := c.DefaultQuery("month", fmt.Sprintf("%d", time.Now().Month()))
	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)

	var response DashboardWidgetsResponse
	response.Budgets = []DashboardBudgetSummary{}
	response.Loans = []DashboardLoanInfo{}

	// --- 预算部分逻辑 ---
	budgetPeriods := map[string]struct {
		Year  int
		Month int
	}{
		"monthly": {Year: year, Month: month},
		"yearly":  {Year: year, Month: 0}, // 年预算不需要月份
	}

	for period, dateInfo := range budgetPeriods {
		summary := DashboardBudgetSummary{Period: period}

		// 1. 获取全局预算金额
		var query string
		var args []interface{}
		if period == "monthly" {
			query = "SELECT amount FROM budgets WHERE user_id = ? AND period = ? AND year = ? AND month = ? AND category_id IS NULL"
			args = append(args, userID, period, dateInfo.Year, dateInfo.Month)
		} else { // yearly
			query = "SELECT amount FROM budgets WHERE user_id = ? AND period = ? AND year = ? AND category_id IS NULL"
			args = append(args, userID, period, dateInfo.Year)
		}

		err := h.DB.QueryRow(query, args...).Scan(&summary.Amount)
		if err == nil {
			summary.IsSet = true
		} else if err != sql.ErrNoRows {
			logger.Warn("查询全局预算金额失败", "period", period, "error", err)
		}

		// 2. 计算总支出
		var spent float64
		var spentQueryBuilder strings.Builder
		spentQueryBuilder.WriteString("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE user_id = ? AND type IN ('expense', 'repayment')")
		spentArgs := []interface{}{userID}

		if period == "monthly" {
			spentQueryBuilder.WriteString(" AND strftime('%Y', transaction_date) = ? AND strftime('%m', transaction_date) = ?")
			spentArgs = append(spentArgs, fmt.Sprintf("%d", dateInfo.Year), fmt.Sprintf("%02d", dateInfo.Month))
		} else { // yearly
			spentQueryBuilder.WriteString(" AND strftime('%Y', transaction_date) = ?")
			spentArgs = append(spentArgs, fmt.Sprintf("%d", dateInfo.Year))
		}

		h.DB.QueryRow(spentQueryBuilder.String(), spentArgs...).Scan(&spent)
		summary.Spent = spent

		if summary.Amount > 0 {
			summary.Progress = spent / summary.Amount
		}
		response.Budgets = append(response.Budgets, summary)
	}

	// --- 贷款部分逻辑保持不变 ---
	rows, err := h.DB.Query("SELECT id, description, principal, loan_date, repayment_date FROM loans WHERE user_id = ? AND status = 'active' ORDER BY loan_date DESC", userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var loanInfo DashboardLoanInfo
			var desc, repaymentDate sql.NullString
			rows.Scan(&loanInfo.ID, &desc, &loanInfo.Principal, &loanInfo.LoanDate, &repaymentDate)
			loanInfo.Description = desc.String
			if repaymentDate.Valid {
				loanInfo.RepaymentDate = &repaymentDate.String
			}
			var totalRepaid float64
			h.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE user_id = ? AND type = 'repayment' AND related_loan_id = ?", userID, loanInfo.ID).Scan(&totalRepaid)
			loanInfo.OutstandingBalance = loanInfo.Principal - totalRepaid
			if loanInfo.Principal > 0 {
				loanInfo.RepaymentAmountProgress = totalRepaid / loanInfo.Principal
			}
			response.Loans = append(response.Loans, loanInfo)
		}
	} else {
		logger.Error("查询活动贷款失败", "error", err)
	}

	c.JSON(http.StatusOK, response)
}
