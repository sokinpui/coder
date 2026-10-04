package app

import (
	"github.com/sokinpui/coder/internal/ui/core"
)

type Position = core.Position

const (
	PositionTop    = core.PositionTop
	PositionRight  = core.PositionRight
	PositionBottom = core.PositionBottom
	PositionLeft   = core.PositionLeft
	PositionCenter = core.PositionCenter
)

func OverlayCenter(fg, bg string) string {
	return core.OverlayCenter(fg, bg)
}

func Composite(fg, bg string, xPos, yPos Position, xOff, yOff int) string {
	return core.Composite(fg, bg, xPos, yPos, xOff, yOff)
}
