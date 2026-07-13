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
