// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package diagnostics provides an ergonomic wrapper around hcl.Diagnostics.
package diagnostics

import (
	"bufio"
	"bytes"
	"errors"
	"regexp"
	"unicode"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/itchyny/gojq"
)

var heredocRe = regexp.MustCompile(`^\s*<<`)

// GoJQError enhances diagnostics produced for gojq errors when attached as Extra.
// Diagnostic must include Subject pointing to the query expression.
type GoJQError struct {
	Err   error
	Query string
}

func extractJQErrorDetails(err GoJQError) map[string]any {
	var parseErr *gojq.ParseError
	if !errors.As(err.Err, &parseErr) {
		return nil
	}

	if err.Query == "" {
		return nil
	}

	return map[string]any{
		"query":  err.Query,
		"offset": parseErr.Offset,
		"token":  parseErr.Token,
	}
}

// Rewrites diag and fileMap to improve the error message for GoJQError on a best-effort basis.
func (gojqErr *GoJQError) improveDiagnostic(diag *hcl.Diagnostic, fileMap map[string]*hcl.File) {
	var err *gojq.ParseError
	if !errors.As(gojqErr.Err, &err) {
		return
	}
	if gojqErr.Query == "" || diag.Subject == nil || fileMap[diag.Subject.Filename] == nil {
		return
	}

	// Pad query with newlines to match the line numbers
	nlNum := diag.Subject.Start.Line - 1
	if heredocRe.Match(diag.Subject.SliceBytes(fileMap[diag.Subject.Filename].Bytes)) {
		// heredocs start from the next line
		nlNum++
	}
	query := bytes.Repeat([]byte{'\n'}, nlNum)
	query = append(query, gojqErr.Query...)

	tokLen := len(err.Token)

	// Get offset to the start of the problematic token
	offset := max(0, min(len(query), err.Offset-tokLen+nlNum))
	if len(bytes.TrimSpace(query[offset:offset+tokLen])) == 0 {
		// If the token is empty, try to find the first non-whitespace character
		// before the token. Hcl diagnostic printer does not highlight the whitespace
		for offset > nlNum {
			r, length := utf8.DecodeLastRune(query[:offset])
			if length == 0 {
				// in case of invalid utf-8 just move byte by byte
				offset -= 1
				tokLen += 1
				continue
			}
			offset -= length
			tokLen += length
			if unicode.IsSpace(r) {
				continue
			}
			// non-space character found
			break
		}
	}

	scannerCalls := 0
	scanner := hcl.NewRangeScanner(query, "", func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		scannerCalls++
		switch scannerCalls {
		case 1: // output padding whitespace
			return nlNum, data[:nlNum], nil
		case 2: // output contextStart
			return offset - nlNum, data[:offset-nlNum], nil
		case 3: // output subject
			return tokLen, data[:tokLen], nil
		case 4: // output contextEnd
			return len(data), data, nil
		default:
			return 0, nil, bufio.ErrFinalToken
		}
	})
	var contextStart, subjectStart, subjectEnd, contextEnd hcl.Pos
	if scanner.Scan() {
		contextStart = scanner.Range().End
	}
	if scanner.Scan() {
		contextStart = scanner.Range().Start
	}
	if scanner.Scan() {
		r := scanner.Range()
		subjectStart = r.Start
		subjectEnd = r.End
	} else {
		// failed to scan subject, cannot improve the error message
		return
	}
	if scanner.Scan() {
		contextEnd = scanner.Range().End
	}
	if contextStart == (hcl.Pos{}) {
		contextStart = subjectStart
	}
	if contextEnd == (hcl.Pos{}) {
		contextEnd = subjectEnd
	}

	syntheticFileName := "query inside " + diag.Subject.Filename
	fileMap[syntheticFileName] = &hcl.File{
		Bytes: query,
	}
	diag.Subject = &hcl.Range{
		Filename: syntheticFileName,
		Start:    subjectStart,
		End:      subjectEnd,
	}
	diag.Context = &hcl.Range{
		Filename: syntheticFileName,
		Start:    contextStart,
		End:      contextEnd,
	}
}
