// bookkeeper-app/category_handlers.go
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mattn/go-sqlite3"
)

// GetCategories (【核心修改】) 处理获取所有分类的请求
func (h *DBHandler) GetCategories(c *gin.Context) {
	userID, _ := c.Get("userID")
	logger := h.Logger.With(slog.Int64("userID", userID.(int64)))

	categories := []Category{}

	// 1. 获取所有共享分类
	sharedRows, err := h.DB.Query("SELECT id, name, type, icon, created_at, is_editable FROM shared_categories ORDER BY type, name")
	if err != nil {
		logger.Error("获取共享分类失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分类数据失败"})
		return
	}
	defer sharedRows.Close()

	for sharedRows.Next() {
		var cat Category
		var icon sql.NullString
		if err := sharedRows.Scan(&cat.ID, &cat.Name, &cat.Type, &icon, &cat.CreatedAt, &cat.IsEditable); err != nil {
			logger.Error("扫描共享分类数据失败", "error", err)
			continue
		}
		cat.Icon = icon.String
		cat.IsShared = true // 标记为共享
		categories = append(categories, cat)
	}

	// 2. 获取当前用户的私有分类
	privateRows, err := h.DB.Query("SELECT id, name, type, icon, created_at FROM categories WHERE user_id = ? ORDER BY type, name", userID)
	if err != nil {
		logger.Error("获取私有分类失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分类数据失败"})
		return
	}
	defer privateRows.Close()

	for privateRows.Next() {
		var cat Category
		var icon sql.NullString
		if err := privateRows.Scan(&cat.ID, &cat.Name, &cat.Type, &icon, &cat.CreatedAt); err != nil {
			logger.Error("扫描私有分类数据失败", "error", err)
			continue
		}
		cat.Icon = icon.String
		cat.IsShared = false  // 标记为私有
		cat.IsEditable = true // 私有分类总是可编辑的
		categories = append(categories, cat)
	}

	c.JSON(http.StatusOK, categories)
}

// CreateCategory (【核心修改】) 只创建用户私有分类
func (h *DBHandler) CreateCategory(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 检查ID和名称是否与任何现有分类（共享或私有）冲突
	var count int
	err := h.DB.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT id, name FROM shared_categories
			UNION ALL
			SELECT id, name FROM categories WHERE user_id = ?
		) WHERE id = ? OR name = ?
	`, userID, req.ID, req.Name).Scan(&count)
	if err != nil {
		h.Logger.Error("检查分类名和ID冲突失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "创建分类失败，名称或ID已存在（可能在共享分类或您的私有分类中）"})
		return
	}

	createdAt := time.Now().Format(time.RFC3339)
	// 只在用户的私有 categories 表中插入
	_, err = h.DB.Exec("INSERT INTO categories(id, user_id, name, type, icon, created_at) VALUES(?, ?, ?, ?, ?, ?)",
		req.ID, userID, req.Name, req.Type, req.Icon, createdAt)

	if err != nil {
		h.Logger.Warn("创建私有分类失败", "error", err, slog.Int64("userID", userID.(int64)))
		// SQLite的UNIQUE约束错误处理
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && (sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique || sqliteErr.Code == sqlite3.ErrConstraint) {
			c.JSON(http.StatusConflict, gin.H{"error": "创建分类失败，名称或ID已存在（数据库约束）"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建分类失败"})
		}
		return
	}

	newCategory := Category{
		ID:         req.ID,
		Name:       req.Name,
		Type:       req.Type,
		Icon:       req.Icon,
		CreatedAt:  createdAt,
		IsShared:   false,
		IsEditable: true,
	}

	c.JSON(http.StatusCreated, newCategory)
}

// UpdateCategory (【核心修改】) 只更新用户私有分类
func (h *DBHandler) UpdateCategory(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")
	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 检查新名称是否与其他分类冲突
	var count int
	err := h.DB.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT name FROM shared_categories
			UNION ALL
			SELECT name FROM categories WHERE user_id = ?
		) WHERE name = ?
	`, userID, req.Name).Scan(&count)
	if err == nil && count > 0 {
		// 检查这个冲突是不是自己
		var selfName string
		h.DB.QueryRow("SELECT name FROM categories WHERE id = ? AND user_id = ?", id, userID).Scan(&selfName)
		if selfName != req.Name {
			c.JSON(http.StatusConflict, gin.H{"error": "更新分类失败，该名称已被其他分类使用"})
			return
		}
	}

	// 只允许更新私有分类
	result, err := h.DB.Exec("UPDATE categories SET name = ?, icon = ? WHERE id = ? AND user_id = ?", req.Name, req.Icon, id, userID)
	if err != nil {
		h.Logger.Error("更新私有分类失败", "error", err, "categoryID", id, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新分类失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定的私有分类，或该分类为不可编辑的共享分类"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "分类更新成功"})
}

// DeleteCategory (【核心修改】) 只删除用户私有分类
func (h *DBHandler) DeleteCategory(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	// 检查是否有流水正在使用此分类
	var count int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE category_id = ? AND user_id = ?", id, userID).Scan(&count)
	if err != nil {
		h.Logger.Error("检查分类使用情况失败", "error", err, "categoryID", id, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查分类使用情况失败"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("无法删除分类，仍有 %d 条流水记录正在使用它", count)})
		return
	}

	// 只允许删除私有分类
	result, err := h.DB.Exec("DELETE FROM categories WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		h.Logger.Error("删除私有分类失败", "error", err, "categoryID", id, slog.Int64("userID", userID.(int64)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除分类失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定的私有分类，或该分类为不可删除的共享分类"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "分类删除成功"})
}

// 【新增】管理员管理共享分类的处理器
// ... (我们可以在 Admin 页面功能中添加这些接口)
