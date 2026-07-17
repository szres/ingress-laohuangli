package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestGenarate(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("OPENAI_BASE_URL", "")
	SetupApp()
	initAIs()
}

func TestGetPromptConstraints(t *testing.T) {
	aiStoreMu.Lock()
	aiStore = AIContentStore{
		Recent: []AIRecentResult{{Text: "最近生成的词条A", Hour: "2026-07-15 10"}},
		Good:   []string{"管理员标注的好词条"},
		Bad:    []string{"管理员标注的坏词条"},
	}
	aiStoreMu.Unlock()

	p := getPrompt(time.Now())
	for _, want := range []string{
		"至少 4 行完全不含 Ingress 元素", // 每小时 10 条时非 Ingress 下限
		"Ingress 术语表",
		"XMP Burster",
		"最长不超过 12 个词",
		"时间贴合",
		"严禁时间错位",
		"节假日",
		"{管理员标注的好词条}",
		"{管理员标注的坏词条}",
		"{最近生成的词条A}",
		"禁止重复或高度相似",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt 缺少 %q", want)
		}
	}
}

func TestGetPromptSampleCaps(t *testing.T) {
	store := AIContentStore{}
	for i := 0; i < 100; i++ {
		store.Good = append(store.Good, fmt.Sprintf("标注好词条%03d", i))
		store.Bad = append(store.Bad, fmt.Sprintf("标注坏词条%03d", i))
	}
	aiStoreMu.Lock()
	aiStore = store
	aiStoreMu.Unlock()

	p := getPrompt(time.Now())
	if n := strings.Count(p, "标注好词条"); n != aiGoodSampleCap {
		t.Errorf("注入了 %d 条标注 good，want %d", n, aiGoodSampleCap)
	}
	if n := strings.Count(p, "标注坏词条"); n != aiBadSampleCap {
		t.Errorf("注入了 %d 条标注 bad，want %d", n, aiBadSampleCap)
	}
	for _, sample := range defaultBadSamples {
		if !strings.Contains(p, "{"+sample+"}") {
			t.Errorf("默认 bad 样本 %q 未注入", sample)
		}
	}
}
