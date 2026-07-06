package itf

import (
	"bufio"
	"bytes"
	"strings"
)

type CodeBlock struct {
	Hint    string
	Lang    string
	Content string
}

func ExtractCodeBlocks(source []byte) ([]CodeBlock, error) {
	var blocks []CodeBlock
	var currentBlock *CodeBlock
	var fenceChar byte
	var fenceCount int
	var lastNonEmptyLine string

	scanner := bufio.NewScanner(bytes.NewReader(source))
	for scanner.Scan() {
		line := scanner.Text()

		if currentBlock == nil {
			char, count, ok := parseOpeningFence(line)
			if ok {
				fenceChar = char
				fenceCount = count
				currentBlock = &CodeBlock{
					Lang: strings.TrimSpace(line[count:]),
					Hint: lastNonEmptyLine,
				}
				continue
			}

			if trimmed := strings.TrimSpace(line); trimmed != "" {
				lastNonEmptyLine = trimmed
			}
			continue
		}

		if isClosingFence(line, fenceChar, fenceCount) {
			blocks = append(blocks, *currentBlock)
			currentBlock = nil
			lastNonEmptyLine = ""
			continue
		}

		currentBlock.Content += line + "\n"
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if currentBlock != nil {
		blocks = append(blocks, *currentBlock)
	}

	return blocks, nil
}

func parseOpeningFence(line string) (byte, int, bool) {
	if len(line) < 3 {
		return 0, 0, false
	}

	char := line[0]
	if char != '`' && char != '~' {
		return 0, 0, false
	}

	count := 0
	for count < len(line) && line[count] == char {
		count++
	}

	if count < 3 {
		return 0, 0, false
	}

	return char, count, true
}

func isClosingFence(line string, char byte, count int) bool {
	if len(line) < count {
		return false
	}

	i := 0
	for i < len(line) && line[i] == char {
		i++
	}

	if i < count {
		return false
	}

	return strings.TrimSpace(line[i:]) == ""
}
