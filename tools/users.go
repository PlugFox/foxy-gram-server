// Extract users from history file

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	// Чтение файла и обработка ошибок
	raw, err := os.ReadFile("result.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file: %v\n", err)
		os.Exit(1)
	}

	// Парсинг JSON
	var result struct {
		Messages []struct {
			Type   string      `json:"type"`
			FromID interface{} `json:"from_id"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(raw, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse JSON: %v\n", err)
		os.Exit(1)
	}

	// Сбор уникальных ID
	ids := make(map[int]struct{})
	for _, msg := range result.Messages {
		if msg.Type != "message" {
			continue
		}

		switch v := msg.FromID.(type) {
		case float64: // ID в числовом формате
			ids[int(v)] = struct{}{}
		case string: // ID в строковом формате, начинающемся с "user"
			if strings.HasPrefix(v, "user") {
				if id, err := strconv.Atoi(v[4:]); err == nil {
					ids[id] = struct{}{}
				}
			}
		}
	}

	// Запись уникальных ID в файл
	if err := writeIDsToFile("ids.txt", ids); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write IDs to file: %v\n", err)
		os.Exit(1)
	}
}

// writeIDsToFile записывает ID в файл, по одному на строку.
func writeIDsToFile(filename string, ids map[int]struct{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	for id := range ids {
		if _, err := fmt.Fprintf(file, "%d\n", id); err != nil {
			return fmt.Errorf("write to file: %w", err)
		}
	}

	return nil
}
