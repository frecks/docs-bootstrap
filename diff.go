package main

import (
	"fmt"
	"strings"
)

// diffContext is the number of unchanged lines around each change.
const diffContext = 3

type diffLine struct {
	op   byte // ' ', '-', or '+'
	text string
}

// unifiedDiff returns the changes from a to b in unified format, or "" if
// they're the same.
func unifiedDiff(a, b, fromFile, toFile string) string {
	lines := diffLines(splitLines(a), splitLines(b))
	var changes []int
	for i, line := range lines {
		if line.op != ' ' {
			changes = append(changes, i)
		}
	}
	if len(changes) == 0 {
		return ""
	}
	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n+++ %s\n", fromFile, toFile)
	for start := 0; start < len(changes); {
		// A hunk takes every change within twice the context of the last one.
		end := start
		for end+1 < len(changes) && changes[end+1]-changes[end] <= 2*diffContext+1 {
			end++
		}
		from := max(0, changes[start]-diffContext)
		to := min(len(lines), changes[end]+1+diffContext)
		aStart, bStart := 0, 0
		for _, line := range lines[:from] {
			if line.op != '+' {
				aStart++
			}
			if line.op != '-' {
				bStart++
			}
		}
		aLen, bLen := 0, 0
		for _, line := range lines[from:to] {
			if line.op != '+' {
				aLen++
			}
			if line.op != '-' {
				bLen++
			}
		}
		fmt.Fprintf(&out, "@@ -%s +%s @@\n", hunkRange(aStart, aLen), hunkRange(bStart, bLen))
		for _, line := range lines[from:to] {
			out.WriteByte(line.op)
			out.WriteString(line.text)
		}
		start = end + 1
	}
	return out.String()
}

func hunkRange(start, length int) string {
	switch length {
	case 0:
		return fmt.Sprintf("%d,0", start)
	case 1:
		return fmt.Sprint(start + 1)
	}
	return fmt.Sprintf("%d,%d", start+1, length)
}

// splitLines splits text into lines, keeping each line's newline.
func splitLines(text string) []string {
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// diffLines aligns a and b on their longest common subsequence.
func diffLines(a, b []string) []diffLine {
	// common[i][j] is the length of the longest common subsequence of a[i:] and b[j:].
	common := make([][]int, len(a)+1)
	for i := range common {
		common[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				common[i][j] = common[i+1][j+1] + 1
			} else {
				common[i][j] = max(common[i+1][j], common[i][j+1])
			}
		}
	}
	var lines []diffLine
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			lines = append(lines, diffLine{' ', a[i]})
			i++
			j++
		case common[i+1][j] >= common[i][j+1]:
			lines = append(lines, diffLine{'-', a[i]})
			i++
		default:
			lines = append(lines, diffLine{'+', b[j]})
			j++
		}
	}
	for ; i < len(a); i++ {
		lines = append(lines, diffLine{'-', a[i]})
	}
	for ; j < len(b); j++ {
		lines = append(lines, diffLine{'+', b[j]})
	}
	return lines
}
