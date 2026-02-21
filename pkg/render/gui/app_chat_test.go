//go:build cgo

package gui

import "testing"

func TestChatHistoryNavigation(t *testing.T) {
	a := &App{
		chatHistory: []string{"one", "two", "three"},
		chatHistPos: 3,
		chatInput:   "",
		chatDraft:   "",
	}

	a.moveChatHistory(-1)
	if a.chatInput != "three" || a.chatHistPos != 2 {
		t.Fatalf("first up mismatch: input=%q pos=%d", a.chatInput, a.chatHistPos)
	}

	a.moveChatHistory(-1)
	if a.chatInput != "two" || a.chatHistPos != 1 {
		t.Fatalf("second up mismatch: input=%q pos=%d", a.chatInput, a.chatHistPos)
	}

	a.moveChatHistory(1)
	if a.chatInput != "three" || a.chatHistPos != 2 {
		t.Fatalf("down mismatch: input=%q pos=%d", a.chatInput, a.chatHistPos)
	}

	a.chatHistPos = len(a.chatHistory)
	a.chatInput = "/gamemode 1"
	a.moveChatHistory(-1)
	if a.chatInput != "three" {
		t.Fatalf("history recall mismatch: input=%q", a.chatInput)
	}
	a.moveChatHistory(1)
	if a.chatInput != "/gamemode 1" {
		t.Fatalf("draft restore mismatch: input=%q", a.chatInput)
	}
}

func TestPushChatHistoryLimit(t *testing.T) {
	a := &App{}
	for i := 0; i < 120; i++ {
		a.pushChatHistory("msg")
	}
	if len(a.chatHistory) != 100 {
		t.Fatalf("history length mismatch: got=%d want=100", len(a.chatHistory))
	}
	if a.chatHistPos != 100 {
		t.Fatalf("history cursor mismatch: got=%d want=100", a.chatHistPos)
	}
}
