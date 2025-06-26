// bookkeeper-app/routes.go
package main

import (
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func isOriginAllowed(origin string, allowedSubnet *net.IPNet, allowedOrigins []string) bool {
	for _, o := range allowedOrigins {
		if o == origin {
			return true
		}
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	hostname := u.Hostname()
	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}
	if allowedSubnet.Contains(ip) {
		return true
	}
	return false
}

func setupRouter(handler *DBHandler) *gin.Engine {
	router := gin.Default()

	_, subnet, _ := net.ParseCIDR("192.168.31.0/24")
	allowedStaticOrigins := []string{
		"http://localhost:5173",
		"https://acc.zeaolv.top",
	}

	config := cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return isOriginAllowed(origin, subnet, allowedStaticOrigins) || origin == ""
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(config))

	base := router.Group("/api/v1")
	{
		auth := base.Group("/auth")
		{
			auth.POST("/login", handler.Login)
			auth.POST("/refresh", handler.RefreshToken) // 【新增】
		}

		protected := base.Group("/")
		protected.Use(AuthMiddleware())
		{
			protected.PUT("/auth/update_password", handler.UpdatePassword)

			protected.GET("/categories", handler.GetCategories)
			protected.POST("/categories", handler.CreateCategory)
			protected.PUT("/categories/:id", handler.UpdateCategory)
			protected.DELETE("/categories/:id", handler.DeleteCategory)

			protected.POST("/transactions", handler.CreateTransaction)
			protected.GET("/transactions", handler.GetTransactions)
			protected.DELETE("/transactions/:id", handler.DeleteTransaction)

			protected.POST("/loans", handler.CreateLoan)
			protected.GET("/loans", handler.GetLoans)
			protected.PUT("/loans/:id", handler.UpdateLoan)
			protected.PUT("/loans/:id/status", handler.UpdateLoanStatus)
			protected.POST("/loans/:id/settle", handler.SettleLoan)
			protected.DELETE("/loans/:id", handler.DeleteLoan)

			protected.POST("/budgets", handler.CreateOrUpdateBudget)
			protected.GET("/budgets", handler.GetBudgets)
			protected.DELETE("/budgets/:id", handler.DeleteBudget)

			accounts := protected.Group("/accounts")
			{
				accounts.GET("", handler.GetAccounts)
				accounts.POST("", handler.CreateAccount)
				accounts.PUT("/:id", handler.UpdateAccount)
				accounts.DELETE("/:id", handler.DeleteAccount)
				accounts.POST("/:id/set_primary", handler.SetPrimaryAccount)
			}

			protected.GET("/dashboard/cards", handler.GetDashboardCards)
			protected.GET("/analytics/charts", handler.GetAnalyticsCharts)
			protected.GET("/dashboard/widgets", handler.GetDashboardWidgets)

			protected.GET("/data/export", handler.ExportData)
			protected.POST("/data/import", handler.ImportData)
		}

		admin := base.Group("/admin")
		admin.Use(AdminMiddleware())
		{
			admin.POST("/users/register", handler.Register)
			admin.GET("/users", handler.GetUsers)
			admin.DELETE("/users/:id", handler.DeleteUser)
			admin.GET("/stats", handler.GetSystemStats)
		}
	}

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "欢迎来到简易记账本后端! V10.0 - 最终稳定版"})
	})

	return router
}
