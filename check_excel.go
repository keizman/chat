package main

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	workExcelPath := "work/work.xlsx"
	workSheetName := "Sheet1"

	// Read and display the Excel content
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("CURRENT EXCEL CONTENT:")
	fmt.Println(strings.Repeat("=", 60))
	
	f, err := excelize.OpenFile(workExcelPath)
	if err != nil {
		fmt.Printf("Error opening Excel file: %v\n", err)
		return
	}
	defer f.Close()

	rows, err := f.GetRows(workSheetName)
	if err != nil {
		fmt.Printf("Error reading rows: %v\n", err)
		return
	}

	for i, row := range rows {
		if i == 0 {
			fmt.Printf("Headers: %v\n", row)
			fmt.Println(strings.Repeat("-", 50))
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

func getColumnValue(row []string, index int) string {
	if index < len(row) {
		return row[index]
	}
	return ""
}