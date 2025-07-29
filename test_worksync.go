package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	workExcelPath = "work/work.xlsx"
	workSheetName = "Sheet1"
)

// Excel column mapping
const (
	colTask        = "A" // 任务
	colTodo        = "B" // Todo
	colDescription = "C" // 说明
	colDate        = "D" // 日期
)

// testUpdateWorkExcel is a standalone test function for Excel sync
func testUpdateWorkExcel(content string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(workExcelPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	var f *excelize.File
	var err error

	// Try to open existing file, create new if it doesn't exist
	if _, err := os.Stat(workExcelPath); os.IsNotExist(err) {
		f = excelize.NewFile()
		// Get the default sheet name and rename it to our desired name
		defaultSheet := f.GetSheetName(0)
		f.SetSheetName(defaultSheet, workSheetName)
		// Set headers
		f.SetCellValue(workSheetName, colTask+"1", "任务")
		f.SetCellValue(workSheetName, colTodo+"1", "Todo")
		f.SetCellValue(workSheetName, colDescription+"1", "说明")
		f.SetCellValue(workSheetName, colDate+"1", "日期")
	} else {
		f, err = excelize.OpenFile(workExcelPath)
		if err != nil {
			return fmt.Errorf("failed to open Excel file: %v", err)
		}
		// Check if our worksheet exists, if not create it
		if _, err := f.GetSheetIndex(workSheetName); err != nil {
			// Sheet doesn't exist, create it
			index, err := f.NewSheet(workSheetName)
			if err != nil {
				return fmt.Errorf("failed to create sheet: %v", err)
			}
			f.SetActiveSheet(index)
			// Set headers
			f.SetCellValue(workSheetName, colTask+"1", "任务")
			f.SetCellValue(workSheetName, colTodo+"1", "Todo")
			f.SetCellValue(workSheetName, colDescription+"1", "说明")
			f.SetCellValue(workSheetName, colDate+"1", "日期")
		}
	}
	defer f.Close()

	// Get current date string in format: 29/07/2025 (DD/MM/YYYY)
	now := time.Now()
	currentDate := fmt.Sprintf("%d/%02d/%d", now.Day(), int(now.Month()), now.Year())

	// Find if there's already a row for today
	rows, err := f.GetRows(workSheetName)
	if err != nil {
		// If we can't get rows, assume it's a new file and create first data row
		rows = [][]string{
			{"任务", "Todo", "说明", "日期"}, // Header row
		}
	}

	var todayRowIndex int = -1
	for i, row := range rows {
		if i == 0 {
			continue // Skip header row
		}
		if len(row) > 3 && row[3] == currentDate { // Check date column
			todayRowIndex = i + 1 // Excel rows are 1-indexed
			break
		}
	}

	// Determine which column to update based on content prefix
	var targetCol string
	var cleanContent string

	if strings.HasPrefix(content, "Ta:") {
		targetCol = colTask
		cleanContent = strings.TrimSpace(content[3:])
	} else if strings.HasPrefix(content, "To:") {
		targetCol = colTodo
		cleanContent = strings.TrimSpace(content[3:])
	} else {
		targetCol = colDescription
		cleanContent = content
	}

	if todayRowIndex == -1 {
		// Insert new row for today (insert at row 2 to keep latest date at top)
		insertRowIndex := 2
		f.InsertRows(workSheetName, insertRowIndex, 1)
		
		// Set date for new row
		f.SetCellValue(workSheetName, colDate+fmt.Sprintf("%d", insertRowIndex), currentDate)
		
		// Set content in appropriate column
		f.SetCellValue(workSheetName, targetCol+fmt.Sprintf("%d", insertRowIndex), cleanContent)
	} else {
		// Update existing row for today
		// Get existing content in the target column
		existingContent, err := f.GetCellValue(workSheetName, targetCol+fmt.Sprintf("%d", todayRowIndex))
		if err != nil {
			existingContent = ""
		}
		
		// Append new content with line break if existing content exists
		var newContent string
		if existingContent != "" {
			newContent = existingContent + "\n" + cleanContent
		} else {
			newContent = cleanContent
		}
		
		f.SetCellValue(workSheetName, targetCol+fmt.Sprintf("%d", todayRowIndex), newContent)
	}

	// Save the file
	if err := f.SaveAs(workExcelPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %v", err)
	}

	log.Printf("Successfully synced work message to Excel: %s", workExcelPath)
	return nil
}

func main() {
	// Test with different message types
	testMessages := []string{
		"Ta: 热水器又坏, 现象相同, 原因可能也相同, 这次灯会一直亮",
		"To: 研究自动添加qq docs 内容的方法， 希望发送内容到某处即可每日自动添加当日内容到 qq docs",
		"发现 anyrouter 已不可用",
		"Ta: 另一个任务测试",
		"测试普通描述内容",
	}

	for i, msg := range testMessages {
		fmt.Printf("Testing message %d: %s\n", i+1, msg)
		if err := testUpdateWorkExcel(msg); err != nil {
			log.Printf("Error: %v", err)
		} else {
			fmt.Println("Success!")
		}
		fmt.Println()
	}

	// Read back and display the Excel content
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("FINAL EXCEL CONTENT:")
	fmt.Println(strings.Repeat("=", 50))
	if f, err := excelize.OpenFile(workExcelPath); err == nil {
		defer f.Close()
		if rows, err := f.GetRows(workSheetName); err == nil {
			for i, row := range rows {
				if i == 0 {
					fmt.Printf("Headers: %v\n", row)
					fmt.Println(strings.Repeat("-", 30))
				} else {
					fmt.Printf("Row %d:\n", i)
					if len(row) > 0 {
						fmt.Printf("  任务: %s\n", getColumnValue(row, 0))
					}
					if len(row) > 1 {
						fmt.Printf("  Todo: %s\n", getColumnValue(row, 1))
					}
					if len(row) > 2 {
						fmt.Printf("  说明: %s\n", getColumnValue(row, 2))
					}
					if len(row) > 3 {
						fmt.Printf("  日期: %s\n", getColumnValue(row, 3))
					}
					fmt.Println()
				}
			}
		}
	}
}

func getColumnValue(row []string, index int) string {
	if index < len(row) {
		return row[index]
	}
	return ""
}