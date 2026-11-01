package ui

import (
	"regexp"
	"strconv"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	sgrFragmentRegex = regexp.MustCompile(`\[?<\d+;\d+;\d+[Mm]`)
	sgrPartialRegex  = regexp.MustCompile(`\[<\d*(?:;\d*)*$`)
	sgrMouseRegex    = regexp.MustCompile(`\[?<(\d+);(\d+);(\d+)([Mm])`)
)

func CleanSGRLeakedText(s string) string {
	if !sgrFragmentRegex.MatchString(s) {
		return s
	}
	return sgrFragmentRegex.ReplaceAllString(s, "")
}

const wheelThrottleInterval = 42 * time.Millisecond

type sgrSequenceFilter struct {
	mu         sync.Mutex
	pending    string
	sender     func(tea.Msg)
	accumDelta int
	lastFlush  time.Time
	flushTimer *time.Timer
}

func newSGRSequenceFilter() *sgrSequenceFilter {
	return &sgrSequenceFilter{}
}

func (filter *sgrSequenceFilter) SetSender(sender func(tea.Msg)) {
	filter.mu.Lock()
	defer filter.mu.Unlock()
	filter.sender = sender
}

func (filter *sgrSequenceFilter) Filter(tm tea.Model, msg tea.Msg) tea.Msg {
	if wheelMsg, ok := msg.(MouseWheelScrollMsg); ok {
		return wheelMsg
	}

	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		return filter.handleMouse(tm, mouseMsg)
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return msg
	}

	if keyMsg.Type != tea.KeyRunes || len(keyMsg.Runes) == 0 {
		filter.pending = ""
		return keyMsg
	}

	raw := string(keyMsg.Runes)
	if keyMsg.Alt && raw == "[" {
		filter.pending = "["
		return nil
	}

	combined := filter.pending + raw
	filter.pending = ""

	var mouseEv tea.Msg
	var isMouse bool
	if sgrFragmentRegex.MatchString(combined) {
		mouseEv, isMouse = parseSGRMouseMsg(combined)
		combined = sgrFragmentRegex.ReplaceAllString(combined, "")
	}

	cleaned := combined
	combined = ""

	if loc := sgrPartialRegex.FindStringIndex(cleaned); loc != nil {
		filter.pending = cleaned[loc[0]:]
		cleaned = cleaned[:loc[0]]
	}

	if len(filter.pending) > 32 {
		filter.pending = ""
	}

	if isMouse && mouseEv != nil {
		if mm, ok := mouseEv.(tea.MouseMsg); ok {
			return filter.handleMouse(tm, mm)
		}
		return mouseEv
	}

	if cleaned == "" {
		return nil
	}

	if cleaned == raw && !keyMsg.Alt {
		return keyMsg
	}

	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(cleaned),
	}
}

func (filter *sgrSequenceFilter) handleMouse(tm tea.Model, mouseMsg tea.MouseMsg) tea.Msg {
	if mouseMsg.Action == tea.MouseActionMotion {
		main := extractMainModel(tm)
		if main == nil || !main.Chat.Selection.Dragging {
			return nil
		}
		return mouseMsg
	}

	if mouseMsg.Button != tea.MouseButtonWheelUp && mouseMsg.Button != tea.MouseButtonWheelDown {
		filter.flushPending()
		return mouseMsg
	}

	if !canScrollMouse(tm, mouseMsg) {
		filter.flushPending()
		return nil
	}

	step := 2
	if mouseMsg.Button == tea.MouseButtonWheelUp {
		step = -2
	}

	return filter.coalesceWheel(step, mouseMsg.X, mouseMsg.Y)
}

func (filter *sgrSequenceFilter) coalesceWheel(step, x, y int) tea.Msg {
	filter.mu.Lock()
	defer filter.mu.Unlock()

	now := time.Now()
	if filter.accumDelta != 0 && (filter.accumDelta > 0) != (step > 0) {
		filter.clearTimerLocked()
		filter.accumDelta = 0
	}

	if now.Sub(filter.lastFlush) >= wheelThrottleInterval {
		filter.clearTimerLocked()
		total := filter.accumDelta + step
		filter.accumDelta = 0
		filter.lastFlush = now
		return MouseWheelScrollMsg{Lines: total, X: x, Y: y}
	}

	filter.accumDelta += step
	if filter.flushTimer == nil && filter.sender != nil {
		remaining := max(time.Millisecond, wheelThrottleInterval-now.Sub(filter.lastFlush))
		filter.flushTimer = time.AfterFunc(remaining, func() {
			filter.mu.Lock()
			if filter.accumDelta == 0 {
				filter.flushTimer = nil
				filter.mu.Unlock()
				return
			}
			lines := filter.accumDelta
			filter.accumDelta = 0
			filter.lastFlush = time.Now()
			filter.flushTimer = nil
			send := filter.sender
			filter.mu.Unlock()

			if send != nil {
				send(MouseWheelScrollMsg{Lines: lines, X: x, Y: y})
			}
		})
	}
	return nil
}

func parseSGRMouseMsg(s string) (tea.Msg, bool) {
	match := sgrMouseRegex.FindStringSubmatch(s)
	if len(match) < 5 {
		return nil, false
	}

	btn, _ := strconv.Atoi(match[1])
	x, _ := strconv.Atoi(match[2])
	y, _ := strconv.Atoi(match[3])

	if x > 0 {
		x--
	}
	if y > 0 {
		y--
	}

	switch btn {
	case 64:
		return tea.MouseMsg{
			X:      x,
			Y:      y,
			Button: tea.MouseButtonWheelUp,
			Action: tea.MouseActionPress,
		}, true
	case 65:
		return tea.MouseMsg{
			X:      x,
			Y:      y,
			Button: tea.MouseButtonWheelDown,
			Action: tea.MouseActionPress,
		}, true
	default:
		return nil, true
	}
}

func canScrollMouse(tm tea.Model, msg tea.MouseMsg) bool {
	if msg.Button != tea.MouseButtonWheelUp && msg.Button != tea.MouseButtonWheelDown {
		return true
	}

	main := extractMainModel(tm)
	if main == nil {
		return true
	}

	switch main.ActiveOverlay {
	case overlayNone:
		if msg.Button == tea.MouseButtonWheelUp {
			return main.Chat.Viewport.YOffset > 0
		}
		return !main.Chat.Viewport.AtBottom()

	case overlayQuickView:
		if main.QuickView == nil {
			return false
		}
		if main.QuickView.needsRender {
			return true
		}
		if msg.Button == tea.MouseButtonWheelUp {
			return main.QuickView.Viewport.YOffset > 0
		}
		return !main.QuickView.Viewport.AtBottom()

	default:
		return false
	}
}

func (filter *sgrSequenceFilter) flushPending() {
	filter.mu.Lock()
	defer filter.mu.Unlock()
	filter.clearTimerLocked()
	filter.accumDelta = 0
}

func (filter *sgrSequenceFilter) clearTimerLocked() {
	if filter.flushTimer != nil {
		filter.flushTimer.Stop()
		filter.flushTimer = nil
	}
}

func extractMainModel(tm tea.Model) *Model {
	switch m := tm.(type) {
	case *Manager:
		if m != nil {
			return m.Main
		}
	case *Model:
		return m
	case Model:
		return &m
	}
	return nil
}
