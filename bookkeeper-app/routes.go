// bookkeeper-app/routes.go
package main

import (
	"net"
	"net/http"
	"net/url"

	//	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// 【新增】isOriginAllowed 函数，用于动态判断来源是否被允许
// 这个函数会检查来源是否匹配我们的白名单或指定的IP网段
func isOriginAllowed(origin string, allowedSubnet *net.IPNet, allowedOrigins []string) bool {
	// 1. 直接匹配精确的来源白名单
	for _, o := range allowedOrigins {
		if o == origin {
			return true
		}
	}

	// 2. 解析来源 URL，提取主机名
	u, err := url.Parse(origin)
	if err != nil {
		return false // 如果来源不是一个有效的URL，则拒绝
	}
	hostname := u.Hostname() // 提取主机名，如 "192.168.31.32"

	// 3. 将主机名解析为 IP 地址
	ip := net.ParseIP(hostname)
	if ip == nil {
		return false // 如果主机名不是一个有效的IP地址，则不进行网段匹配
	}

	// 4. 检查 IP 地址是否在允许的子网内
	if allowedSubnet.Contains(ip) {
		return true
	}

	// 5. 如果都不匹配，则拒绝
	return false
}

// setupRouter (完整的，无省略)
func setupRouter(handler *DBHandler) *gin.Engine {
	router := gin.Default()

	// 【核心修改】使用 AllowOriginFunc 自定义 CORS 策略
	_, subnet, _ := net.ParseCIDR("192.168.31.0/24") // 定义允许的局域网网段
	allowedStaticOrigins := []string{
		"http://localhost:5173",
		"https://acc.zeaolv.top",
	}

	config := cors.Config{
		// 不再使用 AllowOrigins，而是使用 AllowOriginFunc
		AllowOriginFunc: func(origin string) bool {
			return isOriginAllowed(origin, subnet, allowedStaticOrigins)
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// 如果你想在日志中看到被拒绝的来源，可以添加一个检查
	// 这对于调试非常有用
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" && !isOriginAllowed(origin, subnet, allowedStaticOrigins) {
			// 你可以在这里记录日志，了解哪些未授权的来源正在尝试访问
			// log.Printf("CORS: Rejected origin %s", origin)
		}
		c.Next()
	})

	router.Use(cors.New(config))

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "欢迎来到简易记账本后端! V7.0 - 资金闭环版"})
	})

	apiV1 := router.Group("/api/v1")
	{
		// ... [其他路由代码保持不变] ...
		// 分类 (Category)
		apiV1.GET("/categories", handler.GetCategories)
		apiV1.POST("/categories", handler.CreateCategory)
		apiV1.PUT("/categories/:id", handler.UpdateCategory)
		apiV1.DELETE("/categories/:id", handler.DeleteCategory)

		// 流水 (Transaction)
		apiV1.POST("/transactions", handler.CreateTransaction)
		apiV1.GET("/transactions", handler.GetTransactions)
		apiV1.DELETE("/transactions/:id", handler.DeleteTransaction)

		// 借贷 (Loan)
		apiV1.POST("/loans", handler.CreateLoan)
		apiV1.GET("/loans", handler.GetLoans)
		apiV1.PUT("/loans/:id", handler.UpdateLoan)
		apiV1.PUT("/loans/:id/status", handler.UpdateLoanStatus)
		apiV1.POST("/loans/:id/settle", handler.SettleLoan)
		apiV1.DELETE("/loans/:id", handler.DeleteLoan)

		// 预算 (Budget)
		apiV1.POST("/budgets", handler.CreateOrUpdateBudget)
		apiV1.GET("/budgets", handler.GetBudgets)
		apiV1.DELETE("/budgets/:id", handler.DeleteBudget)

		// 账户 (Account)
		accounts := apiV1.Group("/accounts")
		{
			accounts.GET("", handler.GetAccounts)
			accounts.POST("", handler.CreateAccount)
			accounts.PUT("/:id", handler.UpdateAccount)
			accounts.DELETE("/:id", handler.DeleteAccount)
			accounts.POST("/:id/set_primary", handler.SetPrimaryAccount)
			accounts.POST("/transfer", handler.TransferFunds)
			accounts.POST("/execute_monthly_transfer", handler.ExecuteMonthlyTransfer)
		}

		// 仪表盘 (Dashboard) & 分析 (Analytics)
		apiV1.GET("/dashboard/cards", handler.GetDashboardCards)
		apiV1.GET("/analytics/charts", handler.GetAnalyticsCharts)
		apiV1.GET("/dashboard/widgets", handler.GetDashboardWidgets)

		// 数据管理 (Data Management)
		apiV1.GET("/data/export", handler.ExportData)
		apiV1.POST("/data/import", handler.ImportData)
	}

	return router
}

// 检查字符串是否在切片中
// 这个函数现在被 isOriginAllowed 中的逻辑替代了，但保留以供参考
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
