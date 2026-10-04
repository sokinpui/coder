package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type Position int

const (
	PositionTop Position = iota + 1
	PositionRight
	PositionBottom
	PositionLeft
	PositionCenter
)

func OverlayCenter(fg, bg string) string {
	return Composite(fg, bg, PositionCenter, PositionCenter, 0, 0)
}

func Composite(fg, bg string, xPos, yPos Position, xOff, yOff int) string {
	if fg == "" {
		return bg
	}
	if bg == "" {
		return fg
	}
	if !strings.Contains(fg, "\n") && !strings.Contains(bg, "\n") {
		return fg
	}

	fgWidth, fgHeight := lipgloss.Size(fg)
	bgWidth, bgHeight := lipgloss.Size(bg)

	if fgWidth >= bgWidth && fgHeight >= bgHeight {
		return fg
	}

	x, y := calculateOffsets(fg, bg, xPos, yPos, xOff, yOff)
	x = clamp(x, 0, bgWidth-fgWidth)
	y = clamp(y, 0, bgHeight-fgHeight)

	fgLines := strings.Split(strings.ReplaceAll(fg, "\r\n", "\n"), "\n")
	bgLines := strings.Split(strings.ReplaceAll(bg, "\r\n", "\n"), "\n")
	var sb strings.Builder

	for i, bgLine := range bgLines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if i < y || i >= y+fgHeight {
			sb.WriteString(bgLine)
			continue
		}

		pos := 0
		if x > 0 {
			left := ansi.Truncate(bgLine, x, "")
			pos = ansi.StringWidth(left)
			sb.WriteString(left)
			if pos < x {
				sb.WriteString(strings.Repeat(" ", x-pos))
				pos = x
			}
		}

		fgLine := fgLines[i-y]
		sb.WriteString(fgLine)
		pos += ansi.StringWidth(fgLine)

		right := ansi.TruncateLeft(bgLine, pos, "")
		bgLineWidth := ansi.StringWidth(bgLine)
		rightWidth := ansi.StringWidth(right)
		if rightWidth <= bgLineWidth-pos {
			sb.WriteString(strings.Repeat(" ", bgLineWidth-rightWidth-pos))
		}
		sb.WriteString(right)
	}
	return sb.String()
}

func calculateOffsets(fg, bg string, xPos, yPos Position, xOff, yOff int) (int, int) {
	var x, y int

	switch xPos {
	case PositionLeft:
		x = 0
	case PositionCenter:
		x = (lipgloss.Width(bg) - lipgloss.Width(fg)) / 2
	case PositionRight:
		x = lipgloss.Width(bg) - lipgloss.Width(fg)
	}

	switch yPos {
	case PositionTop:
		y = 0
	case PositionCenter:
		y = (lipgloss.Height(bg) - lipgloss.Height(fg)) / 2
	case PositionBottom:
		y = lipgloss.Height(bg) - lipgloss.Height(fg)
	}

	return x + xOff, y + yOff
}

func clamp(v, minVal, maxVal int) int {
	if minVal > maxVal {
		minVal, maxVal = maxVal, minVal
	}
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}
