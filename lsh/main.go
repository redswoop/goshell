package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	// Custom escape sequences for HTML mode
	htmlStart = "\x1b]9001;HTML_START\x07"
	htmlEnd   = "\x1b]9001;HTML_END\x07"
)

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lsh: %v\n", err)
		os.Exit(1)
	}

	// Start HTML mode
	fmt.Print(htmlStart)

	// Build HTML output
	var html strings.Builder
	html.WriteString(`<style>
.file-list {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
	gap: 10px;
	margin: 20px 0;
}
.file-item {
	padding: 12px 16px;
	background-color: #2d2d2d;
	border-radius: 6px;
	border: 1px solid #404040;
	transition: all 0.2s;
	cursor: default;
}
.file-item:hover {
	background-color: #363636;
	border-color: #505050;
	transform: translateY(-2px);
	box-shadow: 0 4px 8px rgba(0,0,0,0.3);
}
.file-name {
	font-weight: 500;
	color: #61afef;
	word-break: break-word;
}
.file-item.dir .file-name {
	color: #c678dd;
}
.file-item.dir .file-name:before {
	content: '📁 ';
}
.file-item.file .file-name:before {
	content: '📄 ';
}
.file-meta {
	font-size: 11px;
	color: #888;
	margin-top: 6px;
}
</style>

<h2>` + dir + `</h2>
<div class="file-list">`)

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		itemType := "file"
		if entry.IsDir() {
			itemType = "dir"
		}

		size := formatSize(info.Size())
		mode := info.Mode().String()

		html.WriteString(fmt.Sprintf(`
	<div class="file-item %s">
		<div class="file-name">%s</div>
		<div class="file-meta">%s • %s</div>
	</div>`,
			itemType,
			htmlEscape(entry.Name()),
			size,
			mode,
		))
	}

	html.WriteString("\n</div>")

	// Output the HTML
	fmt.Print(html.String())

	// End HTML mode
	fmt.Print(htmlEnd)
}

func formatSize(size int64) string {
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

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
