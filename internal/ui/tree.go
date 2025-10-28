package ui

import (
	"coder/internal/config"
	"coder/internal/source"
	"coder/internal/utils"
	"fmt"
	"github.com/rmhubbert/bubbletea-overlay"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type treeNode struct {
	path     string
	name     string
	children []*treeNode
	isDir    bool
	expanded bool
	parent   *treeNode
}

type TreeModel struct {
	root         *treeNode
	visibleNodes []*treeNode
	selected     map[string]struct{}
	cursor       int
	Width        int
	Height       int
	Visible      bool
	viewOffset   int
	ggPressed    bool
}

func NewTree() TreeModel {
	return TreeModel{
		selected: make(map[string]struct{}),
		Visible:  false,
	}
}

func (m *TreeModel) Reset() {
	m.cursor = 0
	m.viewOffset = 0
}

func (m *TreeModel) loadInitialSelection(cfg *config.Config) {
	m.selected = make(map[string]struct{})
	for _, p := range cfg.Context.Files {
		absPath, err := filepath.Abs(p)
		if err == nil {
			m.selected[absPath] = struct{}{}
		}
	}
	for _, p := range cfg.Context.Dirs {
		absPath, err := filepath.Abs(p)
		if err == nil {
			m.selected[absPath] = struct{}{}
		}
	}
}

func (m *TreeModel) initCmd() tea.Cmd {
	return func() tea.Msg {
		rootPath, err := utils.FindRepoRoot()
		if err != nil {
			rootPath, err = os.Getwd()
			if err != nil {
				return errorMsg{fmt.Errorf("failed to get current directory: %w", err)}
			}
		}

		absRoot, err := filepath.Abs(rootPath)
		if err != nil {
			return errorMsg{fmt.Errorf("failed to get absolute path for root: %w", err)}
		}

		rootNode := &treeNode{
			path:     absRoot,
			name:     filepath.Base(absRoot),
			isDir:    true,
			expanded: true,
		}

		nodesByPath := map[string]*treeNode{absRoot: rootNode}
		exclusions := source.Exclusions
		exclusions = append(exclusions, ".git")

		walkErr := filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path == absRoot {
				return nil
			}

			// Check exclusions
			base := filepath.Base(path)
			for _, pattern := range exclusions {
				if matched, _ := filepath.Match(pattern, base); matched {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			parentPath := filepath.Dir(path)
			parent, ok := nodesByPath[parentPath]
			if !ok {
				return nil // Should not happen if we walk top-down
			}

			node := &treeNode{
				path:   path,
				name:   d.Name(),
				isDir:  d.IsDir(),
				parent: parent,
			}
			parent.children = append(parent.children, node)
			if d.IsDir() {
				nodesByPath[path] = node
			}

			return nil
		})

		if walkErr != nil {
			return errorMsg{fmt.Errorf("error walking directory: %w", walkErr)}
		}

		sortTree(rootNode)
		return treeReadyMsg{root: rootNode}
	}
}

func sortTree(node *treeNode) {
	if node == nil || !node.isDir {
		return
	}
	sort.Slice(node.children, func(i, j int) bool {
		if node.children[i].isDir != node.children[j].isDir {
			return node.children[i].isDir
		}
		return node.children[i].name < node.children[j].name
	})
	for _, child := range node.children {
		sortTree(child)
	}
}

func (m *TreeModel) buildVisibleNodes() {
	m.visibleNodes = []*treeNode{}
	var addNodes func(*treeNode)
	addNodes = func(node *treeNode) {
		m.visibleNodes = append(m.visibleNodes, node)
		if node.expanded {
			for _, child := range node.children {
				addNodes(child)
			}
		}
	}
	if m.root != nil {
		addNodes(m.root)
	}
	if m.cursor >= len(m.visibleNodes) {
		m.cursor = len(m.visibleNodes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m TreeModel) Update(msg tea.Msg) (TreeModel, tea.Cmd) {
	prevGGPressed := m.ggPressed
	m.ggPressed = false

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.Visible = false
			return m, nil
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}

		case tea.KeyDown:
			if m.cursor < len(m.visibleNodes)-1 {
				m.cursor++
			}

		case tea.KeyRunes:
			switch msg.String() {
			case "q":
				m.Visible = false
				return m, nil
			case "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "j":
				if m.cursor < len(m.visibleNodes)-1 {
					m.cursor++
				}
			case "g":
				if prevGGPressed {
					m.cursor = 0
				} else {
					m.ggPressed = true
				}
			case "G":
				if len(m.visibleNodes) > 0 {
					m.cursor = len(m.visibleNodes) - 1
				}
			case "l": // expand
				if len(m.visibleNodes) > 0 {
					node := m.visibleNodes[m.cursor]
					if node.isDir && !node.expanded {
						node.expanded = true
						m.buildVisibleNodes()
					}
				}
			case "h": // collapse
				if len(m.visibleNodes) > 0 {
					node := m.visibleNodes[m.cursor]
					if node.isDir && node.expanded {
						node.expanded = false
						m.buildVisibleNodes()
					} else if !node.isDir && node.parent != nil {
						// find parent in visible nodes and move cursor there
						for i, n := range m.visibleNodes {
							if n == node.parent {
								m.cursor = i
								break
							}
						}
					}
				}
			}

		case tea.KeyEnter:
			m.Visible = false
			var paths []string
			repoRoot := m.root.path
			for p := range m.selected {
				relPath, err := filepath.Rel(repoRoot, p)
				if err != nil {
					paths = append(paths, p) // fallback to absolute
				} else {
					paths = append(paths, relPath)
				}
			}
			return m, func() tea.Msg { return treeSelectionResultMsg{selectedPaths: paths} }
		case tea.KeySpace:
			if len(m.visibleNodes) > 0 {
				node := m.visibleNodes[m.cursor]
				if _, ok := m.selected[node.path]; ok {
					delete(m.selected, node.path)
				} else {
					m.selected[node.path] = struct{}{}
				}
				// Move to next item
				if m.cursor < len(m.visibleNodes)-1 {
					m.cursor++
				}
			}
		}
	}
	return m, nil
}

func (m *TreeModel) View() string {
	if !m.Visible || len(m.visibleNodes) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Select files/directories for context (space to select, enter to confirm):\n\n")

	// Scrolling logic
	maxItems := m.Height - 4 // account for header and padding
	if m.cursor < m.viewOffset {
		m.viewOffset = m.cursor
	}
	if m.cursor >= m.viewOffset+maxItems {
		m.viewOffset = m.cursor - maxItems + 1
	}

	end := m.viewOffset + maxItems
	if end > len(m.visibleNodes) {
		end = len(m.visibleNodes)
	}

	for i, node := range m.visibleNodes[m.viewOffset:end] {
		actualIndex := i + m.viewOffset
		var parts []string

		// Cursor
		if actualIndex == m.cursor {
			parts = append(parts, paletteSelectedItemStyle.Render("▸ "))
		} else {
			parts = append(parts, "  ")
		}

		// Selection
		if _, ok := m.selected[node.path]; ok {
			parts = append(parts, "[x] ")
		} else {
			parts = append(parts, "[ ] ")
		}

		// Indentation and icon
		depth := 0
		p := node.parent
		for p != nil {
			depth++
			p = p.parent
		}
		indent := strings.Repeat("  ", depth)
		parts = append(parts, indent)

		var icon string
		if node.isDir {
			if node.expanded {
				icon = "📁"
			} else {
				icon = "📁"
			}
		} else {
			icon = "📄"
		}
		parts = append(parts, icon+" ")

		// Name
		name := filepath.Base(node.path)
		if node.path == m.root.path {
			name = utils.ShortenPath(node.path)
		}
		parts = append(parts, name)

		line := strings.Join(parts, "")
		if actualIndex == m.cursor {
			b.WriteString(paletteSelectedItemStyle.Render(line))
		} else {
			b.WriteString(paletteItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return paletteContainerStyle.Width(m.Width).Render(b.String())
}

// TreeOverlay implements the Overlay interface for the file tree explorer.
type TreeOverlay struct{}

func (t *TreeOverlay) IsVisible(main *Model) bool {
	return main.State == stateTree
}

func (t *TreeOverlay) View(main *Model) string {
	treeWidth := main.Width * 3 / 4
	treeHeight := main.Height * 3 / 4
	if treeWidth < 60 {
		treeWidth = 60
	}
	if treeHeight < 10 {
		treeHeight = 10
	}
	main.Tree.Width = treeWidth
	main.Tree.Height = treeHeight

	treeContent := main.Tree.View()
	if treeContent == "" {
		return main.View()
	}

	treeModel := simpleModel{content: treeContent}

	overlayModel := overlay.New(
		treeModel,
		main,
		overlay.Center,
		overlay.Center,
		0,
		0,
	)
	return overlayModel.View()
}
