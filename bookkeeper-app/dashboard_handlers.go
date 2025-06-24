// bookkeeper-app/dashboard_handlers.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// getTotalsForPeriod 计算指定周期的总收入和总支出。
// 【重要修改】现在会排除所有内部类型 ('transfer', 'settlement') 的流水。
func getTotalsForPeriod(db *sql.DB, year, month string) (float64, float64, error) {
	var income, expense float64
	var conditions []string
	var args []interface{}

	// 只计算真正的外部收支
	conditions = append(conditions, "type IN ('income', 'expense')")

	if year != "" {
		conditions = append(conditions, "strftime('%Y', transaction_date) = ?")
		args = append(args, year)
	}
	if month != "" {
		monthFormatted := fmt.Sprintf("%02s", month)
		conditions = append(conditions, "strftime('%m', transaction_date) = ?")
		args = append(args, monthFormatted)
	}

	query := "SELECT type, SUM(amount) FROM transactions"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " GROUP BY type"

	rows, err := db.Query(query, args...)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var transType string
		var total float64
		if err := rows.Scan(&transType, &total); err != nil {
			log.Printf("扫描收支总额数据失败: %v", err)
			continue
		}
		if transType == "income" {
			income = total
		} else if transType == "expense" {
			expense = total
		}
	}
	return income, expense, nil
}

func (h *DBHandler) GetDashboardCards(c *gin.Context) {
	year := c.Query("year")
	month := c.Query("month")
	currentIncome, currentExpense, err := getTotalsForPeriod(h.DB, year, month)
	if err != nil {
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
		prevIncome, prevExpense, err = getTotalsForPeriod(h.DB, prevYear, prevMonth)
		if err != nil {
			log.Printf("获取上一周期数据失败: %v", err)
		}
	}

	var totalDeposits float64
	var accountCount int
	err = h.DB.QueryRow("SELECT COALESCE(SUM(balance), 0), COUNT(id) FROM accounts").Scan(&totalDeposits, &accountCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取总存款数据失败: " + err.Error()})
		return
	}

	var totalLoan float64
	h.DB.QueryRow("SELECT COALESCE(SUM(principal), 0) FROM loans WHERE status = 'active'").Scan(&totalLoan)

	cards := []DashboardCard{
		{Title: "总收入", Value: currentIncome, PrevValue: prevIncome, Icon: "TrendingUp"},
		{Title: "总支出", Value: currentExpense, PrevValue: prevExpense, Icon: "TrendingDown"},
		{Title: "净结余", Value: currentIncome - currentExpense, PrevValue: prevIncome - prevExpense, Icon: "Scale"},
		{Title: "总存款", Value: totalDeposits, PrevValue: 0, Icon: "PiggyBank", Meta: gin.H{"account_count": accountCount}},
	}
	c.JSON(http.StatusOK, cards)
}

func (h *DBHandler) GetAnalyticsCharts(c *gin.Context) {
	year := c.Query("year")
	month := c.Query("month")

	var response AnalyticsChartsResponse
	response.ExpenseTrend = []ChartDataPoint{}
	response.CategoryExpense = []ChartDataPoint{}

	var trendQuery strings.Builder
	var trendArgs []interface{}
	trendQuery.WriteString("SELECT ")
	if year != "" && month == "" {
		trendQuery.WriteString("strftime('%Y-%m', transaction_date) as period, SUM(amount)")
	} else if year != "" && month != "" {
		trendQuery.WriteString("strftime('%d', transaction_date) as period, SUM(amount)")
	} else {
		trendQuery.WriteString("strftime('%Y', transaction_date) as period, SUM(amount)")
	}
	trendQuery.WriteString(" FROM transactions WHERE type = 'expense'")
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
		log.Printf("查询支出趋势失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取支出趋势数据失败"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var point ChartDataPoint
		if err := rows.Scan(&point.Name, &point.Value); err != nil {
			log.Printf("扫描支出趋势数据失败: %v", err)
			continue
		}
		response.ExpenseTrend = append(response.ExpenseTrend, point)
	}
	rows.Close()

	var catQueryBuilder strings.Builder
	catQueryBuilder.WriteString(`
        SELECT COALESCE(c.name, '未分类') as category_name, SUM(t.amount)
        FROM transactions t
        LEFT JOIN categories c ON t.category_id = c.id
        WHERE t.type = 'expense'
    `)
	var catArgs []interface{}
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
		log.Printf("查询分类支出失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取支出分类数据失败"})
		return
	}
	defer catRows.Close()

	for catRows.Next() {
		var point ChartDataPoint
		if err := catRows.Scan(&point.Name, &point.Value); err != nil {
			log.Printf("扫描分类支出数据失败: %v", err)
			continue
		}
		response.CategoryExpense = append(response.CategoryExpense, point)
	}

	c.JSON(http.StatusOK, response)
}

func (h *DBHandler) GetDashboardWidgets(c *gin.Context) {
	yearStr := c.Query("year")
	monthStr := c.Query("month")

	var response DashboardWidgetsResponse
	response.Budgets = []DashboardBudgetSummary{}
	response.Loans = []DashboardLoanInfo{}

	budgetPeriods := []string{"monthly", "yearly"}
	for _, period := range budgetPeriods {
		var summary DashboardBudgetSummary
		summary.Period = period
		err := h.DB.QueryRow("SELECT amount FROM budgets WHERE period = ? AND category_id IS NULL", period).Scan(&summary.Amount)
		summary.IsSet = err == nil
		if err != nil && err != sql.ErrNoRows {
			log.Printf("查询预算金额失败: %v", err)
		}

		var spent float64
		var timeCondition strings.Builder
		timeCondition.WriteString("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'expense'")
		var args []interface{}

		now := time.Now()
		targetYear := now.Year()
		if y, err := strconv.Atoi(yearStr); err == nil && yearStr != "" {
			targetYear = y
		}

		if period == "monthly" {
			targetMonth := now.Month()
			if m, err := strconv.Atoi(monthStr); err == nil && monthStr != "" {
				targetMonth = time.Month(m)
			}
			timeCondition.WriteString(" AND strftime('%Y', transaction_date) = ? AND strftime('%m', transaction_date) = ?")
			args = append(args, fmt.Sprintf("%d", targetYear), fmt.Sprintf("%02d", targetMonth))
		} else {
			timeCondition.WriteString(" AND strftime('%Y', transaction_date) = ?")
			args = append(args, fmt.Sprintf("%d", targetYear))
		}

		h.DB.QueryRow(timeCondition.String(), args...).Scan(&spent)
		summary.Spent = spent
		if summary.Amount > 0 {
			summary.Progress = spent / summary.Amount
		}
		response.Budgets = append(response.Budgets, summary)
	}

	rows, err := h.DB.Query("SELECT id, description, principal, loan_date, repayment_date FROM loans WHERE status = 'active' ORDER BY loan_date DESC")
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
			h.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'repayment' AND related_loan_id = ?", loanInfo.ID).Scan(&totalRepaid)
			loanInfo.OutstandingBalance = loanInfo.Principal - totalRepaid
			if loanInfo.Principal > 0 {
				loanInfo.RepaymentAmountProgress = totalRepaid / loanInfo.Principal
			}
			response.Loans = append(response.Loans, loanInfo)
		}
	} else {
		log.Printf("查询活动贷款失败: %v", err)
	}

	c.JSON(http.StatusOK, response)
}
