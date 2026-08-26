package cmd

import (
	"strings"
)

func matchLines(lines []string, query string) []int {
	var matches []int
	q := strings.ToLower(query)
	if q == "" {
		return matches
	}
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), q) {
			matches = append(matches, i)
		}
	}
	return matches
}

func nextMatchLine(matches []int, current int) int {
	if len(matches) == 0 {
		return 0
	}
	for _, line := range matches {
		if line > current {
			return line
		}
	}
	return matches[0]
}

func prevMatchLine(matches []int, current int) int {
	if len(matches) == 0 {
		return 0
	}
	for i := len(matches) - 1; i >= 0; i-- {
		if matches[i] < current {
			return matches[i]
		}
	}
	return matches[len(matches)-1]
}

func matchRank(matches []int, line int) int {
	for i, m := range matches {
		if m == line {
			return i + 1
		}
	}
	return 0
}

func firstMatchAtOrAfter(matches []int, from int) int {
	for _, line := range matches {
		if line >= from {
			return line
		}
	}
	if len(matches) == 0 {
		return 0
	}
	return matches[0]
}
