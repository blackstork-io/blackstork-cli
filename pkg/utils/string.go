// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package utils

import (
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/exp/maps"
)

func JoinSurround(sep, surround string, elems ...string) string {
	if len(elems) == 0 {
		return ""
	}
	var b strings.Builder
	resLen := len(sep) * (len(elems) - 1)
	resLen += len(surround) * 2 * len(elems)
	for _, e := range elems {
		resLen += len(e)
	}
	b.Grow(resLen)

	b.WriteString(surround)
	b.WriteString(elems[0])
	b.WriteString(surround)
	for _, e := range elems[1:] {
		b.WriteString(sep)
		b.WriteString(surround)
		b.WriteString(e)
		b.WriteString(surround)
	}
	return b.String()
}

// CapitalizeFirstLetter capitalizes the first Unicode letter and otherwise returns the string unchanged.
func CapitalizeFirstLetter(s string) string {
	r, offset := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		// do nothing, since this is just a cosmetic function
		return s
	}
	upperR := unicode.ToUpper(r)
	if r == upperR {
		// upper and lower case letters are identical, do not realloc
		return s
	}
	capitalRuneLen := utf8.RuneLen(upperR)
	if capitalRuneLen == -1 {
		return s
	}

	var b strings.Builder

	b.Grow(capitalRuneLen + len(s) - offset)
	b.WriteRune(upperR)
	b.WriteString(s[offset:])
	return b.String()
}

func MemoizedKeys[M ~map[string]V, V any](m *M) func() string {
	return sync.OnceValue(func() string {
		keys := maps.Keys(*m)
		slices.Sort(keys)
		return JoinSurround(", ", "'", keys...)
	})
}

// Dedent strips the common margin from lines.
func Dedent(text string) string {
	lines := strings.Split(text, "\n")
	commonIndent := findCommonIndent(lines)

	newlines := make([]string, 0)

	for _, line := range lines {
		trimmed := strings.TrimLeft(line, "\t")
		// If the line is empty, trim it fully
		var newline string
		if trimmed == "" {
			newline = trimmed
		} else {
			newline = strings.TrimPrefix(line, commonIndent)
		}
		newlines = append(newlines, newline)
	}

	return strings.Join(newlines, "\n")
}

func findCommonIndent(lines []string) string {
	minCommonIndent := ""

	for _, line := range lines {
		trimmed := strings.TrimLeft(line, "\t")

		// Ignore empty lines
		if trimmed == "" {
			continue
		}
		nonIndentCharIndex := strings.IndexFunc(line, func(c rune) bool {
			return c != '\t'
		})

		lineIndent := line[:nonIndentCharIndex]
		if minCommonIndent == "" || len(lineIndent) < len(minCommonIndent) {
			minCommonIndent = lineIndent
		}
	}

	return minCommonIndent
}
