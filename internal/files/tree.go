package files

import (
	"coder/internal/utils"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Node represents a file or directory in the project tree.
type Node struct {
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	Type     string  `json:"type"` // "file" or "directory"
	Children []*Node `json:"children,omitempty"`
}

// GetFileTree builds a file tree of the git repository.
func GetFileTree() (*Node, error) {
	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "ls-files")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	root := &Node{Name: filepath.Base(repoRoot), Path: "", Type: "directory"}

	for _, file := range files {
		if file == "" {
			continue
		}
		parts := strings.Split(file, string(filepath.Separator))
		currentNode := root
		currentPath := ""

		for i, part := range parts {
			isLastPart := i == len(parts)-1
			if currentPath != "" {
				currentPath += string(filepath.Separator)
			}
			currentPath += part

			var foundNode *Node
			for _, child := range currentNode.Children {
				if child.Name == part {
					foundNode = child
					break
				}
			}

			if foundNode == nil {
				newNode := &Node{Name: part, Path: currentPath}
				if isLastPart {
					newNode.Type = "file"
				} else {
					newNode.Type = "directory"
				}
				currentNode.Children = append(currentNode.Children, newNode)
				foundNode = newNode
			}
			currentNode = foundNode
		}
	}

	sortNodes(root)
	return root, nil
}

// sortNodes recursively sorts children of a node. Directories first, then files, both alphabetically.
func sortNodes(node *Node) {
	if node == nil || len(node.Children) == 0 {
		return
	}

	sort.Slice(node.Children, func(i, j int) bool {
		childI := node.Children[i]
		childJ := node.Children[j]
		if childI.Type != childJ.Type {
			return childI.Type == "directory" // directories come first
		}
		return childI.Name < childJ.Name
	})

	for _, child := range node.Children {
		sortNodes(child)
	}
}
