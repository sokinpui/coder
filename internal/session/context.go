package session

import (
	"github.com/sokinpui/coder/internal/history"
	"os"
	"strings"
)

func (s *Session) LoadContext() error {
	if err := s.modeStrategy.LoadSourceCode(s.config); err != nil {
		return err
	}

	// Get the system prompt part for history saving
	msgs := s.modeStrategy.BuildPrompt(nil)
	if len(msgs) > 0 {
		s.context = history.BuildHistorySnippet(msgs)
	}
	return nil
}

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
