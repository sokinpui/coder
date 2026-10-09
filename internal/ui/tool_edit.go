package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/sokinpui/coder/internal/types"
)

type EditToolRenderer struct {
	DefaultToolRenderer
}

func (e *EditToolRenderer) RenderCall(call types.ToolCall, viewportWidth int) string {
	path := ExtractToolJSONField(call.Arguments, "path")
	prefix := ToolCallStyle.Render("⚡ edit")
	if path == "" {
		return e.DefaultToolRenderer.RenderCall(call, viewportWidth)
	}
	return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(TruncateSingleLine(path, 60)))
}

func (e *EditToolRenderer) RenderExpandedCall(call types.ToolCall, viewportWidth int) string {
	path := ExtractToolJSONField(call.Arguments, "path")
	oldStr := ExtractToolJSONField(call.Arguments, "old_string")
	newStr := ExtractToolJSONField(call.Arguments, "new_string")

	header := fmt.Sprintf("%s %s", ToolCallStyle.Render("⚡ edit"), ToolMutedStyle.Render(path))
	if oldStr == "" && newStr == "" {
		return header
	}

	return renderDiffCard(path, oldStr, newStr, viewportWidth)
}

func (e *EditToolRenderer) RenderConfirmPrompt(arguments string, width int) string {
	path := ExtractToolJSONField(arguments, "path")
	oldStr := ExtractToolJSONField(arguments, "old_string")
	newStr := ExtractToolJSONField(arguments, "new_string")

	header := ToolMutedStyle.Render("Target file: ") + path
	if oldStr == "" && newStr == "" {
		return header
	}

	return renderDiffCard(path, oldStr, newStr, width)
}

type editDiffKind int

const (
	diffKindContext editDiffKind = iota
	diffKindDelete
	diffKindAdd
)

type editDiffLine struct {
	kind editDiffKind
	text string
}

func renderDiffCard(path, oldStr, newStr string, width int) string {
	lines := computeAlignedDiff(oldStr, newStr)
	if len(lines) == 0 {
		return fmt.Sprintf("%s %s", ToolCallStyle.Render("⚡ edit"), ToolMutedStyle.Render(path))
	}

	var adds, dels int
	for _, l := range lines {
		if l.kind == diffKindAdd {
			adds++
		} else if l.kind == diffKindDelete {
			dels++
		}
	}

	statsStr := fmt.Sprintf("(%s %s)",
		ToolSuccessStyle.Render(fmt.Sprintf("+%d", adds)),
		ToolErrorStyle.Render(fmt.Sprintf("-%d", dels)),
	)

	header := fmt.Sprintf("%s %s %s",
		ToolCallStyle.Render("⚡ edit"),
		ToolMutedStyle.Render(path),
		statsStr,
	)

	cardWidth := max(30, width-4)
	contentWidth := max(20, cardWidth-PreviewBoxStyle.GetHorizontalFrameSize())

	var diffRows []string
	for _, l := range lines {
		gutter := "  "
		style := ToolMutedStyle

		switch l.kind {
		case diffKindAdd:
			gutter = "+ "
			style = ToolSuccessStyle
		case diffKindDelete:
			gutter = "- "
			style = ToolErrorStyle
		case diffKindContext:
			gutter = "  "
			style = ToolMutedStyle
		}

		rowText := gutter + l.text
		if ansi.StringWidth(rowText) > contentWidth {
			rowText = ansi.Truncate(rowText, contentWidth, "…")
		}
		diffRows = append(diffRows, style.Render(rowText))
	}

	cardBody := strings.Join(diffRows, "\n")
	styledBox := PreviewBoxStyle.Width(cardWidth).Render(cardBody)

	return fmt.Sprintf("%s\n%s", header, styledBox)
}

func computeAlignedDiff(oldStr, newStr string) []editDiffLine {
	oldLines := splitEditLines(oldStr)
	newLines := splitEditLines(newStr)

	var prefix []editDiffLine
	for len(oldLines) > 0 && len(newLines) > 0 && oldLines[0] == newLines[0] {
		prefix = append(prefix, editDiffLine{kind: diffKindContext, text: oldLines[0]})
		oldLines = oldLines[1:]
		newLines = newLines[1:]
	}

	var suffix []editDiffLine
	for len(oldLines) > 0 && len(newLines) > 0 && oldLines[len(oldLines)-1] == newLines[len(newLines)-1] {
		suffix = append([]editDiffLine{{kind: diffKindContext, text: oldLines[len(oldLines)-1]}}, suffix...)
		oldLines = oldLines[:len(oldLines)-1]
		newLines = newLines[:len(newLines)-1]
	}

	middle := diffMiddleBlock(oldLines, newLines)
	result := append(prefix, middle...)
	return append(result, suffix...)
}

func diffMiddleBlock(oldLines, newLines []string) []editDiffLine {
	n := len(oldLines)
	m := len(newLines)
	if n == 0 && m == 0 {
		return nil
	}

	if n*m > 160000 {
		var fallback []editDiffLine
		for _, l := range oldLines {
			fallback = append(fallback, editDiffLine{kind: diffKindDelete, text: l})
		}
		for _, l := range newLines {
			fallback = append(fallback, editDiffLine{kind: diffKindAdd, text: l})
		}
		return fallback
	}

	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var result []editDiffLine
	i, j := n, m
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			result = append(result, editDiffLine{kind: diffKindContext, text: oldLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			result = append(result, editDiffLine{kind: diffKindAdd, text: newLines[j-1]})
			j--
		} else if i > 0 && (j == 0 || dp[i][j-1] < dp[i-1][j]) {
			result = append(result, editDiffLine{kind: diffKindDelete, text: oldLines[i-1]})
			i--
		}
	}

	slices.Reverse(result)
	return result
}

func splitEditLines(text string) []string {
	if text == "" {
		return nil
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return []string{""}
	}
	return strings.Split(text, "\n")
}

func (e *EditToolRenderer) RenderExpandedResult(output string, callID string, viewportWidth int) string {
	if strings.HasPrefix(strings.TrimSpace(output), "Error:") {
		return fmt.Sprintf("↳ %s %s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(strings.TrimSpace(output)))
	}
	return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render(strings.TrimSpace(output)))
}

func init() {
	DefaultToolRegistry.Register("edit", &EditToolRenderer{})
}
