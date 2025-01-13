package main

import (
	"os"
	"testing"
)

func TestGenarate(t *testing.T) {
	os.Setenv("GEMINI_API_KEY", "test")
	os.Setenv("OPENAI_API_KEY", "test")
	initAIs()
	for _, ai := range AIs {
		ai.Init(&ai)
	}
}
