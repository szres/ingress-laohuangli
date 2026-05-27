package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type monthlyArchive map[string]laohuangliCache

var historyMu sync.RWMutex

// archiveMonthly 归档所有非当月的 history 日文件为月度归档文件
func archiveMonthly(historyDir string) {
	historyMu.Lock()
	defer historyMu.Unlock()

	currentMonth := time.Now().Format("2006-01")

	dirEntries, err := os.ReadDir(historyDir)
	if err != nil {
		fmt.Println("归档: 读取 history 目录失败:", err)
		return
	}

	monthGroups := make(map[string][]string)
	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		dateStr := strings.TrimSuffix(e.Name(), ".json")
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		month := fileDate.Format("2006-01")
		if month >= currentMonth {
			continue
		}
		monthGroups[month] = append(monthGroups[month], dateStr)
	}

	if len(monthGroups) == 0 {
		return
	}

	for month, dates := range monthGroups {
		archive := make(monthlyArchive)
		for _, dateStr := range dates {
			var cache laohuangliCache
			data, err := os.ReadFile(filepath.Join(historyDir, dateStr+".json"))
			if err != nil {
				fmt.Printf("归档: 读取 %s.json 失败: %v\n", dateStr, err)
				continue
			}
			if err := json.Unmarshal(data, &cache); err != nil {
				fmt.Printf("归档: 解析 %s.json 失败: %v\n", dateStr, err)
				continue
			}
			archive[dateStr] = cache
		}

		if len(archive) == 0 {
			continue
		}

		outPath := filepath.Join(historyDir, month+".json")
		if existingData, err := os.ReadFile(outPath); err == nil {
			var existing monthlyArchive
			if json.Unmarshal(existingData, &existing) == nil {
				for k, v := range existing {
					if _, ok := archive[k]; !ok {
						archive[k] = v
					}
				}
			}
		}

		archiveData, err := json.Marshal(archive)
		if err != nil {
			fmt.Printf("归档: 序列化 %s 失败: %v\n", month, err)
			continue
		}
		if err := os.WriteFile(outPath, archiveData, 0o644); err != nil {
			fmt.Printf("归档: 写入 %s.json 失败: %v\n", month, err)
			continue
		}

		for dateStr := range archive {
			os.Remove(filepath.Join(historyDir, dateStr+".json"))
		}
		fmt.Printf("归档: %s 完成，%d 个日文件\n", month, len(archive))
	}
}

// readHistoryEntries 读取 cutoff 之后的所有 history 条目（兼容日文件 + 月归档）
func readHistoryEntries(historyDir string, cutoff time.Time) map[string]laohuangliCache {
	historyMu.RLock()
	defer historyMu.RUnlock()

	result := make(map[string]laohuangliCache)

	dirEntries, err := os.ReadDir(historyDir)
	if err != nil {
		return result
	}

	cutoffMonth := cutoff.Format("2006-01")
	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := e.Name()
		base := strings.TrimSuffix(name, ".json")

		if _, err := time.Parse("2006-01", base); err == nil {
			if base < cutoffMonth {
				continue
			}
			var archive monthlyArchive
			data, err := os.ReadFile(filepath.Join(historyDir, name))
			if err != nil {
				continue
			}
			if err := json.Unmarshal(data, &archive); err != nil {
				continue
			}
			for dateStr, cache := range archive {
				fileDate, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					continue
				}
				if fileDate.After(cutoff) || fileDate.Equal(cutoff) {
					result[dateStr] = cache
				}
			}
			continue
		}

		fileDate, err := time.Parse("2006-01-02", base)
		if err != nil {
			continue
		}
		if fileDate.Before(cutoff) {
			continue
		}
		var cache laohuangliCache
		data, err := os.ReadFile(filepath.Join(historyDir, name))
		if err != nil {
			continue
		}
		if err := json.Unmarshal(data, &cache); err != nil {
			continue
		}
		result[base] = cache
	}

	return result
}
