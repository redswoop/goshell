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

type dirEntry struct {
	path     string
	name     string
	size     int64
	isDir    bool
	children []*dirEntry
}

func main() {
	maxDepth := flag.Int("d", -1, "max depth to traverse (-1 for unlimited)")
	showAll := flag.Bool("a", false, "include hidden files")
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	// Build the tree and calculate sizes
	root := buildTree(absDir, *maxDepth, *showAll, 0)
	if root == nil {
		fmt.Fprintf(os.Stderr, "duh: cannot access '%s'\n", dir)
		os.Exit(1)
	}

	// Render HTML
	fmt.Print(styles.HTMLStart)
	renderHTML(root, absDir)
	os.Stdout.Sync()
	fmt.Println(styles.HTMLEnd)
	os.Stdout.Sync()
}

func buildTree(path string, maxDepth int, showAll bool, currentDepth int) *dirEntry {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	entry := &dirEntry{
		path:  path,
		name:  filepath.Base(path),
		isDir: info.IsDir(),
	}

	if !info.IsDir() {
		entry.size = info.Size()
		return entry
	}

	// It's a directory - read contents
	entries, err := os.ReadDir(path)
	if err != nil {
		return entry
	}

	// Check depth limit
	if maxDepth >= 0 && currentDepth >= maxDepth {
		// Just calculate size without building children
		entry.size = calcDirSize(path, showAll)
		return entry
	}

	var totalSize int64
	for _, e := range entries {
		name := e.Name()
		if !showAll && strings.HasPrefix(name, ".") {
			continue
		}

		childPath := filepath.Join(path, name)
		child := buildTree(childPath, maxDepth, showAll, currentDepth+1)
		if child != nil {
			entry.children = append(entry.children, child)
			totalSize += child.size
		}
	}

	// Sort children by size (largest first)
	sort.Slice(entry.children, func(i, j int) bool {
		return entry.children[i].size > entry.children[j].size
	})

	entry.size = totalSize
	return entry
}

func calcDirSize(path string, showAll bool) int64 {
	var size int64
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !showAll && strings.HasPrefix(info.Name(), ".") && p != path {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func renderHTML(root *dirEntry, absDir string) {
	var html strings.Builder

	html.WriteString(`<style>`)
	html.WriteString(styles.BaseCSS())
	html.WriteString(`
.duh-total {
	font-size: 14px;
	font-weight: 700;
	color: ` + styles.Colors.Green + `;
}
.duh-total-label {
	font-size: 12px;
	color: ` + styles.Colors.TextGray + `;
	font-weight: 400;
}
</style>
<div class="shell-container">
<div class="shell-header">
<div class="shell-title">` + styles.HTMLEscape(absDir) + `</div>
<div class="duh-total">` + styles.FormatSize(root.size) + ` <span class="duh-total-label">total</span></div>
</div>
<ul class="shell-list">
`)

	// Render children of root (not the root itself since we show it in header)
	for _, child := range root.children {
		renderNode(&html, child, root.size, 0)
	}

	html.WriteString(`
</ul>
</div>
`)

	fmt.Print(html.String())
}

var nodeID int

func renderNode(html *strings.Builder, entry *dirEntry, parentSize int64, depth int) {
	nodeID++
	id := nodeID

	// Calculate percentage of parent
	var pct float64
	if parentSize > 0 {
		pct = float64(entry.size) / float64(parentSize) * 100
	}

	hasChildren := len(entry.children) > 0

	html.WriteString(`<li>`)
	html.WriteString(`<div class="shell-row">`)

	// Toggle button
	if hasChildren {
		html.WriteString(fmt.Sprintf(`<span id="duh-toggle-%d" class="shell-toggle" onclick="var c=document.getElementById('duh-children-%d');var t=this;if(c.classList.contains('expanded')){c.classList.remove('expanded');t.textContent='▶';}else{c.classList.add('expanded');t.textContent='▼';}">▶</span>`, id, id))
	} else {
		html.WriteString(`<span class="shell-toggle empty"></span>`)
	}

	// Icon
	if entry.isDir {
		html.WriteString(`<span class="shell-icon">📁</span>`)
	} else {
		html.WriteString(`<span class="shell-icon">📄</span>`)
	}

	// Name
	nameClass := "shell-name"
	if entry.isDir {
		nameClass += " dir"
	}
	html.WriteString(fmt.Sprintf(`<span class="%s">%s</span>`, nameClass, styles.HTMLEscape(entry.name)))

	// Size bar
	html.WriteString(fmt.Sprintf(`<div class="shell-bar-container"><div class="shell-bar" style="width: %.1f%%"></div></div>`, pct))

	// Size
	html.WriteString(fmt.Sprintf(`<span class="shell-size">%s</span>`, styles.FormatSize(entry.size)))

	html.WriteString(`</div>`)

	// Children
	if hasChildren {
		html.WriteString(fmt.Sprintf(`<ul id="duh-children-%d" class="shell-children">`, id))
		for _, child := range entry.children {
			renderNode(html, child, entry.size, depth+1)
		}
		html.WriteString(`</ul>`)
	}

	html.WriteString(`</li>`)
}
