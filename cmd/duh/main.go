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
	html.WriteString(styles.TreeTableCSS())
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
`)

	// Build tree nodes from directory entries
	nodes := buildTreeNodes(root.children, root.size)

	config := styles.TreeTableConfig{
		Columns: []styles.Column{
			{Class: "name"},
			{Class: "size"},
		},
		ShowBar:      true,
		BarAfterCell: 0, // Insert bar after name
		TogglePrefix: "duh",
		TreeID:       "duh",
	}

	styles.ResetTreeNodeCounter()
	html.WriteString(styles.RenderTreeTable(nodes, config))

	html.WriteString(`
</div>
`)

	fmt.Print(html.String())
}

func buildTreeNodes(entries []*dirEntry, parentSize int64) []*styles.TreeNode {
	var nodes []*styles.TreeNode
	for _, entry := range entries {
		node := buildTreeNode(entry, parentSize)
		nodes = append(nodes, node)
	}
	return nodes
}

func buildTreeNode(entry *dirEntry, parentSize int64) *styles.TreeNode {
	// Calculate percentage of parent
	var pct float64
	if parentSize > 0 {
		pct = float64(entry.size) / float64(parentSize) * 100
	}

	icon := "📄"
	if entry.isDir {
		icon = "📁"
	}

	node := &styles.TreeNode{
		Icon:       icon,
		IsDir:      entry.isDir,
		Expandable: len(entry.children) > 0,
		BarPercent: pct,
		Value:      styles.HTMLEscape(styles.ShellQuote(entry.name)),
		Cells: []string{
			styles.HTMLEscape(entry.name),
			styles.FormatSize(entry.size),
		},
	}

	// Recursively build children
	if len(entry.children) > 0 {
		node.Children = buildTreeNodes(entry.children, entry.size)
	}

	return node
}
