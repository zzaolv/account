// bookkeeper-app/routes.go
package main

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// setupRouter (完整的，无省略)
func setupRouter(handler *DBHandler) *gin.Engine {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "欢迎来到简易记账本后端! V5.0 - 最终修复版"})
	})

	apiV1 := router.Group("/api/v1")
	{
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
		apiV1.PUT("/loans/:id", handler.UpdateLoan) // 【新增】编辑借贷的路由
		apiV1.PUT("/loans/:id/status", handler.UpdateLoanStatus)
		apiV1.DELETE("/loans/:id", handler.DeleteLoan)

		// 预算 (Budget)
		apiV1.POST("/budgets", handler.CreateOrUpdateBudget)
		apiV1.GET("/budgets", handler.GetBudgets)
		apiV1.DELETE("/budgets/:id", handler.DeleteBudget)

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
