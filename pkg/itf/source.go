package itf

import (
	"io"
	"os"
	"strings"

	"github.com/atotto/clipboard"
)

type SourceProvider struct{}

func NewSourceProvider() *SourceProvider {
	return &SourceProvider{}
}

func (sp *SourceProvider) GetContent() (string, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		c, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(c), nil
	}

	c, err := clipboard.ReadAll()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(c), nil
}
