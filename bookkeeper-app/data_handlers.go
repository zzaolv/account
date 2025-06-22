// bookkeeper-app/data_handlers.go
package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ExportData 导出交易数据为CSV文件
func (h *DBHandler) ExportData(c *gin.Context) {
	rows, err := h.DB.Query(`
        SELECT t.id, t.type, t.amount, t.transaction_date, t.description, c.name as category_name
        FROM transactions t
        LEFT JOIN categories c ON t.category_id = c.id
        ORDER BY t.transaction_date DESC
    `)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询交易数据失败: " + err.Error()})
		return
	}
	defer rows.Close()

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename=transactions_"+time.Now().Format("20060102")+".csv")
	c.Header("Content-Type", "text/csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// 写入CSV头部
	writer.Write([]string{"ID", "Type", "Amount", "Date", "Description", "Category"})

	for rows.Next() {
		var id int
		var transType, date, description, categoryName string
		var amount float64
		if err := rows.Scan(&id, &transType, &amount, &date, &description, &categoryName); err != nil {
			// log error, but continue
			fmt.Println("Error scanning row for CSV export:", err)
			continue
		}
		record := []string{
			fmt.Sprintf("%d", id),
			transType,
			fmt.Sprintf("%.2f", amount),
			date,
			description,
			categoryName,
		}
		writer.Write(record)
	}
}

// ImportData 从上传的文件导入数据
// ImportData 从上传的文件导入数据 (完整实现)
func (h *DBHandler) ImportData(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败: " + err.Error()})
		return
	}

	// 检查文件类型
	if !strings.HasSuffix(file.Filename, ".csv") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传CSV格式的文件"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开上传文件失败: " + err.Error()})
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)

	// 读取并验证CSV头部
	header, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取CSV头部失败: " + err.Error()})
		return
	}
	expectedHeader := []string{"ID", "Type", "Amount", "Date", "Description", "Category"}
	if fmt.Sprintf("%v", header) != fmt.Sprintf("%v", expectedHeader) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV文件头部格式不正确，应为: " + strings.Join(expectedHeader, ",")})
		return
	}

	// 开始数据库事务
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "开启数据库事务失败: " + err.Error()})
		return
	}

	// 使用 tx.Prepare
	stmt, err := tx.Prepare(`INSERT INTO transactions(type, amount, transaction_date, description, category_id, created_at) VALUES(?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库准备插入语句失败: " + err.Error()})
		return
	}
	defer stmt.Close()

	var successCount, errorCount int
	var errorMessages []string
	createdAt := time.Now().Format(time.RFC3339)

	// 逐行读取和处理
	for i := 2; ; i++ { // i 从2开始，因为第1行是头部
		record, err := reader.Read()
		if err == io.EOF {
			break // 文件结束
		}
		if err != nil {
			log.Printf("读取第 %d 行CSV记录失败: %v", i, err)
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("第%d行: 读取失败", i))
			continue
		}

		if len(record) != len(expectedHeader) {
			log.Printf("第 %d 行字段数量不匹配: 期望 %d, 得到 %d", i, len(expectedHeader), len(record))
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("第%d行: 字段数量不匹配", i))
			continue
		}

		// --- 数据验证和转换 ---
		transType := strings.ToLower(record[1])
		if transType != "income" && transType != "expense" {
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("第%d行: 无效的类型 '%s'", i, record[1]))
			continue
		}

		amount, err := strconv.ParseFloat(record[2], 64)
		if err != nil || amount <= 0 {
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("第%d行: 无效的金额 '%s'", i, record[2]))
			continue
		}

		date, err := time.Parse("2006-01-02", record[3])
		if err != nil {
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("第%d行: 无效的日期格式 '%s'", i, record[3]))
			continue
		}

		description := record[4]
		categoryName := record[5]
		var categoryID sql.NullString // 使用 sql.NullString 处理可能为空的情况

		if categoryName != "" {
			// 在事务中查询
			err := tx.QueryRow("SELECT id FROM categories WHERE name = ?", categoryName).Scan(&categoryID)
			if err != nil {
				// 如果找不到分类，可以选择跳过或归为默认，这里我们选择报错
				errorCount++
				errorMessages = append(errorMessages, fmt.Sprintf("第%d行: 找不到分类 '%s'", i, categoryName))
				continue
			}
		}

		// --- 执行插入 ---
		_, err = stmt.Exec(transType, amount, date.Format("2006-01-02"), description, categoryID, createdAt)
		if err != nil {
			tx.Rollback() // 任何插入失败都回滚整个事务
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("数据库插入第%d行数据时失败: %v", i, err)})
			return
		}
		successCount++
	}

	if errorCount > 0 {
		tx.Rollback() // 如果有任何验证失败的行，则回滚所有已插入的数据
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "导入失败，文件中有错误数据",
			"details": errorMessages,
		})
		return
	}

	// 全部成功，提交事务
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交数据库事务失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("文件上传成功，成功导入 %d 条记录。", successCount)})
}
