package coagentui

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sokinpui/coder/internal/ui/markdown"
	"github.com/sokinpui/coder/internal/engine/coagent"
)

type readParams struct {
	Path  string `json:"path"`
	Pages string `json:"pages"`
}

type bashParams struct {
	Command string `json:"command"`
}

type writeParams struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type editParams struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

func parseEditParams(args string) editParams {
	var params editParams
	trimmed := strings.TrimSpace(args)
	_ = json.Unmarshal([]byte(trimmed), &params)
	return params
}

func parseBashParams(args string) bashParams {
	var params bashParams
	trimmed := strings.TrimSpace(args)
	if err := json.Unmarshal([]byte(trimmed), &params); err == nil && params.Command != "" {
		return params
	}
	if !strings.HasPrefix(trimmed, "{") && trimmed != "" {
		params.Command = trimmed
	}
	return params
}

func parseWriteParams(args string) writeParams {
	var params writeParams
	trimmed := strings.TrimSpace(args)
	_ = json.Unmarshal([]byte(trimmed), &params)
	return params
}

func parseReadParams(args string) readParams {
	var params readParams
	trimmed := strings.TrimSpace(args)
	if err := json.Unmarshal([]byte(trimmed), &params); err == nil && params.Path != "" {
		return params
	}
	if !strings.HasPrefix(trimmed, "{") && trimmed != "" {
		params.Path = trimmed
	}
	return params
}

func renderToolCall(call *coagent.ToolCallInfo, width int) string {
	if call == nil {
		return ""
	}
	switch call.Name {
	case "read":
		return renderReadCall(call)
	case "bash":
		return renderBashCall(call)
	case "write":
		return renderWriteCall(call, width)
	case "edit":
		return renderEditCall(call, width)
	default:
		return fmt.Sprintf("⚡ %s %s(%s)", toolCallStyle.Render("Tool:"), call.Name, call.Arguments)
	}
}

func renderReadCall(call *coagent.ToolCallInfo) string {
	params := parseReadParams(call.Arguments)
	if params.Path == "" {
		return fmt.Sprintf("⚡ %s %s(%s)", toolCallStyle.Render("Tool:"), call.Name, call.Arguments)
	}

	callHeader := fmt.Sprintf("📖 %s %s", toolCallStyle.Render("Read:"), toolPathStyle.Render(params.Path))
	if params.Pages != "" {
		callHeader += " " + toolMutedStyle.Render(fmt.Sprintf("[pages: %s]", params.Pages))
	}
	return callHeader
}

func renderBashCall(call *coagent.ToolCallInfo) string {
	params := parseBashParams(call.Arguments)
	if params.Command == "" {
		return fmt.Sprintf("⚡ %s %s(%s)", toolCallStyle.Render("Tool:"), call.Name, call.Arguments)
	}
	return fmt.Sprintf("⌨️  %s %s", toolCallStyle.Render("Exec:"), toolCommandStyle.Render("$ "+params.Command))
}

func renderWriteCall(call *coagent.ToolCallInfo, width int) string {
	params := parseWriteParams(call.Arguments)
	if params.Path == "" {
		return fmt.Sprintf("⚡ %s %s(%s)", toolCallStyle.Render("Tool:"), call.Name, call.Arguments)
	}
	lineCount := countLines(params.Content)
	byteCount := len([]byte(params.Content))
	meta := toolMutedStyle.Render(fmt.Sprintf("(%d lines, %s)", lineCount, formatBytes(byteCount)))
	header := fmt.Sprintf("📝 %s %s %s", toolCallStyle.Render("Write:"), toolPathStyle.Render(params.Path), meta)

	lang := strings.TrimPrefix(filepath.Ext(params.Path), ".")
	fence := "```"
	if lang == "md" || lang == "markdown" {
		fence = "````"
	}

	rawBlock := fmt.Sprintf("%s%s\n%s\n%s", fence, lang, strings.TrimRight(params.Content, "\n"), fence)
	renderedCode := rawBlock
	if width > 0 {
		boxWidth := max(20, width-2)
		innerWidth := max(10, boxWidth-toolBoxStyle.GetHorizontalFrameSize())
		if rendered := markdown.Render(rawBlock, innerWidth); rendered != "" {
			renderedCode = toolBoxStyle.Width(boxWidth).Render(strings.Trim(rendered, "\r\n"))
		} else {
			renderedCode = toolBoxStyle.Width(boxWidth).Render(rawBlock)
		}
	}

	return fmt.Sprintf("%s\n%s", header, renderedCode)
}

func renderEditCall(call *coagent.ToolCallInfo, width int) string {
	params := parseEditParams(call.Arguments)
	if params.Path == "" {
		return fmt.Sprintf("⚡ %s %s(%s)", toolCallStyle.Render("Tool:"), call.Name, call.Arguments)
	}

	lineCountOld := countLines(params.OldString)
	lineCountNew := countLines(params.NewString)
	meta := toolMutedStyle.Render(fmt.Sprintf("(-%d / +%d lines)", lineCountOld, lineCountNew))
	header := fmt.Sprintf("✏️  %s %s %s", toolCallStyle.Render("Edit:"), toolPathStyle.Render(params.Path), meta)

	diffText := renderInlineDiff(params.OldString, params.NewString)
	renderedBox := diffText
	if width > 0 {
		boxWidth := max(20, width-2)
		renderedBox = toolBoxStyle.Width(boxWidth).Render(diffText)
	} else {
		renderedBox = toolBoxStyle.Render(diffText)
	}

	return fmt.Sprintf("%s\n%s", header, renderedBox)
}

func renderToolResult(result *coagent.ToolResultInfo, call *coagent.ToolCallInfo) string {
	if result == nil {
		return ""
	}
	switch result.Name {
	case "read":
		return renderReadResult(result, call)
	case "bash":
		return renderBashResult(result)
	case "write":
		return renderWriteResult(result, call)
	case "edit":
		return renderEditResult(result, call)
	default:
		out := formatFullOutput(result.Output)
		if isErrorOutput(result.Output) {
			return fmt.Sprintf("↳ %s %s", toolErrorStyle.Render("✗"), out)
		}
		return fmt.Sprintf("↳ %s", toolResultStyle.Render(out))
	}
}

func renderReadResult(result *coagent.ToolResultInfo, call *coagent.ToolCallInfo) string {
	output := strings.TrimSpace(result.Output)
	if isErrorOutput(output) {
		cleanErr := strings.TrimPrefix(output, "Error: ")
		return fmt.Sprintf("↳ %s %s", toolErrorStyle.Render("✗"), cleanErr)
	}

	targetPath := ""
	if call != nil {
		targetPath = parseReadParams(call.Arguments).Path
	}

	if strings.HasPrefix(output, "Successfully rendered PDF ") {
		return renderPDFResult(output, targetPath)
	}

	if strings.HasPrefix(output, "Successfully read image ") {
		return renderImageResult(output, targetPath)
	}

	if output == "(empty file)" {
		return renderEmptyFileResult(targetPath)
	}

	return renderTextFileResult(result.Output, targetPath)
}

func renderBashResult(result *coagent.ToolResultInfo) string {
	output := strings.TrimSpace(result.Output)
	if output == "" || output == "(no output)" {
		return fmt.Sprintf("↳ %s", toolMutedStyle.Render("(no output)"))
	}

	isErr := strings.Contains(output, "Command exited with error:") || strings.HasPrefix(output, "Error:")
	formatted := formatFullOutput(result.Output)

	if isErr {
		return fmt.Sprintf("↳ %s\n%s", toolErrorStyle.Render("✗ Command failed:"), toolResultStyle.Render(formatted))
	}
	return fmt.Sprintf("↳\n%s", toolResultStyle.Render(formatted))
}

func renderWriteResult(result *coagent.ToolResultInfo, call *coagent.ToolCallInfo) string {
	output := strings.TrimSpace(result.Output)
	if isErrorOutput(output) {
		cleanErr := strings.TrimPrefix(output, "Error: ")
		return fmt.Sprintf("↳ %s %s", toolErrorStyle.Render("✗"), cleanErr)
	}

	targetPath := ""
	if call != nil {
		targetPath = parseWriteParams(call.Arguments).Path
	}

	if targetPath != "" {
		return fmt.Sprintf("↳ %s %s %s", toolSuccessStyle.Render("✓ Written:"), toolPathStyle.Render(targetPath), toolMutedStyle.Render(fmt.Sprintf("(%s)", output)))
	}
	return fmt.Sprintf("↳ %s %s", toolSuccessStyle.Render("✓"), output)
}

func renderEditResult(result *coagent.ToolResultInfo, call *coagent.ToolCallInfo) string {
	output := strings.TrimSpace(result.Output)
	if isErrorOutput(output) {
		cleanErr := strings.TrimPrefix(output, "Error: ")
		return fmt.Sprintf("↳ %s %s", toolErrorStyle.Render("✗"), cleanErr)
	}

	targetPath := ""
	if call != nil {
		targetPath = parseEditParams(call.Arguments).Path
	}

	if targetPath != "" {
		return fmt.Sprintf("↳ %s %s", toolSuccessStyle.Render("✓ Replaced text in:"), toolPathStyle.Render(targetPath))
	}
	return fmt.Sprintf("↳ %s %s", toolSuccessStyle.Render("✓"), output)
}

func isErrorOutput(output string) bool {
	return strings.HasPrefix(output, "Error:") ||
		strings.HasPrefix(output, "cannot read") ||
		strings.HasPrefix(output, "failed to") ||
		strings.HasPrefix(output, "guard rejected:") ||
		strings.HasPrefix(output, "old_string ") ||
		strings.Contains(output, "Command exited with error:")
}

func renderPDFResult(output, fallbackPath string) string {
	var path, pages string
	n, _ := fmt.Sscanf(output, "Successfully rendered PDF %s (%s", &path, &pages)
	if n >= 1 && path != "" {
		fallbackPath = path
	}
	pagesStr := strings.TrimSuffix(pages, ")")
	if pagesStr != "" {
		return fmt.Sprintf("↳ %s %s %s", toolSuccessStyle.Render("✓ Rendered PDF:"), toolPathStyle.Render(fallbackPath), toolMutedStyle.Render("("+pagesStr+")"))
	}
	return fmt.Sprintf("↳ %s %s", toolSuccessStyle.Render("✓ Rendered PDF:"), toolPathStyle.Render(fallbackPath))
}

func renderImageResult(output, fallbackPath string) string {
	var path string
	var bytesCount int
	n, _ := fmt.Sscanf(output, "Successfully read image %s (%d bytes)", &path, &bytesCount)
	if n >= 1 && path != "" {
		fallbackPath = path
	}
	if bytesCount > 0 {
		return fmt.Sprintf("↳ %s %s %s", toolSuccessStyle.Render("✓ Loaded image:"), toolPathStyle.Render(fallbackPath), toolMutedStyle.Render(fmt.Sprintf("(%s)", formatBytes(bytesCount))))
	}
	return fmt.Sprintf("↳ %s %s", toolSuccessStyle.Render("✓ Loaded image:"), toolPathStyle.Render(fallbackPath))
}

func renderEmptyFileResult(targetPath string) string {
	if targetPath != "" {
		return fmt.Sprintf("↳ %s %s %s", toolSuccessStyle.Render("✓ Read:"), toolPathStyle.Render(targetPath), toolMutedStyle.Render("(empty file, 0 lines)"))
	}
	return fmt.Sprintf("↳ %s %s", toolSuccessStyle.Render("✓ Read:"), toolMutedStyle.Render("(empty file, 0 lines)"))
}

func renderTextFileResult(rawOutput, targetPath string) string {
	lineCount := countLines(rawOutput)
	byteCount := len([]byte(rawOutput))
	meta := toolMutedStyle.Render(fmt.Sprintf("(%d lines, %s)", lineCount, formatBytes(byteCount)))

	if targetPath != "" {
		return fmt.Sprintf("↳ %s %s %s", toolSuccessStyle.Render("✓ Read:"), toolPathStyle.Render(targetPath), meta)
	}
	return fmt.Sprintf("↳ %s %s", toolSuccessStyle.Render("✓ Read file"), meta)
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func formatFullOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "(no output)"
	}
	lines := strings.Split(trimmed, "\n")
	indented := make([]string, len(lines))
	for i, line := range lines {
		indented[i] = "  " + line
	}
	return strings.Join(indented, "\n")
}

type diffOp int

const (
	diffEqual diffOp = iota
	diffDelete
	diffInsert
)

type diffChunk struct {
	op   diffOp
	line string
}

func renderInlineDiff(oldStr, newStr string) string {
	oldLines := splitLines(oldStr)
	newLines := splitLines(newStr)

	chunks := computeLineDiff(oldLines, newLines)
	if len(chunks) == 0 {
		return toolMutedStyle.Render("  (no changes)")
	}

	var sb strings.Builder
	for i, c := range chunks {
		if i > 0 {
			sb.WriteByte('\n')
		}
		switch c.op {
		case diffDelete:
			sb.WriteString(diffRemovedStyle.Render("- " + c.line))
		case diffInsert:
			sb.WriteString(diffAddedStyle.Render("+ " + c.line))
		case diffEqual:
			sb.WriteString(diffContextStyle.Render("  " + c.line))
		}
	}
	return sb.String()
}

func computeLineDiff(a, b []string) []diffChunk {
	m := len(a)
	n := len(b)

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			if a[i] == b[j] {
				dp[i+1][j+1] = dp[i][j] + 1
				continue
			}
			dp[i+1][j+1] = max(dp[i+1][j], dp[i][j+1])
		}
	}

	var diff []diffChunk
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			diff = append(diff, diffChunk{op: diffEqual, line: a[i-1]})
			i--
			j--
			continue
		}
		if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			diff = append(diff, diffChunk{op: diffInsert, line: b[j-1]})
			j--
			continue
		}
		if i > 0 && (j == 0 || dp[i][j-1] < dp[i-1][j]) {
			diff = append(diff, diffChunk{op: diffDelete, line: a[i-1]})
			i--
		}
	}
	slices.Reverse(diff)
	return diff
}

func formatBytes(b int) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	normalized := strings.ReplaceAll(s, "\r\n", "\n")
	trimmed := strings.TrimSuffix(normalized, "\n")
	return strings.Split(trimmed, "\n")
}
