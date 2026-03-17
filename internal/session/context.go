package session

import (
	"os"
	"strings"
)

// LoadContext loads the initial context for the session using the current mode strategy.
func (s *Session) LoadContext() error {
	if err := s.modeStrategy.LoadSourceCode(s.config); err != nil {
		return err
	}
	s.context = s.modeStrategy.BuildPrompt(nil) // return only the Role instruction + source code
	return nil
}

// applyContextFiles overrides the current config context with specific paths.
func (s *Session) applyContextFiles(paths []string) {
	s.config.Context.Dirs = []string{}
	s.config.Context.Files = []string{}
	s.config.Context.Exclusions = []string{}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			s.config.Context.Dirs = append(s.config.Context.Dirs, p)
		} else {
			s.config.Context.Files = append(s.config.Context.Files, p)
		}
	}
}
