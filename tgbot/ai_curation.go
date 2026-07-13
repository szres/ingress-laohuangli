package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

const (
	aiHourLayout       = "2006-01-02 15"
	aiBatchHours       = 6
	aiEntriesPerHour   = 10
	aiRefillThreshold  = 3
	aiRecentResultSize = 100
)

type AIRecentResult struct {
	Text        string `json:"text"`
	Hour        string `json:"hour"`
	GeneratedAt string `json:"generated_at"`
}

type AIContentStore struct {
	Recent []AIRecentResult `json:"recent"`
	Good   []string         `json:"good"`
	Bad    []string         `json:"bad"`
}

var (
	aiContentMu     sync.Mutex
	aiHourlyContent = map[string][]string{}

	aiStoreMu sync.RWMutex
	aiStore   AIContentStore
)

func loadAIContentStore() {
	aiStoreMu.Lock()
	defer aiStoreMu.Unlock()

	aiStore = AIContentStore{}
	if err := db.Read("datas", "ai_content", &aiStore); err != nil {
		fmt.Println("AI 词条库不存在或加载失败，初始化为空")
	}
	aiStore.Recent = trimRecent(aiStore.Recent)
	aiStore.Good = uniqueAITexts(aiStore.Good)
	aiStore.Bad = uniqueAITexts(aiStore.Bad)
}

func saveAIContentStoreLocked() {
	if err := db.Write("datas", "ai_content", aiStore); err != nil {
		fmt.Println("保存 AI 词条库失败:", err)
	}
}

func trimRecent(entries []AIRecentResult) []AIRecentResult {
	if len(entries) <= aiRecentResultSize {
		return entries
	}
	return entries[len(entries)-aiRecentResultSize:]
}

func uniqueAITexts(texts []string) []string {
	out := make([]string, 0, len(texts))
	seen := make(map[string]bool, len(texts))
	for _, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" || seen[text] {
			continue
		}
		seen[text] = true
		out = append(out, text)
	}
	return out
}

func recordAIResults(entries []AIRecentResult) {
	if len(entries) == 0 {
		return
	}
	aiStoreMu.Lock()
	aiStore.Recent = trimRecent(append(aiStore.Recent, entries...))
	saveAIContentStoreLocked()
	aiStoreMu.Unlock()
}

func recentAIResults() []AIRecentResult {
	aiStoreMu.RLock()
	defer aiStoreMu.RUnlock()
	out := make([]AIRecentResult, len(aiStore.Recent))
	for i := range aiStore.Recent {
		out[len(aiStore.Recent)-1-i] = aiStore.Recent[i]
	}
	return out
}

func aiCuratedEntries(kind string) []string {
	aiStoreMu.RLock()
	defer aiStoreMu.RUnlock()
	var source []string
	if kind == "good" {
		source = aiStore.Good
	} else {
		source = aiStore.Bad
	}
	return append([]string(nil), source...)
}

func aiCuratedLabels() map[string]string {
	aiStoreMu.RLock()
	defer aiStoreMu.RUnlock()
	labels := make(map[string]string, len(aiStore.Good)+len(aiStore.Bad))
	for _, text := range aiStore.Good {
		labels[text] = "good"
	}
	for _, text := range aiStore.Bad {
		labels[text] = "bad"
	}
	return labels
}

func labelAIEntry(text, kind string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("词条不能为空")
	}
	if len([]rune(text)) > 64 {
		return fmt.Errorf("词条不能超过 64 个字符")
	}
	if kind != "good" && kind != "bad" {
		return fmt.Errorf("无效标签")
	}

	aiStoreMu.Lock()
	defer aiStoreMu.Unlock()
	if err := setAIEntryLabel(&aiStore, text, kind); err != nil {
		return err
	}
	saveAIContentStoreLocked()
	return nil
}

func setAIEntryLabel(store *AIContentStore, text, kind string) error {
	if kind == "good" {
		store.Bad = removeAIText(store.Bad, text)
		store.Good = addAIText(store.Good, text)
	} else if kind == "bad" {
		store.Good = removeAIText(store.Good, text)
		store.Bad = addAIText(store.Bad, text)
	} else {
		return fmt.Errorf("无效标签")
	}
	return nil
}

func deleteAICuratedEntry(text, kind string) bool {
	aiStoreMu.Lock()
	defer aiStoreMu.Unlock()
	var entries *[]string
	if kind == "good" {
		entries = &aiStore.Good
	} else if kind == "bad" {
		entries = &aiStore.Bad
	} else {
		return false
	}
	before := len(*entries)
	*entries = removeAIText(*entries, text)
	if before == len(*entries) {
		return false
	}
	saveAIContentStoreLocked()
	return true
}

func addAIText(texts []string, text string) []string {
	for _, candidate := range texts {
		if candidate == text {
			return texts
		}
	}
	return append(texts, text)
}

func removeAIText(texts []string, text string) []string {
	for i, candidate := range texts {
		if candidate == text {
			return append(texts[:i], texts[i+1:]...)
		}
	}
	return texts
}

func randomAIEntries(entries []string, n int) []string {
	if n > len(entries) {
		n = len(entries)
	}
	out := make([]string, 0, n)
	for _, i := range cryptoPermutation(len(entries))[:n] {
		out = append(out, entries[i])
	}
	return out
}

func cryptoPermutation(n int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			j = big.NewInt(0)
		}
		indices[i], indices[j.Int64()] = indices[j.Int64()], indices[i]
	}
	return indices
}

func hourKey(t time.Time) string {
	return t.Truncate(time.Hour).Format(aiHourLayout)
}

func nextHour(t time.Time) time.Time {
	return t.Truncate(time.Hour).Add(time.Hour)
}

func aiContentNeedsRefill(t time.Time) bool {
	aiContentMu.Lock()
	defer aiContentMu.Unlock()
	return len(aiHourlyContent[hourKey(t)])+len(aiHourlyContent[hourKey(nextHour(t))]) < aiRefillThreshold
}

func appendAIHourlyContent(entries []AIRecentResult) {
	aiContentMu.Lock()
	defer aiContentMu.Unlock()
	for _, entry := range entries {
		aiHourlyContent[entry.Hour] = append(aiHourlyContent[entry.Hour], entry.Text)
	}
}

func pruneAIHourlyContent(now time.Time) {
	current := hourKey(now)
	aiContentMu.Lock()
	defer aiContentMu.Unlock()
	for hour := range aiHourlyContent {
		if hour < current {
			delete(aiHourlyContent, hour)
		}
	}
}
