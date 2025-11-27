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
	htmlStart = "\x1b]9001;HTML_START\x07"
	htmlEnd   = "\x1b]9001;HTML_END\x07"
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
	fmt.Print(htmlStart)
	renderHTML(root, absDir)
	os.Stdout.Sync()
	fmt.Println(htmlEnd)
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

	html.WriteString(`<style>
.duh-container {
	font-family: monospace;
	font-size: 12px;
	line-height: 1.3;
}
.duh-header {
	margin-bottom: 8px;
	padding-bottom: 6px;
	border-bottom: 1px solid #404040;
}
.duh-title {
	font-size: 13px;
	color: #61afef;
}
.duh-total {
	font-size: 14px;
	font-weight: 700;
	color: #98c379;
}
.duh-total-label {
	font-size: 12px;
	color: #888;
	font-weight: 400;
}
.duh-tree {
	margin: 0;
	padding: 0;
	list-style: none;
}
.duh-tree ul {
	margin: 0;
	padding: 0 0 0 14px;
	list-style: none;
}
.duh-node {
	padding: 0;
}
.duh-row {
	display: flex;
	align-items: center;
	padding: 1px 4px;
	border-radius: 2px;
	cursor: default;
}
.duh-row:hover {
	background-color: #2a2a2a;
}
.duh-toggle {
	width: 14px;
	display: inline-block;
	text-align: center;
	cursor: pointer;
	color: #888;
	font-size: 10px;
	user-select: none;
	flex-shrink: 0;
}
.duh-toggle:hover {
	color: #61afef;
}
.duh-toggle.empty {
	visibility: hidden;
}
.duh-icon {
	margin-right: 4px;
	flex-shrink: 0;
	font-size: 11px;
}
.duh-name {
	flex: 1;
	color: #abb2bf;
}
.duh-name.dir {
	color: #c678dd;
}
.duh-size {
	margin-left: 8px;
	color: #98c379;
	white-space: nowrap;
	flex-shrink: 0;
	min-width: 60px;
	text-align: right;
}
.duh-children {
	display: none;
}
.duh-children.expanded {
	display: block;
}
.duh-bar-container {
	width: 40px;
	height: 4px;
	background-color: #2d2d2d;
	border-radius: 2px;
	margin-left: 8px;
	overflow: hidden;
	flex-shrink: 0;
}
.duh-bar {
	height: 100%;
	background: linear-gradient(90deg, #98c379, #61afef);
	border-radius: 2px;
}
</style>
<div class="duh-container">
<div class="duh-header">
<div class="duh-title">` + htmlEscape(absDir) + `</div>
<div class="duh-total">` + formatSize(root.size) + ` <span class="duh-total-label">total</span></div>
</div>
<ul class="duh-tree">
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

	html.WriteString(`<li class="duh-node">`)
	html.WriteString(`<div class="duh-row">`)

	// Toggle button - use inline JS since script tags don't execute in innerHTML
	if hasChildren {
		html.WriteString(fmt.Sprintf(`<span id="duh-toggle-%d" class="duh-toggle" onclick="var c=document.getElementById('duh-children-%d');var t=this;if(c.classList.contains('expanded')){c.classList.remove('expanded');t.textContent='▶';}else{c.classList.add('expanded');t.textContent='▼';}">▶</span>`, id, id))
	} else {
		html.WriteString(`<span class="duh-toggle empty"></span>`)
	}

	// Icon
	if entry.isDir {
		html.WriteString(`<span class="duh-icon">📁</span>`)
	} else {
		html.WriteString(`<span class="duh-icon">📄</span>`)
	}

	// Name
	nameClass := "duh-name"
	if entry.isDir {
		nameClass += " dir"
	}
	html.WriteString(fmt.Sprintf(`<span class="%s">%s</span>`, nameClass, htmlEscape(entry.name)))

	// Size bar
	html.WriteString(fmt.Sprintf(`<div class="duh-bar-container"><div class="duh-bar" style="width: %.1f%%"></div></div>`, pct))

	// Size
	html.WriteString(fmt.Sprintf(`<span class="duh-size">%s</span>`, formatSize(entry.size)))

	html.WriteString(`</div>`)

	// Children
	if hasChildren {
		html.WriteString(fmt.Sprintf(`<ul id="duh-children-%d" class="duh-children">`, id))
		for _, child := range entry.children {
			renderNode(html, child, entry.size, depth+1)
		}
		html.WriteString(`</ul>`)
	}

	html.WriteString(`</li>`)
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
