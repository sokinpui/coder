package session

// LoadContext loads the initial context for the session using the current mode strategy.
func (s *Session) LoadContext() error {
	if err := s.modeStrategy.LoadSourceCode(s.config); err != nil {
		return err
	}
	s.context = s.modeStrategy.BuildPrompt(nil) // return only the Role instruction + source code
	return nil
}
