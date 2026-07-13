package main

import (
	"errors"
	"testing"
	"time"
)

func TestParseOpenAIModels(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{"", nil},
		{"gpt-4o-mini", []string{"gpt-4o-mini"}},
		{"gpt-4o-mini, gpt-4o", []string{"gpt-4o-mini", "gpt-4o"}},
		{"a\nb\nc", []string{"a", "b", "c"}},
		{"a;b, a\nc", []string{"a", "b", "c"}},
		{"  x  ,\n y ", []string{"x", "y"}},
	}
	for _, tc := range cases {
		got := parseOpenAIModels(tc.raw)
		if len(got) != len(tc.want) {
			t.Fatalf("parseOpenAIModels(%q) len=%d want %d (%v)", tc.raw, len(got), len(tc.want), got)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("parseOpenAIModels(%q)[%d]=%q want %q", tc.raw, i, got[i], tc.want[i])
			}
		}
	}
}

// ponytail: 不打真实 API；验证首选退避时才 fallback，失败立即换下一个。
func TestOpenAIModelPriorityBackoff(t *testing.T) {
	resetOpenAIModelStates()
	models := []string{"m1", "m2", "m3"}

	openaiModelMu.Lock()
	st1 := modelStateLocked("m1")
	st1.nextRetry = time.Now().Add(time.Hour)
	openaiModelMu.Unlock()

	order := make([]string, 0)
	err := rotateTryModels(models, func(model string) error {
		order = append(order, model)
		if model == "m2" {
			return errors.New("fail m2")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(order) != 2 || order[0] != "m2" || order[1] != "m3" {
		t.Fatalf("order=%v want [m2 m3] (skip m1 in backoff, fail m2, succeed m3)", order)
	}

	openaiModelMu.Lock()
	defer openaiModelMu.Unlock()
	if !modelStateLocked("m2").nextRetry.After(time.Now()) {
		t.Fatal("m2 should be in backoff after failure")
	}
	if !modelStateLocked("m3").nextRetry.IsZero() {
		t.Fatal("m3 backoff should be cleared on success")
	}
}

func TestOpenAIModelPriorityUsesFirstHealthyModel(t *testing.T) {
	resetOpenAIModelStates()
	order := make([]string, 0)
	err := rotateTryModels([]string{"m1", "m2"}, func(model string) error {
		order = append(order, model)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(order) != 1 || order[0] != "m1" {
		t.Fatalf("order=%v want [m1]", order)
	}
}
