package main

import (
	"bytes"
	"sync"
	"testing"
)

func TestContainsAltScreenExit(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{"empty", []byte{}, false},
		{"no escape", []byte("hello world"), false},
		{"xterm 1049", []byte("prefix\x1b[?1049lsuffix"), true},
		{"older xterm 47", []byte("\x1b[?47l"), true},
		{"variant 1047", []byte("data\x1b[?1047lmore"), true},
		{"enter alt screen (not exit)", []byte("\x1b[?1049h"), false},
		{"partial sequence", []byte("\x1b[?1049"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsAltScreenExit(tt.input)
			if got != tt.expected {
				t.Errorf("containsAltScreenExit(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStripHTMLMode(t *testing.T) {
	start := string(htmlStartMarker)
	end := string(htmlEndMarker)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"no markers", "hello world", "hello world"},
		{"single block", "before" + start + "html content" + end + "after", "beforeafter"},
		{"multiple blocks", "a" + start + "x" + end + "b" + start + "y" + end + "c", "abc"},
		{"incomplete start only", "before" + start + "partial", "before"},
		{"nested content", start + "<div>test</div>" + end, ""},
		{"markers in sequence", "pre" + start + end + "post", "prepost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTMLMode([]byte(tt.input))
			if string(got) != tt.expected {
				t.Errorf("stripHTMLMode(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWidgetIDFromPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantID    string
		wantError bool
	}{
		{"valid", "/widget/abc123/action", "abc123", false},
		{"valid with dashes", "/widget/my-widget-1/action", "my-widget-1", false},
		{"missing prefix", "/other/abc/action", "", true},
		{"missing action", "/widget/abc", "", true},
		{"wrong suffix", "/widget/abc/other", "", true},
		{"empty id", "/widget//action", "", true},
		{"extra segments", "/widget/abc/action/extra", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := widgetIDFromPath(tt.path)
			if tt.wantError {
				if err == nil {
					t.Errorf("widgetIDFromPath(%q) expected error, got %q", tt.path, got)
				}
			} else {
				if err != nil {
					t.Errorf("widgetIDFromPath(%q) unexpected error: %v", tt.path, err)
				}
				if got != tt.wantID {
					t.Errorf("widgetIDFromPath(%q) = %q, want %q", tt.path, got, tt.wantID)
				}
			}
		})
	}
}

func TestExtractAndStoreHTML(t *testing.T) {
	start := string(htmlStartMarker)
	end := string(htmlEndMarker)

	// Create a minimal server just for the HTML storage
	s := &ShellServer{
		htmlWidgets:   make(map[int]string),
		htmlWidgetsMu: sync.RWMutex{},
	}

	tests := []struct {
		name              string
		input             string
		wantProcessedLen  int // just check length since replacement includes dynamic ID
		wantRemainingNil  bool
		wantWidgetCount   int
		wantStoredContent string // content of first widget if any
	}{
		{
			name:             "no html",
			input:            "plain text",
			wantProcessedLen: 10,
			wantRemainingNil: true,
			wantWidgetCount:  0,
		},
		{
			name:              "single complete block",
			input:             "before" + start + "<div>hello</div>" + end + "after",
			wantProcessedLen:  -1, // variable due to link text
			wantRemainingNil:  true,
			wantWidgetCount:   1,
			wantStoredContent: "<div>hello</div>",
		},
		{
			name:             "incomplete block",
			input:            "before" + start + "partial content",
			wantProcessedLen: 6, // just "before"
			wantRemainingNil: false,
			wantWidgetCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			s.htmlWidgets = make(map[int]string)
			s.htmlCounter = 0

			processed, remaining, widgetIDs := s.extractAndStoreHTML([]byte(tt.input))

			if tt.wantProcessedLen >= 0 && len(processed) != tt.wantProcessedLen {
				t.Errorf("processed length = %d, want %d", len(processed), tt.wantProcessedLen)
			}

			if tt.wantRemainingNil && remaining != nil {
				t.Errorf("remaining = %q, want nil", remaining)
			}
			if !tt.wantRemainingNil && remaining == nil {
				t.Errorf("remaining = nil, want non-nil")
			}

			if len(widgetIDs) != tt.wantWidgetCount {
				t.Errorf("widget count = %d, want %d", len(widgetIDs), tt.wantWidgetCount)
			}

			if tt.wantStoredContent != "" && len(widgetIDs) > 0 {
				stored := s.htmlWidgets[widgetIDs[0]]
				if stored != tt.wantStoredContent {
					t.Errorf("stored content = %q, want %q", stored, tt.wantStoredContent)
				}
			}

			// Verify the processed output contains a link for each widget
			for _, id := range widgetIDs {
				if !bytes.Contains(processed, []byte("htmlwidget:")) {
					t.Errorf("processed output missing htmlwidget link for id %d", id)
				}
			}
		})
	}
}
