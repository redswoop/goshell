package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// Custom escape sequences for HTML mode
	htmlStart = "\x1b]9001;HTML_START\x07"
	htmlEnd   = "\x1b]9001;HTML_END\x07"
)

func main() {
	// Parse flags
	sortTime := flag.Bool("t", false, "sort by modification time")
	sortSize := flag.Bool("S", false, "sort by size")
	sortReverse := flag.Bool("r", false, "reverse sort order")
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	// Get absolute path to this executable
	exePath, err := os.Executable()
	if err != nil {
		exePath = "lsh" // fallback
	}

	// Get absolute path to directory
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lsh: %v\n", err)
		os.Exit(1)
	}

	// Sort entries based on flags
	sortedEntries := make([]os.DirEntry, len(entries))
	copy(sortedEntries, entries)

	if *sortTime {
		sort.Slice(sortedEntries, func(i, j int) bool {
			infoI, _ := sortedEntries[i].Info()
			infoJ, _ := sortedEntries[j].Info()
			if *sortReverse {
				return infoI.ModTime().Before(infoJ.ModTime())
			}
			return infoI.ModTime().After(infoJ.ModTime())
		})
	} else if *sortSize {
		sort.Slice(sortedEntries, func(i, j int) bool {
			infoI, _ := sortedEntries[i].Info()
			infoJ, _ := sortedEntries[j].Info()
			if *sortReverse {
				return infoI.Size() < infoJ.Size()
			}
			return infoI.Size() > infoJ.Size()
		})
	} else {
		// Sort by name
		sort.Slice(sortedEntries, func(i, j int) bool {
			if *sortReverse {
				return sortedEntries[i].Name() > sortedEntries[j].Name()
			}
			return sortedEntries[i].Name() < sortedEntries[j].Name()
		})
	}

	// Build command line representation
	cmdLine := "lsh"
	if len(os.Args) > 1 {
		cmdLine = strings.Join(os.Args, " ")
	}

	// Start HTML mode
	fmt.Print(htmlStart)

	// Build HTML output
	var html strings.Builder
	html.WriteString(`<style>
.lsh-header {
	margin-bottom: 20px;
	padding-bottom: 15px;
	border-bottom: 1px solid #404040;
}
.lsh-title {
	font-size: 20px;
	font-weight: 600;
	color: #61afef;
	margin-bottom: 8px;
}
.lsh-meta {
	font-size: 12px;
	color: #888;
	margin-bottom: 10px;
}
.lsh-meta-label {
	color: #666;
	margin-right: 4px;
}
.lsh-sort-buttons {
	display: flex;
	gap: 8px;
	margin-top: 10px;
}
.lsh-sort-btn {
	background-color: #2d2d2d;
	color: #61afef;
	border: 1px solid #404040;
	padding: 6px 12px;
	border-radius: 4px;
	font-size: 11px;
	cursor: pointer;
	text-decoration: none;
	transition: all 0.2s;
	display: inline-block;
}
.lsh-sort-btn:hover {
	background-color: #363636;
	border-color: #505050;
	transform: translateY(-1px);
}
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

<div class="lsh-header">
	<div class="lsh-title">` + htmlEscape(absDir) + `</div>
	<div class="lsh-meta">
		<span class="lsh-meta-label">$</span>` + htmlEscape(cmdLine) + `
	</div>
	<div class="lsh-sort-buttons">
		<a href="#" class="lsh-sort-btn" onclick="runCommand(&quot;` + htmlEscape(shellQuote(exePath)+" "+shellQuote(absDir)) + `&quot;); return false;">Name</a>
		<a href="#" class="lsh-sort-btn" onclick="runCommand(&quot;` + htmlEscape(shellQuote(exePath)+" -t "+shellQuote(absDir)) + `&quot;); return false;">Date</a>
		<a href="#" class="lsh-sort-btn" onclick="runCommand(&quot;` + htmlEscape(shellQuote(exePath)+" -S "+shellQuote(absDir)) + `&quot;); return false;">Size</a>
		<a href="#" class="lsh-sort-btn" onclick="runCommand(&quot;` + htmlEscape(shellQuote(exePath)+" -r "+shellQuote(absDir)) + `&quot;); return false;">Reverse</a>
	</div>
</div>

<div class="file-list">`)

	for _, entry := range sortedEntries {
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
	os.Stdout.Sync()

	// End HTML mode
	fmt.Println(htmlEnd)
	os.Stdout.Sync()
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

func shellQuote(s string) string {
	// Simple shell quoting: wrap in single quotes and escape any single quotes
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
