package store

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tinode/chat/server/store/types"
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

// syncWorkMessage syncs a work message to Excel if the topic has 'work' tag
func syncWorkMessage(msg *types.Message, topicTags []string) error {
	// Check if topic has 'work' tag
	hasWorkTag := false
	for _, tag := range topicTags {
		if tag == "work" {
			hasWorkTag = true
			break
		}
	}

	log.Printf("syncWorkMessage: topic=%s, tags=%v, hasWorkTag=%v", msg.Topic, topicTags, hasWorkTag)

	if !hasWorkTag {
		log.Printf("syncWorkMessage: skipping message for topic %s - no 'work' tag", msg.Topic)
		return nil // Not a work message, skip
	}

	// Extract message content as string
	log.Printf("syncWorkMessage: raw msg.Content type=%T, value=%+v", msg.Content, msg.Content)
	
	content, ok := msg.Content.(string)
	if !ok {
		// Try to convert from map or other types
		if contentMap, ok := msg.Content.(map[string]interface{}); ok {
			log.Printf("syncWorkMessage: content is map, keys=%v", getMapKeys(contentMap))
			if txt, exists := contentMap["txt"]; exists {
				content = fmt.Sprintf("%v", txt)
				log.Printf("syncWorkMessage: extracted txt=%s", content)
			}
		}
		if content == "" {
			content = fmt.Sprintf("%v", msg.Content)
			log.Printf("syncWorkMessage: fallback content=%s", content)
		}
	} else {
		log.Printf("syncWorkMessage: content is string=%s", content)
	}

	if content == "" {
		log.Printf("syncWorkMessage: no content to sync")
		return nil // No content to sync
	}

	log.Printf("syncWorkMessage: calling updateWorkExcel with content=%s", content)
	return updateWorkExcel(content)
}

// updateWorkExcel updates the work Excel file with the message content
func updateWorkExcel(content string) error {
	log.Printf("updateWorkExcel: starting with content=%s", content)
	
	// Get current working directory for debugging
	if cwd, err := os.Getwd(); err == nil {
		log.Printf("updateWorkExcel: current working directory=%s", cwd)
	}
	
	// Get absolute path
	absPath, _ := filepath.Abs(workExcelPath)
	log.Printf("updateWorkExcel: absolute file path=%s", absPath)
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(workExcelPath), 0755); err != nil {
		log.Printf("updateWorkExcel: failed to create directory: %v", err)
		return fmt.Errorf("failed to create directory: %v", err)
	}
	log.Printf("updateWorkExcel: directory created/exists")

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
	log.Printf("updateWorkExcel: saving file to %s", workExcelPath)
	if err := f.SaveAs(workExcelPath); err != nil {
		log.Printf("updateWorkExcel: failed to save Excel file: %v", err)
		return fmt.Errorf("failed to save Excel file: %v", err)
	}

	log.Printf("Successfully synced work message to Excel: %s", workExcelPath)
	log.Printf("updateWorkExcel: completed successfully")
	return nil
}

// getTopicTags retrieves the tags for a given topic
func getTopicTags(topicName string) ([]string, error) {
	// Get topic information from the database
	topic, err := adp.TopicGet(topicName)
	if err != nil {
		return nil, err
	}
	
	if topic == nil {
		return nil, nil
	}
	
	return topic.Tags, nil
}

// getMapKeys returns the keys of a map for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}