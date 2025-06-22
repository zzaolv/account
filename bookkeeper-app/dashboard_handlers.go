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

// getTotalsForPeriod 函数保持不变
func getTotalsForPeriod(db *sql.DB, year, month string) (float64, float64, error) {
	var income, expense float64
	query := "SELECT type, SUM(amount) FROM transactions"
	var conditions []string
	var args []interface{}
	if year == "" && month == "" {
	} else {
		if year != "" {
			conditions = append(conditions, "strftime('%Y', transaction_date) = ?")
			args = append(args, year)
		}
		if month != "" {
			monthFormatted := fmt.Sprintf("%02s", month)
			conditions = append(conditions, "strftime('%m', transaction_date) = ?")
			args = append(args, monthFormatted)
		}
	}
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
		if rows.Scan(&transType, &total) == nil {
			if transType == "income" {
				income = total
			} else if transType == "expense" {
				expense = total
			}
		}
	}
	return income, expense, nil
}

// GetDashboardCards 函数保持不变
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
	if prevYear != "" {
		prevIncome, prevExpense, err = getTotalsForPeriod(h.DB, prevYear, prevMonth)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取上一周期数据失败: " + err.Error()})
			return
		}
	}
	var totalLoan float64
	h.DB.QueryRow("SELECT COALESCE(SUM(principal), 0) FROM loans WHERE status = 'active'").Scan(&totalLoan)
	cards := []DashboardCard{{Title: "总收入", Value: currentIncome, PrevValue: prevIncome, Icon: "TrendingUp"}, {Title: "总支出", Value: currentExpense, PrevValue: prevExpense, Icon: "TrendingDown"}, {Title: "净结余", Value: currentIncome - currentExpense, PrevValue: prevIncome - prevExpense, Icon: "Scale"}, {Title: "总借款", Value: totalLoan, PrevValue: 0, Icon: "Landmark"}}
	c.JSON(http.StatusOK, cards)
}

// GetAnalyticsCharts (【后端重塑数据】版本)
func (h *DBHandler) GetAnalyticsCharts(c *gin.Context) {
	year := c.Query("year")
	month := c.Query("month")

	// 1. 创建最终的响应结构体
	var response AnalyticsChartsResponse
	response.ExpenseTrend = []ChartDataPoint{}    // 初始化为空切片，避免null
	response.CategoryExpense = []ChartDataPoint{} // 初始化为空切片，避免null

	// 2. 获取支出趋势 (逻辑不变，但确保数据填充到 response.ExpenseTrend)
	trendQuery := ""
	var trendArgs []interface{}
	if year != "" && month == "" { // 按年查询
		trendQuery = `SELECT strftime('%Y-%m', transaction_date) as period, SUM(amount) FROM transactions WHERE type = 'expense' AND strftime('%Y', transaction_date) = ? GROUP BY period ORDER BY period ASC`
		trendArgs = append(trendArgs, year)
	} else if year != "" && month != "" { // 按月查询 (趋势按日)
		monthFormatted := fmt.Sprintf("%02s", month)
		trendQuery = `SELECT strftime('%d', transaction_date) as period, SUM(amount) FROM transactions WHERE type = 'expense' AND strftime('%Y', transaction_date) = ? AND strftime('%m', transaction_date) = ? GROUP BY period ORDER BY period ASC`
		trendArgs = append(trendArgs, year, monthFormatted)
	} else { // 查询所有年份
		trendQuery = `SELECT strftime('%Y', transaction_date) as period, SUM(amount) FROM transactions WHERE type = 'expense' GROUP BY period ORDER BY period ASC`
	}

	rows, err := h.DB.Query(trendQuery, trendArgs...)
	if err != nil {
		log.Printf("查询支出趋势失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取支出趋势数据失败"})
		return
	}
	// defer rows.Close() // Defer is removed to close it manually before the next query

	for rows.Next() {
		var point ChartDataPoint
		if err := rows.Scan(&point.Name, &point.Value); err != nil {
			log.Printf("扫描支出趋势数据失败: %v", err)
			continue // 跳过错误行
		}
		response.ExpenseTrend = append(response.ExpenseTrend, point)
	}
	rows.Close() // Manually close rows here

	// 3. 获取按分类的支出 (逻辑不变，但确保数据填充到 response.CategoryExpense)
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

	log.Printf("最终饼图查询语句: %s, 参数: %v", catQueryBuilder.String(), catArgs)

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
			continue // 跳过错误行
		}
		response.CategoryExpense = append(response.CategoryExpense, point)
	}

	log.Printf("成功构建图表API响应: %+v", response)
	c.JSON(http.StatusOK, response)
}

// GetDashboardWidgets (完整的，无省略)
func (h *DBHandler) GetDashboardWidgets(c *gin.Context) {
	yearStr := c.Query("year")
	monthStr := c.Query("month")

	var response DashboardWidgetsResponse

	// 1. 获取预算信息
	response.Budgets = []DashboardBudgetSummary{}
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

		targetYear := time.Now().Year()
		if y, err := strconv.Atoi(yearStr); err == nil {
			targetYear = y
		}

		if period == "monthly" {
			targetMonth := time.Now().Month()
			if m, err := strconv.Atoi(monthStr); err == nil && monthStr != "" {
				targetMonth = time.Month(m)
			} else if yearStr == "" && monthStr == "" {
				// 如果是“全部时间”，预算组件也显示当前月份
			}
			timeCondition.WriteString(" AND strftime('%Y', transaction_date) = ? AND strftime('%m', transaction_date) = ?")
			args = append(args, fmt.Sprintf("%d", targetYear), fmt.Sprintf("%02d", targetMonth))
		} else if period == "yearly" {
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

	// 2. 获取活动中的借贷信息
	response.Loans = []DashboardLoanInfo{}
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
	}
	c.JSON(http.StatusOK, response)
}
