// bookkeeper-app/auth_handlers.go
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// checkPasswordHash 比较明文密码和哈希值是否匹配
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateJWT 为指定用户生成一个JWT
// 【修改】接受 duration 参数以控制有效期
func generateJWT(user User, duration time.Duration) (string, error) {
	expirationTime := time.Now().Add(duration)
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// 【新增】生成安全的随机字符串用于 Refresh Token
func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// 【新增】哈希 Refresh Token 以便安全存储
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// 【新增】记录登录尝试
func (h *DBHandler) recordLoginAttempt(userID sql.NullInt64, username, ip, userAgent, status string) {
	_, err := h.DB.Exec(
		"INSERT INTO login_history (user_id, username_attempt, ip_address, user_agent, status, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		userID, username, ip, userAgent, status, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		h.Logger.Error("记录登录历史失败", "error", err)
	}
}

// Login 处理用户登录请求 (【核心重构】)
func (h *DBHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	var user User
	var lockoutUntil sql.NullString
	err := h.DB.QueryRow(
		"SELECT id, username, password_hash, is_admin, must_change_password, failed_login_attempts, lockout_until FROM users WHERE username = ?",
		req.Username,
	).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.MustChangePassword, &user.FailedLoginAttempts, &lockoutUntil,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			h.recordLoginAttempt(sql.NullInt64{}, req.Username, ip, userAgent, "failure")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		} else {
			h.Logger.Error("查询用户失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库错误"})
		}
		return
	}

	// 检查账户是否被锁定
	if lockoutUntil.Valid {
		lockoutTime, _ := time.Parse(time.RFC3339, lockoutUntil.String)
		if time.Now().Before(lockoutTime) {
			h.recordLoginAttempt(sql.NullInt64{Int64: user.ID, Valid: true}, req.Username, ip, userAgent, "failure")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": fmt.Sprintf("登录尝试过于频繁，请于 %.0f 分钟后重试", time.Until(lockoutTime).Minutes())})
			return
		}
	}

	// 验证密码
	if !checkPasswordHash(req.Password, user.PasswordHash) {
		// 登录失败，更新失败计数器
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= 5 {
			newLockoutTime := time.Now().Add(15 * time.Minute)
			_, err = h.DB.Exec("UPDATE users SET failed_login_attempts = ?, lockout_until = ? WHERE id = ?", user.FailedLoginAttempts, newLockoutTime.Format(time.RFC3339), user.ID)
		} else {
			_, err = h.DB.Exec("UPDATE users SET failed_login_attempts = ? WHERE id = ?", user.FailedLoginAttempts, user.ID)
		}
		if err != nil {
			h.Logger.Error("更新登录失败计数失败", "error", err)
		}
		h.recordLoginAttempt(sql.NullInt64{Int64: user.ID, Valid: true}, req.Username, ip, userAgent, "failure")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 登录成功，重置失败计数器
	if user.FailedLoginAttempts > 0 || lockoutUntil.Valid {
		_, err = h.DB.Exec("UPDATE users SET failed_login_attempts = 0, lockout_until = NULL WHERE id = ?", user.ID)
		if err != nil {
			h.Logger.Error("重置登录失败计数失败", "error", err)
		}
	}
	h.recordLoginAttempt(sql.NullInt64{Int64: user.ID, Valid: true}, req.Username, ip, userAgent, "success")

	// 生成 Access Token
	accessToken, err := generateJWT(user, 1*time.Hour) // 有效期1小时
	if err != nil {
		h.Logger.Error("生成JWT失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法生成token"})
		return
	}

	response := gin.H{
		"access_token":         accessToken,
		"must_change_password": user.MustChangePassword,
		"username":             user.Username,
		"is_admin":             user.IsAdmin,
	}

	// 如果需要“记住我”，则生成并返回 Refresh Token
	if req.RememberMe {
		refreshToken, err := generateSecureToken(32)
		if err != nil {
			h.Logger.Error("生成Refresh Token失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法生成refresh token"})
			return
		}

		refreshTokenHash := hashToken(refreshToken)
		expiresAt := time.Now().Add(30 * 24 * time.Hour) // 有效期30天
		createdAt := time.Now().Format(time.RFC3339)

		_, err = h.DB.Exec(
			"INSERT INTO refresh_tokens (user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?)",
			user.ID, refreshTokenHash, expiresAt.Format(time.RFC3339), createdAt,
		)
		if err != nil {
			h.Logger.Error("存储Refresh Token失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法存储refresh token"})
			return
		}
		response["refresh_token"] = refreshToken
	}

	c.JSON(http.StatusOK, response)
}

// 【新增】RefreshToken 处理器
func (h *DBHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}

	refreshTokenHash := hashToken(req.RefreshToken)

	var userID int64
	var expiresAtStr string
	err := h.DB.QueryRow("SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash = ?", refreshTokenHash).Scan(&userID, &expiresAtStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的refresh token"})
		return
	}

	expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)
	if time.Now().After(expiresAt) {
		// 清理过期的token
		h.DB.Exec("DELETE FROM refresh_tokens WHERE token_hash = ?", refreshTokenHash)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token已过期"})
		return
	}

	// 成功，颁发新的Access Token
	var user User
	err = h.DB.QueryRow("SELECT id, username, is_admin FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.IsAdmin)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "找不到关联用户"})
		return
	}

	newAccessToken, err := generateJWT(user, 1*time.Hour)
	if err != nil {
		h.Logger.Error("刷新时生成JWT失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法生成新token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": newAccessToken})
}

// Register (管理员功能) 创建一个新用户
func (h *DBHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		h.Logger.Error("哈希密码失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
		return
	}

	createdAt := time.Now().Format(time.RFC3339)

	// 【修正】新用户不再需要创建预设分类，直接使用共享分类即可
	_, err = h.DB.Exec(
		"INSERT INTO users (username, password_hash, is_admin, must_change_password, created_at) VALUES (?, ?, ?, ?, ?)",
		req.Username, hashedPassword, 0, 1, createdAt, // 默认非管理员，需要修改密码
	)
	if err != nil {
		// 检查是否是唯一约束冲突
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		} else {
			h.Logger.Error("插入新用户失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("用户 '%s' 创建成功", req.Username)})
}

// UpdatePassword 处理用户修改密码的请求
func (h *DBHandler) UpdatePassword(c *gin.Context) {
	claims, _ := c.Get("claims")
	userClaims := claims.(*Claims)

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	var user User
	err := h.DB.QueryRow("SELECT password_hash, must_change_password FROM users WHERE id = ?", userClaims.UserID).Scan(
		&user.PasswordHash, &user.MustChangePassword,
	)
	if err != nil {
		h.Logger.Error("更新密码时查询用户失败", "error", err, "userID", userClaims.UserID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	if !user.MustChangePassword {
		if req.OldPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "需要提供旧密码"})
			return
		}
		if !checkPasswordHash(req.OldPassword, user.PasswordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "旧密码错误"})
			return
		}
	}

	newHashedPassword, err := hashPassword(req.NewPassword)
	if err != nil {
		h.Logger.Error("更新密码时哈希新密码失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	_, err = h.DB.Exec(
		"UPDATE users SET password_hash = ?, must_change_password = 0 WHERE id = ?",
		newHashedPassword, userClaims.UserID,
	)
	if err != nil {
		h.Logger.Error("更新用户密码到数据库失败", "error", err, "userID", userClaims.UserID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码更新成功"})
}

// GetUsers (管理员功能) 获取所有用户列表
func (h *DBHandler) GetUsers(c *gin.Context) {
	rows, err := h.DB.Query("SELECT id, username, is_admin, must_change_password, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		h.Logger.Error("获取用户列表失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.IsAdmin, &u.MustChangePassword, &u.CreatedAt); err != nil {
			h.Logger.Error("扫描用户数据失败", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "处理数据失败"})
			return
		}
		users = append(users, u)
	}
	c.JSON(http.StatusOK, users)
}

// DeleteUser (管理员功能) 删除一个用户
func (h *DBHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	var user User
	err := h.DB.QueryRow("SELECT id, is_admin FROM users WHERE id = ?", userID).Scan(&user.ID, &user.IsAdmin)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定用户"})
		return
	}
	if user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能删除管理员账户"})
		return
	}

	_, err = h.DB.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		h.Logger.Error("删除用户失败", "error", err, "userID", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户及其所有数据已成功删除"})
}
