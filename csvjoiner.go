package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func loadCSV(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return csv.NewReader(f).ReadAll()
}

func normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func findJoinKeyIndex(headers []string, candidates []string) (int, string) {
	for i, h := range headers {
		for _, c := range candidates {
			if normalize(h) == normalize(c) {
				return i, h
			}
		}
	}
	return -1, ""
}

func loadKeyCandidates(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var keys []string
	err = json.Unmarshal(data, &keys)
	return keys, err
}

func main() {
	config := flag.String("config", "config.json", "JSON file with join key column name candidates")
	output := flag.String("out", "output.csv", "Output CSV filename")
	flag.Parse()
	files := flag.Args()

	if len(files) < 2 {
		fmt.Println("Need at least two input CSV files")
		os.Exit(1)
	}

	keyCandidates, err := loadKeyCandidates(*config)
	if err != nil {
		panic(fmt.Errorf("could not read config: %w", err))
	}

	// Read and prepare first file
	baseCSV, err := loadCSV(files[0])
	if err != nil {
		panic(err)
	}
	if len(baseCSV) < 2 {
		panic("base file has no data")
	}
	baseHeader := baseCSV[0]
	baseKeyIdx, baseKeyName := findJoinKeyIndex(baseHeader, keyCandidates)
	if baseKeyIdx == -1 {
		panic(fmt.Errorf("no matching join key in %s", files[0]))
	}

	// Build base map
	baseData := map[string]map[string]string{}
	for _, row := range baseCSV[1:] {
		if baseKeyIdx >= len(row) {
			continue
		}
		key := row[baseKeyIdx]
		entry := map[string]string{}
		for i, val := range row {
			entry[baseHeader[i]] = val
		}
		baseData[key] = entry
	}

	// Merge in all other files
	for _, file := range files[1:] {
		csvData, err := loadCSV(file)
		if err != nil {
			panic(err)
		}
		if len(csvData) < 2 {
			continue
		}
		header := csvData[0]
		keyIdx, keyName := findJoinKeyIndex(header, keyCandidates)
		if keyIdx == -1 {
			panic(fmt.Errorf("no matching join key in %s", file))
		}

		for _, row := range csvData[1:] {
			if keyIdx >= len(row) {
				continue
			}
			key := row[keyIdx]
			if baseEntry, ok := baseData[key]; ok {
				for i, val := range row {
					col := header[i]
					if normalize(col) == normalize(keyName) {
						continue // skip duplicate key
					}
					if _, exists := baseEntry[col]; !exists {
						baseEntry[col] = val
					}
				}
			} else {
				// remove keys not found in subsequent files
				delete(baseData, key)
			}
		}
	}

	// Collect all headers from merged data
	nonEmptyColumns := map[string]bool{}
	for _, row := range baseData {
		for k, v := range row {
			if strings.TrimSpace(v) != "" {
				nonEmptyColumns[k] = true
			}
		}
	}

	// Reorder: key column first
	finalHeaders := []string{}
	if nonEmptyColumns[baseKeyName] {
		finalHeaders = append(finalHeaders, baseKeyName)
	}
	for h := range nonEmptyColumns {
		if normalize(h) != normalize(baseKeyName) {
			finalHeaders = append(finalHeaders, h)
		}
	}

	// Write output
	outF, err := os.Create(*output)
	if err != nil {
		panic(err)
	}
	defer outF.Close()
	writer := csv.NewWriter(outF)
	defer writer.Flush()

	writer.Write(finalHeaders)
	for _, row := range baseData {
		// Skip if key is empty or missing
		keyVal, hasKey := row[baseKeyName]
		if !hasKey || strings.TrimSpace(keyVal) == "" {
			continue
		}
		// Check if all values are empty
		allEmpty := true
		for _, col := range finalHeaders {
			if val, ok := row[col]; ok && strings.TrimSpace(val) != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			continue
		}
		// Write valid row
		line := make([]string, len(finalHeaders))
		for i, col := range finalHeaders {
			line[i] = row[col]
		}
		writer.Write(line)
	}

	fmt.Printf("✅ Wrote %s with %d rows\n", *output, len(baseData))
}
