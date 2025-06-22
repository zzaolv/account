// bookkeeper-app/category_handlers.go
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mattn/go-sqlite3"
)

// GetCategories 处理获取所有分类的请求
func (h *DBHandler) GetCategories(c *gin.Context) {
	rows, err := h.DB.Query("SELECT id, name, type, icon, created_at FROM categories ORDER BY type, name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法从数据库获取分类: " + err.Error()})
		return
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var cat Category
		var icon sql.NullString
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &icon, &cat.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "扫描分类数据失败: " + err.Error()})
			return
		}
		cat.Icon = icon.String
		categories = append(categories, cat)
	}

	c.JSON(http.StatusOK, categories)
}

// CreateCategory 处理新增一个分类的请求
func (h *DBHandler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	stmt, err := h.DB.Prepare("INSERT INTO categories(id, name, type, icon, created_at) VALUES(?, ?, ?, ?, ?)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库准备语句失败: " + err.Error()})
		return
	}
	defer stmt.Close()

	createdAt := time.Now().Format(time.RFC3339)
	_, err = stmt.Exec(req.ID, req.Name, req.Type, req.Icon, createdAt)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			c.JSON(http.StatusConflict, gin.H{"error": "创建分类失败，ID或名称已存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建分类失败: " + err.Error()})
		}
		return
	}

	newCategory := Category{
		ID:        req.ID,
		Name:      req.Name,
		Type:      req.Type,
		Icon:      req.Icon,
		CreatedAt: createdAt,
	}

	c.JSON(http.StatusCreated, newCategory)
}

// UpdateCategory 更新一个分类
func (h *DBHandler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	stmt, err := h.DB.Prepare("UPDATE categories SET name = ?, icon = ? WHERE id = ?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库准备更新语句失败: " + err.Error()})
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(req.Name, req.Icon, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新分类失败: " + err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的分类"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "分类更新成功"})
}

// DeleteCategory 【安全删除】一个分类
func (h *DBHandler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")

	// 【安全检查】在删除前，检查是否有流水正在使用此分类
	var count int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE category_id = ?", id).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查分类使用情况失败: " + err.Error()})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("无法删除分类，仍有 %d 条流水记录正在使用它", count)})
		return
	}

	// 执行删除
	stmt, err := h.DB.Prepare("DELETE FROM categories WHERE id = ?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库准备删除语句失败: " + err.Error()})
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(id)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			c.JSON(http.StatusConflict, gin.H{"error": "由于外键约束，无法删除此分类"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除分类失败: " + err.Error()})
		}
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定ID的分类"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "分类删除成功"})
}
