// bookkeeper-app/data_handlers.go
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// ExportData (【全新备份功能】) - 导出整个数据库文件
func (h *DBHandler) ExportData(c *gin.Context) {
	// 从中间件获取用户信息
	userIDValue, _ := c.Get("userID")
	userID, _ := userIDValue.(int64)

	usernameValue, _ := c.Get("claims")
	claims, ok := usernameValue.(*Claims)
	username := "user" // 默认值
	if ok {
		username = claims.Username
	}

	logger := h.Logger.With(slog.Int64("userID", userID))

	dbPath := getDBPath()

	// 确保数据库文件存在
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		logger.Error("数据库文件不存在，无法导出", "path", dbPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库文件不存在，无法导出"})
		return
	}

	filename := fmt.Sprintf("bookkeeper_backup_%s_%s.db", username, time.Now().Format("20060102_150405"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")

	c.File(dbPath)
}

// ImportData (【全新恢复功能】) - 恢复整个数据库文件
func (h *DBHandler) ImportData(c *gin.Context) {
	userIDValue, _ := c.Get("userID")
	userID, _ := userIDValue.(int64)
	logger := h.Logger.With(slog.Int64("userID", userID))

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败: " + err.Error()})
		return
	}

	// 验证文件扩展名
	if filepath.Ext(file.Filename) != ".db" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传 .db 格式的数据库备份文件"})
		return
	}

	// 在进行危险操作前，先关闭当前的数据库连接，释放文件锁
	if err := h.DB.Close(); err != nil {
		logger.Error("关闭当前数据库连接失败", "error", err)
		// 即使关闭失败，也继续尝试，因为接下来的覆盖操作可能会成功
	}
	logger.Info("数据库连接已关闭，准备进行文件替换")

	dbPath := getDBPath()

	// 备份当前数据库，以防恢复失败
	backupPath := dbPath + ".bak-" + time.Now().Format("20060102150405")
	if _, err := os.Stat(dbPath); err == nil {
		err := os.Rename(dbPath, backupPath)
		if err != nil {
			logger.Error("备份当前数据库失败", "error", err)
			// 尝试重新连接数据库
			h.DB, _ = initializeDB(h.Logger)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复失败：无法备份现有数据库"})
			return
		}
		logger.Info("当前数据库已备份", "path", backupPath)
	}

	// 使用上传的文件覆盖当前数据库文件
	err = c.SaveUploadedFile(file, dbPath)
	if err != nil {
		logger.Error("用上传文件覆盖数据库失败", "error", err)
		// 恢复失败，尝试将备份文件还原
		os.Rename(backupPath, dbPath)
		h.DB, _ = initializeDB(h.Logger) // 重新连接
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复失败：无法写入新文件"})
		return
	}

	logger.Info("数据库文件已成功被上传的文件覆盖")

	// 恢复成功后，重新初始化数据库连接
	newDB, err := initializeDB(h.Logger)
	if err != nil {
		logger.Error("恢复后重新初始化数据库连接失败", "error", err)
		// 恢复失败，这是一个严重问题，可能上传的文件是坏的
		os.Rename(backupPath, dbPath) // 再次尝试还原
		h.DB, _ = initializeDB(h.Logger)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复失败：上传的数据库文件可能已损坏或格式不兼容"})
		return
	}

	h.DB = newDB
	logger.Info("数据库已成功从备份恢复，并重新连接")

	c.JSON(http.StatusOK, gin.H{
		"message": "数据恢复成功！应用将需要刷新或重新登录以加载新数据。",
	})
}
