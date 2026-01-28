package main

import (
	"testing"
)

func TestGenarate(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("OPENAI_BASE_URL", "")
	SetupApp()
	initAIs()
}
