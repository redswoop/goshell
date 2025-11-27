// Package styles provides shared HTML/CSS styling for shell commands
package styles

import (
	"fmt"
	"strings"
)

// Colors defines the shared color palette (One Dark theme)
var Colors = struct {
	Blue      string // Primary - titles, links, interactive
	Purple    string // Directories
	Green     string // Sizes, success
	Yellow    string // Dates
	TextLight string // Regular text
	TextGray  string // Secondary/labels
	BgDark    string // Backgrounds
	BgHover   string // Hover states
	Border    string // Borders/dividers
}{
	Blue:      "#61afef",
	Purple:    "#c678dd",
	Green:     "#98c379",
	Yellow:    "#e5c07b",
	TextLight: "#abb2bf",
	TextGray:  "#888",
	BgDark:    "#2d2d2d",
	BgHover:   "#2a2a2a",
	Border:    "#404040",
}

// BaseCSS returns the shared base styles for all shell HTML output
func BaseCSS() string {
	return fmt.Sprintf(`
.shell-container {
	font-family: monospace;
	font-size: 12px;
	line-height: 1.3;
}
.shell-header {
	margin-bottom: 8px;
	padding-bottom: 6px;
	border-bottom: 1px solid %s;
}
.shell-title {
	font-size: 13px;
	color: %s;
}
.shell-meta {
	font-size: 11px;
	color: %s;
	margin-top: 2px;
}
.shell-meta-label {
	color: #666;
	margin-right: 4px;
}
.shell-row {
	display: flex;
	align-items: center;
	padding: 1px 4px;
	border-radius: 2px;
	cursor: default;
}
.shell-row:hover {
	background-color: %s;
}
.shell-icon {
	margin-right: 4px;
	flex-shrink: 0;
	font-size: 11px;
}
.shell-name {
	flex: 1;
	color: %s;
}
.shell-name.dir {
	color: %s;
}
.shell-size {
	margin-left: 8px;
	color: %s;
	white-space: nowrap;
	flex-shrink: 0;
	min-width: 60px;
	text-align: right;
}
.shell-date {
	margin-left: 8px;
	color: %s;
	white-space: nowrap;
	flex-shrink: 0;
}
.shell-mode {
	color: %s;
	white-space: nowrap;
	flex-shrink: 0;
	min-width: 90px;
}
.shell-toggle {
	width: 14px;
	display: inline-block;
	text-align: center;
	cursor: pointer;
	color: %s;
	font-size: 10px;
	user-select: none;
	flex-shrink: 0;
}
.shell-toggle:hover {
	color: %s;
}
.shell-toggle.empty {
	visibility: hidden;
}
.shell-bar-container {
	width: 40px;
	height: 4px;
	background-color: %s;
	border-radius: 2px;
	margin-left: 8px;
	overflow: hidden;
	flex-shrink: 0;
}
.shell-bar {
	height: 100%%;
	background: linear-gradient(90deg, %s, %s);
	border-radius: 2px;
}
.shell-list {
	margin: 0;
	padding: 0;
	list-style: none;
}
.shell-list ul {
	margin: 0;
	padding: 0 0 0 14px;
	list-style: none;
}
.shell-children {
	display: none;
}
.shell-children.expanded {
	display: block;
}
.shell-sort-buttons {
	display: flex;
	gap: 6px;
	margin-top: 4px;
}
.shell-sort-btn {
	background-color: %s;
	color: %s;
	border: 1px solid %s;
	padding: 2px 8px;
	border-radius: 3px;
	font-size: 11px;
	cursor: pointer;
	text-decoration: none;
	transition: background-color 0.15s;
	display: inline-block;
	font-family: monospace;
}
.shell-sort-btn:hover {
	background-color: %s;
}
.shell-sort-btn.active {
	color: %s;
	border-color: %s;
}
`, Colors.Border, Colors.Blue, Colors.TextGray, Colors.BgHover,
		Colors.Blue, Colors.Purple, Colors.Green, Colors.Yellow, Colors.TextGray,
		Colors.TextGray, Colors.Blue, Colors.BgDark, Colors.Green, Colors.Blue,
		Colors.BgDark, Colors.Blue, Colors.Border, Colors.BgHover, Colors.Green, Colors.Green)
}

// FormatSize converts bytes to human-readable format
func FormatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// HTMLEscape escapes HTML special characters
func HTMLEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// ShellQuote wraps a string in single quotes for shell safety
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// HTML markers for special output mode
const (
	HTMLStart = "\x1b]9001;HTML_START\x07"
	HTMLEnd   = "\x1b]9001;HTML_END\x07"
)
