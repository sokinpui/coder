package commands

import "strings"

type ArgToken struct {
	Raw   string
	Value string
	Start int
	End   int
}

func ParseArgTokens(input string) []ArgToken {
	var tokens []ArgToken
	i := 0
	n := len(input)

	for i < n {
		for i < n && isArgWhitespace(input[i]) {
			i++
		}
		if i >= n {
			break
		}

		start := i
		var val strings.Builder
		var quote byte

		for i < n {
			c := input[i]
			if quote == 0 {
				if isArgWhitespace(c) {
					break
				}
				if c == '"' || c == '\'' {
					quote = c
					i++
					continue
				}
				if c == '\\' {
					if i+1 < n {
						val.WriteByte(input[i+1])
						i += 2
						continue
					}
					val.WriteByte(c)
					i++
					continue
				}
				val.WriteByte(c)
				i++
				continue
			}

			if quote == '\'' {
				if c == '\'' {
					quote = 0
					i++
					continue
				}
				val.WriteByte(c)
				i++
				continue
			}

			if c == '"' {
				quote = 0
				i++
				continue
			}
			if c == '\\' && i+1 < n {
				next := input[i+1]
				if next == '"' || next == '\\' || next == '$' || next == '`' {
					val.WriteByte(next)
					i += 2
					continue
				}
			}
			val.WriteByte(c)
			i++
		}

		tokens = append(tokens, ArgToken{
			Raw:   input[start:i],
			Value: val.String(),
			Start: start,
			End:   i,
		})
	}

	return tokens
}

func SplitArgs(input string) []string {
	tokens := ParseArgTokens(input)
	if len(tokens) == 0 {
		return nil
	}
	args := make([]string, len(tokens))
	for i, tok := range tokens {
		args[i] = tok.Value
	}
	return args
}

func FormatArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\n\r\"'\\") {
		return arg
	}
	var b strings.Builder
	b.Grow(len(arg) + 2)
	b.WriteByte('"')
	for i := 0; i < len(arg); i++ {
		c := arg[i]
		if c == '"' || c == '\\' || c == '$' || c == '`' {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	b.WriteByte('"')
	return b.String()
}

func FormatArgs(args []string) string {
	formatted := make([]string, len(args))
	for i, arg := range args {
		formatted[i] = FormatArg(arg)
	}
	return strings.Join(formatted, " ")
}

func ActiveCommandArg(input string) (cmdName string, argPrefix string, replacePrefix string, ok bool) {
	sigils := []struct {
		prefix  string
		cmdName string
	}{
		{"/@", "file"},
		{"/!", "shell"},
		{"@", "file"},
		{"!", "shell"},
	}

	for _, sigil := range sigils {
		if !strings.HasPrefix(input, sigil.prefix) {
			continue
		}
		n := len(sigil.prefix)
		rest := input[n:]
		if rest == "" {
			return sigil.cmdName, "", sigil.prefix + " ", true
		}

		tokens := ParseArgTokens(rest)
		if len(tokens) > 0 && tokens[len(tokens)-1].End == len(rest) {
			last := tokens[len(tokens)-1]
			if !isArgWhitespace(rest[0]) && len(tokens) == 1 {
				return sigil.cmdName, last.Value, sigil.prefix + " ", true
			}
			return sigil.cmdName, last.Value, input[:n+last.Start], true
		}

		if !isArgWhitespace(rest[0]) && len(tokens) == 0 {
			return sigil.cmdName, "", sigil.prefix + " ", true
		}
		return sigil.cmdName, "", input, true
	}

	if !strings.HasPrefix(input, "/") {
		return "", "", "", false
	}

	wsIdx := strings.IndexAny(input, " \t\n\r")
	if wsIdx == -1 {
		return "", "", "", false
	}

	cmdName = input[1:wsIdx]
	rest := input[wsIdx:]
	tokens := ParseArgTokens(rest)
	if len(tokens) > 0 && tokens[len(tokens)-1].End == len(rest) {
		last := tokens[len(tokens)-1]
		return cmdName, last.Value, input[:wsIdx+last.Start], true
	}

	return cmdName, "", input, true
}

func isArgWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
