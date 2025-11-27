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
	// Parse flags (matching ls command line options)
	showAll := flag.Bool("a", false, "include directory entries whose names begin with a dot (.)")
	showAlmostAll := flag.Bool("A", false, "include directory entries whose names begin with a dot (.) except for . and ..")
	longFormat := flag.Bool("l", false, "use a long listing format")
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

	// Filter entries based on -a and -A flags
	var filteredEntries []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			if *showAll {
				// -a: show all entries including . and ..
				filteredEntries = append(filteredEntries, entry)
			} else if *showAlmostAll {
				// -A: show dot files except . and ..
				if name != "." && name != ".." {
					filteredEntries = append(filteredEntries, entry)
				}
			}
			// Otherwise skip hidden files
		} else {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	// Sort entries based on flags
	sortedEntries := make([]os.DirEntry, len(filteredEntries))
	copy(sortedEntries, filteredEntries)

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

	// Build base flags for commands (preserve -a/-A and -l flags)
	baseFlags := ""
	if *showAll {
		baseFlags += " -a"
	} else if *showAlmostAll {
		baseFlags += " -A"
	}
	if *longFormat {
		baseFlags += " -l"
	}

	// Start HTML mode
	fmt.Print(htmlStart)

	// Build HTML output
	var html strings.Builder

	if *longFormat {
		// Table view with clickable headers
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
.lsh-table {
	width: 100%;
	border-collapse: collapse;
	margin: 20px 0;
	font-size: 13px;
}
.lsh-table th {
	text-align: left;
	padding: 10px 12px;
	background-color: #2d2d2d;
	border-bottom: 2px solid #404040;
	color: #888;
	font-weight: 500;
	cursor: pointer;
	user-select: none;
	transition: all 0.2s;
}
.lsh-table th:hover {
	background-color: #363636;
	color: #61afef;
}
.lsh-table th.sorted {
	color: #61afef;
}
.lsh-table th .sort-arrow {
	margin-left: 6px;
	opacity: 0.5;
}
.lsh-table th.sorted .sort-arrow {
	opacity: 1;
}
.lsh-table td {
	padding: 8px 12px;
	border-bottom: 1px solid #333;
}
.lsh-table tr:hover td {
	background-color: #2a2a2a;
}
.lsh-table .col-mode {
	font-family: monospace;
	color: #888;
}
.lsh-table .col-size {
	text-align: right;
	color: #98c379;
	font-family: monospace;
}
.lsh-table .col-date {
	color: #e5c07b;
	white-space: nowrap;
}
.lsh-table .col-name {
	color: #61afef;
	font-weight: 500;
}
.lsh-table .col-name.dir {
	color: #c678dd;
}
.lsh-table .col-name.dir:before {
	content: '📁 ';
}
.lsh-table .col-name.file:before {
	content: '📄 ';
}
</style>

<div class="lsh-header">
	<div class="lsh-title">` + htmlEscape(absDir) + `</div>
	<div class="lsh-meta">
		<span class="lsh-meta-label">$</span>` + htmlEscape(cmdLine) + `
	</div>
</div>

<table class="lsh-table">
	<thead>
		<tr>
			<th>Mode</th>
			<th class="` + getSortedClass(*sortSize, *sortReverse) + `" onclick="runCommand(&quot;` + htmlEscape(getSortCommand(exePath, baseFlags, "-S", absDir, *sortSize, *sortReverse)) + `&quot;)">
				Size<span class="sort-arrow">` + getSortArrow(*sortSize, *sortReverse) + `</span>
			</th>
			<th class="` + getSortedClass(*sortTime, *sortReverse) + `" onclick="runCommand(&quot;` + htmlEscape(getSortCommand(exePath, baseFlags, "-t", absDir, *sortTime, *sortReverse)) + `&quot;)">
				Modified<span class="sort-arrow">` + getSortArrow(*sortTime, *sortReverse) + `</span>
			</th>
			<th class="` + getSortedClass(!*sortSize && !*sortTime, *sortReverse) + `" onclick="runCommand(&quot;` + htmlEscape(getSortCommand(exePath, baseFlags, "", absDir, !*sortSize && !*sortTime, *sortReverse)) + `&quot;)">
				Name<span class="sort-arrow">` + getSortArrow(!*sortSize && !*sortTime, *sortReverse) + `</span>
			</th>
		</tr>
	</thead>
	<tbody>`)

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
			modTime := info.ModTime().Format("Jan _2 15:04")

			html.WriteString(fmt.Sprintf(`
		<tr>
			<td class="col-mode">%s</td>
			<td class="col-size">%s</td>
			<td class="col-date">%s</td>
			<td class="col-name %s">%s</td>
		</tr>`,
				htmlEscape(mode),
				htmlEscape(size),
				htmlEscape(modTime),
				itemType,
				htmlEscape(entry.Name()),
			))
		}

		html.WriteString(`
	</tbody>
</table>`)
	} else {
		// Grid view (original)
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
		<a href="#" class="lsh-sort-btn" onclick="runCommand(&quot;` + htmlEscape(shellQuote(exePath)+baseFlags+" "+shellQuote(absDir)) + `&quot;); return false;">Name</a>
		<a href="#" class="lsh-sort-btn" onclick="runCommand(&quot;` + htmlEscape(shellQuote(exePath)+baseFlags+" -t "+shellQuote(absDir)) + `&quot;); return false;">Date</a>
		<a href="#" class="lsh-sort-btn" onclick="runCommand(&quot;` + htmlEscape(shellQuote(exePath)+baseFlags+" -S "+shellQuote(absDir)) + `&quot;); return false;">Size</a>
		<a href="#" class="lsh-sort-btn" onclick="runCommand(&quot;` + htmlEscape(shellQuote(exePath)+baseFlags+" -r "+shellQuote(absDir)) + `&quot;); return false;">Reverse</a>
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
	}

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

func getSortedClass(isActive bool, isReverse bool) string {
	if isActive {
		return "sorted"
	}
	return ""
}

func getSortArrow(isActive bool, isReverse bool) string {
	if !isActive {
		return "↕"
	}
	if isReverse {
		return "↑"
	}
	return "↓"
}

// getSortCommand generates the command for a column header click.
// If the column is already active, it toggles the reverse flag.
// Otherwise, it switches to that column's sort (without reverse).
func getSortCommand(exePath, baseFlags, sortFlag, absDir string, isActive, currentReverse bool) string {
	cmd := shellQuote(exePath) + baseFlags
	if sortFlag != "" {
		cmd += " " + sortFlag
	}
	// Toggle reverse if clicking on already-active column
	if isActive && !currentReverse {
		cmd += " -r"
	}
	cmd += " " + shellQuote(absDir)
	return cmd
}
