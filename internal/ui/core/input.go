package core

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

type InputBox struct {
	Model textarea.Model
}

func NewInputBox(placeholder string) InputBox {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.Focus()
	ta.SetHeight(1)
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.MaxHeight = 0
	ta.MaxWidth = 0
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	return InputBox{Model: ta}
}

func (ib *InputBox) View() string {
	return TextAreaContainerStyle.Render(ib.Model.View())
}

func (ib *InputBox) SetWidth(width int) {
	inner := max(10, width-TextAreaContainerStyle.GetHorizontalFrameSize())
	ib.Model.SetWidth(inner)
}

func (ib *InputBox) UpdateHeight(screenHeight int) int {
	maxHeight := screenHeight / 4
	visibleLines := CalculateVisibleLines(ib.Model, ib.Model.Width(), maxHeight+1)
	inputHeight := min(visibleLines+1, maxHeight)
	ib.Model.SetHeight(max(1, inputHeight))
	return ib.Model.Height()
}

func CalculateVisibleLines(ta textarea.Model, width int, maxLines int) int {
	if width <= 0 {
		return 1
	}

	visibleLineCount := 0
	for line := range strings.SplitSeq(ta.Value(), "\n") {
		lineWidth := lipgloss.Width(line)
		if lineWidth == 0 {
			visibleLineCount++
			continue
		}
		visibleLineCount += (lineWidth-1)/width + 1
		if maxLines > 0 && visibleLineCount > maxLines {
			return visibleLineCount
		}
	}
	return visibleLineCount
}
