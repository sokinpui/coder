package session

import (
	"coder/internal/config"
	"coder/internal/modes"
)

// LoadContext loads the initial context for the session using the current mode strategy.
func (s *Session) LoadContext() error {
	return s.modeStrategy.LoadContext()
}

// SetMode changes the application mode and reloads the context.
func (s *Session) SetMode(appMode config.AppMode) error {
	s.config.AppMode = appMode
	s.modeStrategy = modes.NewStrategy(appMode)
	return s.LoadContext()
}
