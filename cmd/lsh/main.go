package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"shellserver/internal/styles"
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
	fmt.Print(styles.HTMLStart)

	// Build HTML output
	var html strings.Builder

	// Shared styles + lsh-specific
	html.WriteString(`<style>`)
	html.WriteString(styles.BaseCSS())
	html.WriteString(`
.lsh-grid {
	display: flex;
	flex-wrap: wrap;
	gap: 2px 12px;
	margin: 8px 0;
}
.lsh-item {
	display: inline-flex;
	align-items: center;
	padding: 1px 4px;
	border-radius: 2px;
	white-space: nowrap;
}
.lsh-item:hover {
	background-color: ` + styles.Colors.BgHover + `;
}
.lsh-item .shell-icon {
	font-size: 10px;
}
.lsh-item .shell-name {
	font-size: 12px;
}
.lsh-item .lsh-size {
	font-size: 10px;
	color: ` + styles.Colors.TextGray + `;
	margin-left: 4px;
}
</style>
<div class="shell-container">
<div class="shell-header">
<div class="shell-title">` + styles.HTMLEscape(absDir) + `</div>
<div class="shell-meta">
<span class="shell-meta-label">$</span>` + styles.HTMLEscape(cmdLine) + `
</div>
<div class="shell-sort-buttons">
<a href="#" class="shell-sort-btn` + getActiveClass(!*sortSize && !*sortTime) + `" onclick="runCommand(&quot;` + styles.HTMLEscape(styles.ShellQuote(exePath)+baseFlags+" "+styles.ShellQuote(absDir)) + `&quot;); return false;">Name</a>
<a href="#" class="shell-sort-btn` + getActiveClass(*sortTime) + `" onclick="runCommand(&quot;` + styles.HTMLEscape(styles.ShellQuote(exePath)+baseFlags+" -t "+styles.ShellQuote(absDir)) + `&quot;); return false;">Date</a>
<a href="#" class="shell-sort-btn` + getActiveClass(*sortSize) + `" onclick="runCommand(&quot;` + styles.HTMLEscape(styles.ShellQuote(exePath)+baseFlags+" -S "+styles.ShellQuote(absDir)) + `&quot;); return false;">Size</a>
<a href="#" class="shell-sort-btn" onclick="runCommand(&quot;` + styles.HTMLEscape(styles.ShellQuote(exePath)+baseFlags+" -r "+styles.ShellQuote(absDir)) + `&quot;); return false;">↕</a>
</div>
</div>
`)

	if *longFormat {
		// Long format: use TreeTable component
		html.WriteString(`<style>`)
		html.WriteString(styles.TreeTableCSS())
		html.WriteString(`</style>`)

		// Build tree nodes from entries
		var nodes []*styles.TreeNode
		for _, entry := range sortedEntries {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			icon := "📄"
			if entry.IsDir() {
				icon = "📁"
			}

			mode := info.Mode().String()
			modTime := info.ModTime().Format("Jan _2 15:04")

			nodes = append(nodes, &styles.TreeNode{
				Icon:  icon,
				IsDir: entry.IsDir(),
				Cells: []string{
					styles.HTMLEscape(entry.Name()),
					styles.HTMLEscape(mode),
					styles.HTMLEscape(modTime),
					styles.FormatSize(info.Size()),
				},
			})
		}

		config := styles.TreeTableConfig{
			Columns: []styles.Column{
				{Class: "name"},
				{Class: "mode"},
				{Class: "date"},
				{Class: "size"},
			},
			TogglePrefix: "lsh",
		}

		styles.ResetTreeNodeCounter()
		html.WriteString(styles.RenderTreeTable(nodes, config))
	} else {
		// Default: compact wrapped grid
		html.WriteString(`<div class="lsh-grid">`)
		for _, entry := range sortedEntries {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			nameClass := "shell-name"
			icon := "📄"
			if entry.IsDir() {
				nameClass += " dir"
				icon = "📁"
			}

			html.WriteString(`<span class="lsh-item">`)
			html.WriteString(`<span class="shell-icon">` + icon + `</span>`)
			html.WriteString(fmt.Sprintf(`<span class="%s">%s</span>`, nameClass, styles.HTMLEscape(entry.Name())))
			html.WriteString(fmt.Sprintf(`<span class="lsh-size">%s</span>`, styles.FormatSize(info.Size())))
			html.WriteString(`</span>`)
		}
		html.WriteString(`</div>`)
	}

	html.WriteString(`
</div>`)

	// Output the HTML
	fmt.Print(html.String())
	os.Stdout.Sync()

	// End HTML mode
	fmt.Println(styles.HTMLEnd)
	os.Stdout.Sync()
}

func getActiveClass(isActive bool) string {
	if isActive {
		return " active"
	}
	return ""
}
