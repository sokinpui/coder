package itf

import (
	"fmt"
)

func Apply(content string, config Config) (map[string][]string, error) {
	app, err := NewApp(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize itf app: %w", err)
	}

	summary, err := app.processAndApply(content)
	if err != nil {
		return nil, err
	}

	return map[string][]string{
		"Created":  summary.Created,
		"Modified": summary.Modified,
		"Renamed":  summary.Renamed,
		"Deleted":  summary.Deleted,
		"Failed":   summary.Failed,
		"Message":  []string{summary.Message},
	}, nil
}

func FormatResult(results map[string][]string) string {
	if results == nil {
		return ""
	}

	msg := ""
	if m, ok := results["Message"]; ok && len(m) > 0 {
		msg = m[0]
	}

	return FormatSummary(Summary{
		Created:  results["Created"],
		Modified: results["Modified"],
		Renamed:  results["Renamed"],
		Deleted:  results["Deleted"],
		Failed:   results["Failed"],
		Message:  msg,
	})
}
