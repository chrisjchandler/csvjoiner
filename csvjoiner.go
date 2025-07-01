package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func loadCSV(filename string) ([][]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := csv.NewReader(f)
	return reader.ReadAll()
}

func loadKeyColumns(configPath string) ([]string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var keys []string
	err = json.Unmarshal(data, &keys)
	return keys, err
}

func findKeyColumnIndex(header []string, keyHeaders []string) int {
	for i, col := range header {
		for _, key := range keyHeaders {
			if strings.EqualFold(col, key) {
				return i
			}
		}
	}
	return -1
}

func main() {
	configPath := flag.String("config", "config.json", "Path to config listing equivalent join key column names")
	outputPath := flag.String("out", "output.csv", "Output CSV file path")
	flag.Parse()

	files := flag.Args()
	if len(files) < 2 {
		fmt.Println("Provide at least two CSV files to merge.")
		os.Exit(1)
	}

	keyColumns, err := loadKeyColumns(*configPath)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	merged := make(map[string]map[string]string)
	allHeaders := map[string]bool{}
	seenCols := map[string]int{}

	for _, file := range files {
		records, err := loadCSV(file)
		if err != nil {
			panic(fmt.Errorf("failed to read %s: %w", file, err))
		}
		if len(records) < 2 {
			continue
		}

		header := records[0]
		keyIdx := findKeyColumnIndex(header, keyColumns)
		if keyIdx == -1 {
			panic(fmt.Errorf("no key column found in %s matching keys: %v", file, keyColumns))
		}

		for _, row := range records[1:] {
			if keyIdx >= len(row) {
				continue
			}
			key := row[keyIdx]
			if _, ok := merged[key]; !ok {
				merged[key] = map[string]string{}
			}

			for i, val := range row {
				colName := header[i]
				uniqueCol := colName
				if _, exists := merged[key][uniqueCol]; exists {
					count := seenCols[colName] + 1
					seenCols[colName] = count
					uniqueCol = fmt.Sprintf("%s_%d", colName, count)
				}
				merged[key][uniqueCol] = val
				allHeaders[uniqueCol] = true
			}
		}
	}

	// Consolidate headers
	var finalHeaders []string
	for col := range allHeaders {
		finalHeaders = append(finalHeaders, col)
	}

	// Write output CSV
	outFile, err := os.Create(*outputPath)
	if err != nil {
		panic(fmt.Errorf("failed to write output: %w", err))
	}
	defer outFile.Close()
	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	writer.Write(finalHeaders)
	for _, row := range merged {
		out := make([]string, len(finalHeaders))
		for i, col := range finalHeaders {
			out[i] = row[col]
		}
		writer.Write(out)
	}

	fmt.Printf("✅ Merged CSV written to: %s\n", *outputPath)
}
