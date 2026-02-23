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

func TestChatSizeMappingsMatchVanilla(t *testing.T) {
	if got := chatWidthPixelsForValue(1.0); got != 320 {
		t.Fatalf("chat width max mismatch: got=%d want=320", got)
	}
	if got := chatWidthPixelsForValue(0.0); got != 40 {
		t.Fatalf("chat width min mismatch: got=%d want=40", got)
	}
	if got := chatHeightPixelsForValue(1.0); got != 180 {
		t.Fatalf("chat height max mismatch: got=%d want=180", got)
	}
	if got := chatHeightPixelsForValue(0.0); got != 20 {
		t.Fatalf("chat height min mismatch: got=%d want=20", got)
	}
	a := &App{
		chatHeightFocused:   1.0,
		chatHeightUnfocused: 0.44366196,
	}
	if got := a.chatVisibleLineCount(true); got != 20 {
		t.Fatalf("focused chat line count mismatch: got=%d want=20", got)
	}
	if got := a.chatVisibleLineCount(false); got != 10 {
		t.Fatalf("unfocused chat line count mismatch: got=%d want=10", got)
	}
}

func TestChatFormattingStrip(t *testing.T) {
	in := "\u00A7aGreen \u00A7lBold\u00A7r Done"
	got := stripFormattingCodes(in)
	if got != "Green Bold Done" {
		t.Fatalf("formatting strip mismatch: got=%q want=%q", got, "Green Bold Done")
	}
}
