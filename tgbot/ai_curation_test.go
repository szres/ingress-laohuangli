package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseAIHourlyResults(t *testing.T) {
	start := time.Date(2026, 7, 13, 16, 42, 0, 0, time.Local)
	var content strings.Builder
	for hourOffset := 0; hourOffset < aiBatchHours; hourOffset++ {
		hour := hourKey(start.Add(time.Duration(hourOffset) * time.Hour))
		for entryOffset := 0; entryOffset < aiEntriesPerHour; entryOffset++ {
			fmt.Fprintf(&content, "[%s]{词条%d-%d}\n", hour, hourOffset, entryOffset)
		}
	}

	results, err := parseAIHourlyResults(content.String(), start)
	if err != nil {
		t.Fatalf("parseAIHourlyResults returned error: %v", err)
	}
	if len(results) != aiBatchHours*aiEntriesPerHour {
		t.Fatalf("got %d results, want %d", len(results), aiBatchHours*aiEntriesPerHour)
	}
}

func TestAIEntryWords(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"给Scanner贴钢化膜", 4},     // 5 汉字→3 词 + Scanner→1 词
		{"把XMP Burster当烟花", 4}, // 4 汉字→2 词 + 2 单词
		{"翘班去钓鱼", 3},            // 5 汉字→3 词
		{"看着下班人群汇聚成流幻想这是一场线下Anomaly的集结", 12},
	}
	for _, c := range cases {
		if got := aiEntryWords(c.text); got != c.want {
			t.Errorf("aiEntryWords(%q) = %d, want %d", c.text, got, c.want)
		}
	}
}

func TestParseAIHourlyResultsRejectsOverlongEntry(t *testing.T) {
	start := time.Date(2026, 7, 17, 12, 0, 0, 0, time.Local)
	overlong := strings.Repeat("长", 2*aiMaxEntryWords+1)
	var content strings.Builder
	for hourOffset := 0; hourOffset < aiBatchHours; hourOffset++ {
		hour := hourKey(start.Add(time.Duration(hourOffset) * time.Hour))
		fmt.Fprintf(&content, "[%s]{%s}\n", hour, overlong)
		for entryOffset := 1; entryOffset < aiEntriesPerHour; entryOffset++ {
			fmt.Fprintf(&content, "[%s]{词条%d-%d}\n", hour, hourOffset, entryOffset)
		}
	}

	// 每小时 10 条中有 1 条超长被过滤，只剩 9 条有效，应整体判为失败
	if _, err := parseAIHourlyResults(content.String(), start); err == nil {
		t.Fatal("超长词条应被过滤并导致该小时数量不足")
	}
}

func TestAIContentNeedsRefill(t *testing.T) {
	now := time.Date(2026, 7, 13, 16, 0, 0, 0, time.Local)
	aiContentMu.Lock()
	aiHourlyContent = map[string][]string{
		hourKey(now):           {"a"},
		hourKey(nextHour(now)): {"b"},
	}
	aiContentMu.Unlock()
	if !aiContentNeedsRefill(now) {
		t.Fatal("two current/next hour entries should trigger refill")
	}

	aiContentMu.Lock()
	aiHourlyContent[hourKey(nextHour(now))] = append(aiHourlyContent[hourKey(nextHour(now))], "c")
	aiContentMu.Unlock()
	if aiContentNeedsRefill(now) {
		t.Fatal("three current/next hour entries should not trigger refill")
	}
}

func TestAIEntryLabelsAreMutuallyExclusive(t *testing.T) {
	store := AIContentStore{Bad: []string{"好词条"}}
	if err := setAIEntryLabel(&store, "好词条", "good"); err != nil {
		t.Fatalf("set good label: %v", err)
	}
	if len(store.Good) != 1 || len(store.Bad) != 0 {
		t.Fatalf("good=%v bad=%v, want [好词条] and []", store.Good, store.Bad)
	}
	if err := setAIEntryLabel(&store, "好词条", "bad"); err != nil {
		t.Fatalf("set bad label: %v", err)
	}
	if len(store.Good) != 0 || len(store.Bad) != 1 {
		t.Fatalf("good=%v bad=%v, want [] and [好词条]", store.Good, store.Bad)
	}
}
