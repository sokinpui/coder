package session

func (s *Session) LoadContext() error {
	if err := s.modeStrategy.LoadSourceCode(s.contextFiles); err != nil {
		return err
	}
	return nil
}
