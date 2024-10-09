package util

import "strings"

func Dedent(text string) string {
	lines := strings.Split(text, "\n")

	indent := make([]int, len(lines))
	minIndent := -1
	for i, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		indent[i] = len(line) - len(strings.TrimLeft(line, " \t"))
		if indent[i] == 0 {
			continue
		}
		if minIndent == -1 || indent[i] < minIndent {
			minIndent = indent[i]
		}
	}

	if minIndent == -1 {
		return text
	}

	for i, line := range lines {
		if len(line) >= minIndent && indent[i] > 0 {
			lines[i] = line[minIndent:]
		}
	}

	return strings.Join(lines, "\n")
}
