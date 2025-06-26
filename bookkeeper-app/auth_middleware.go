// bookkeeper-app/auth_middleware.go
package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware 是一个Gin中间件，用于验证JWT
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请求未包含认证信息"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证信息格式错误"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的签名"})
				c.Abort()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的Token", "details": err.Error()})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token不再有效"})
			c.Abort()
			return
		}

		// 将解析出的用户信息存储在 context 中，方便后续的 handler 使用
		c.Set("claims", claims)
		c.Set("userID", claims.UserID) // 特别设置 userID，使用更方便

		c.Next()
	}
}

// AdminMiddleware 确保用户是管理员
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 首先，执行通用的认证中间件逻辑
		AuthMiddleware()(c)
		// 如果前面的中间件已经中止了请求，c.IsAborted() 会是 true
		if c.IsAborted() {
			return
		}

		// 从 context 获取 claims
		claims, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "在中间件中无法找到用户信息"})
			c.Abort()
			return
		}

		userClaims, ok := claims.(*Claims)
		if !ok || !userClaims.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}

		c.Next()
	}
}
