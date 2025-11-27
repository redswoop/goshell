// Package styles provides shared HTML/CSS styling for shell commands
package styles

import (
	"fmt"
	"strings"
)

// ColumnAlign specifies text alignment for a column
type ColumnAlign int

const (
	AlignLeft ColumnAlign = iota
	AlignRight
	AlignCenter
)

// Column defines a column in the tree table
type Column struct {
	Name      string      // Display name (for header, if shown)
	Class     string      // CSS class for this column
	Width     string      // CSS width (e.g., "60px", "auto", "")
	MinWidth  string      // CSS min-width
	Align     ColumnAlign // Text alignment
	FlexGrow  bool        // Whether this column should grow to fill space
	FlexShrink bool       // Whether this column can shrink
}

// TreeNode represents a node in the tree table
type TreeNode struct {
	ID          string            // Unique identifier for this node
	Icon        string            // Icon to display (emoji)
	IsDir       bool              // Whether this is a directory (affects name styling)
	Expandable  bool              // Whether this node can be expanded
	Expanded    bool              // Initial expanded state
	Cells       []string          // Cell content for each column (HTML-safe)
	Children    []*TreeNode       // Child nodes
	OnClick     string            // Optional onclick handler for the row
	BarPercent  float64           // Percentage for bar visualization (0-100)
}

// TreeTableConfig configures the tree table component
type TreeTableConfig struct {
	Columns       []Column    // Column definitions
	ShowBar       bool        // Show percentage bar (like duh)
	BarAfterCell  int         // Insert bar after this cell index (-1 to disable)
	TogglePrefix  string      // ID prefix for toggle elements (e.g., "duh", "lsh")
}

// TreeTableCSS returns CSS for the tree table component
func TreeTableCSS() string {
	return fmt.Sprintf(`
.tree-table {
	margin: 0;
	padding: 0;
	list-style: none;
}
.tree-table ul {
	margin: 0;
	padding: 0 0 0 14px;
	list-style: none;
}
.tree-row {
	display: flex;
	align-items: center;
	padding: 1px 4px;
	border-radius: 2px;
	cursor: default;
}
.tree-row:hover {
	background-color: %s;
}
.tree-toggle {
	width: 14px;
	display: inline-block;
	text-align: center;
	cursor: pointer;
	color: %s;
	font-size: 10px;
	user-select: none;
	flex-shrink: 0;
}
.tree-toggle:hover {
	color: %s;
}
.tree-toggle.empty {
	visibility: hidden;
}
.tree-icon {
	margin-right: 4px;
	flex-shrink: 0;
	font-size: 11px;
}
.tree-cell {
	color: %s;
}
.tree-cell.name {
	flex: 1;
	min-width: 0;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}
.tree-cell.name.dir {
	color: %s;
}
.tree-cell.size {
	margin-left: 8px;
	color: %s;
	white-space: nowrap;
	flex-shrink: 0;
	min-width: 60px;
	text-align: right;
}
.tree-cell.date {
	margin-left: 8px;
	color: %s;
	white-space: nowrap;
	flex-shrink: 0;
}
.tree-cell.mode {
	margin-left: 8px;
	color: %s;
	white-space: nowrap;
	flex-shrink: 0;
	min-width: 90px;
}
.tree-bar-container {
	width: 40px;
	height: 4px;
	background-color: %s;
	border-radius: 2px;
	margin-left: 8px;
	overflow: hidden;
	flex-shrink: 0;
}
.tree-bar {
	height: 100%%;
	background: linear-gradient(90deg, %s, %s);
	border-radius: 2px;
}
.tree-children {
	display: none;
}
.tree-children.expanded {
	display: block;
}
`, Colors.BgHover, Colors.TextGray, Colors.Blue, Colors.Blue,
		Colors.Purple, Colors.Green, Colors.Yellow, Colors.TextGray,
		Colors.BgDark, Colors.Green, Colors.Blue)
}

// RenderTreeTable renders a tree table to HTML
func RenderTreeTable(nodes []*TreeNode, config TreeTableConfig) string {
	var html strings.Builder

	html.WriteString(`<ul class="tree-table">`)
	for _, node := range nodes {
		renderTreeNode(&html, node, config, 0)
	}
	html.WriteString(`</ul>`)

	return html.String()
}

var treeNodeCounter int

func renderTreeNode(html *strings.Builder, node *TreeNode, config TreeTableConfig, depth int) {
	treeNodeCounter++
	id := treeNodeCounter
	if node.ID != "" {
		// Use provided ID if available
	}

	prefix := config.TogglePrefix
	if prefix == "" {
		prefix = "tree"
	}

	html.WriteString(`<li>`)
	html.WriteString(`<div class="tree-row"`)
	if node.OnClick != "" {
		html.WriteString(fmt.Sprintf(` onclick="%s"`, node.OnClick))
	}
	html.WriteString(`>`)

	// Toggle button
	if node.Expandable && len(node.Children) > 0 {
		expandedChar := "▶"
		if node.Expanded {
			expandedChar = "▼"
		}
		html.WriteString(fmt.Sprintf(
			`<span id="%s-toggle-%d" class="tree-toggle" onclick="var c=document.getElementById('%s-children-%d');var t=this;if(c.classList.contains('expanded')){c.classList.remove('expanded');t.textContent='▶';}else{c.classList.add('expanded');t.textContent='▼';}">%s</span>`,
			prefix, id, prefix, id, expandedChar))
	} else {
		html.WriteString(`<span class="tree-toggle empty"></span>`)
	}

	// Icon
	if node.Icon != "" {
		html.WriteString(fmt.Sprintf(`<span class="tree-icon">%s</span>`, node.Icon))
	}

	// Cells
	for i, cell := range node.Cells {
		var class string
		if i < len(config.Columns) {
			class = config.Columns[i].Class
		}

		// Add dir class for name column if this is a directory
		if class == "name" && node.IsDir {
			class = "name dir"
		}

		html.WriteString(fmt.Sprintf(`<span class="tree-cell %s">%s</span>`, class, cell))

		// Insert bar after the specified cell if configured
		if config.ShowBar && i == config.BarAfterCell {
			html.WriteString(fmt.Sprintf(`<div class="tree-bar-container"><div class="tree-bar" style="width: %.1f%%"></div></div>`, node.BarPercent))
		}
	}

	html.WriteString(`</div>`)

	// Children
	if len(node.Children) > 0 {
		expandedClass := ""
		if node.Expanded {
			expandedClass = " expanded"
		}
		html.WriteString(fmt.Sprintf(`<ul id="%s-children-%d" class="tree-children%s">`, prefix, id, expandedClass))
		for _, child := range node.Children {
			renderTreeNode(html, child, config, depth+1)
		}
		html.WriteString(`</ul>`)
	}

	html.WriteString(`</li>`)
}

// ResetTreeNodeCounter resets the node counter (call before rendering a new tree)
func ResetTreeNodeCounter() {
	treeNodeCounter = 0
}
