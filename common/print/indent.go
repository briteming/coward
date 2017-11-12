//  Crypto-Obscured Forwarder
//
//  Copyright (C) 2017 Rui NI <ranqus@gmail.com>
//
//  This file is part of Crypto-Obscured Forwarder.
//
//  Crypto-Obscured Forwarder is free software: you can redistribute it
//  and/or modify it under the terms of the GNU General Public License
//  as published by the Free Software Foundation, either version 3 of
//  the License, or (at your option) any later version.
//
//  Crypto-Obscured Forwarder is distributed in the hope that it will be
//  useful, but WITHOUT ANY WARRANTY; without even the implied warranty
//  of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with Crypto-Obscured Forwarder. If not, see
//  <http://www.gnu.org/licenses/>.

package print

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	spacese      = []byte{' ', '\t'}
	newlines     = []byte{'\n'}
	newlinesTrim = []byte{'\r', '\n'}
)

// Indent errors
var (
	ErrIndentNoEnoughSpace = errors.New(
		"No enough space remain to write any charactor to the line")
)

type indent struct{}

func (i *indent) findFirstLineBreaker(b []byte) int {
	result := -1
	length := len(b)

	for i := 0; i < length; i++ {
		for _, c := range newlines {
			if b[i] != c {
				continue
			}

			result = i

			break
		}

		if result != -1 {
			break
		}
	}

	return result
}

func (i *indent) findLastSpace(b []byte) int {
	result := -1

	for i := len(b) - 1; i >= 0; i-- {
		for _, c := range spacese {
			if b[i] != c {
				continue
			}

			result = i

			break
		}

		if result != -1 {
			break
		}
	}

	return result
}

func (i *indent) writeIndent(
	w io.Writer,
	d []byte,
	firstIndent int,
	indent int,
	maxLen int,
) (int, error) {
	firstLineFmt := fmt.Sprintf(
		"%s%%s\r\n", strings.Repeat(" ", firstIndent))
	restLineFmt := fmt.Sprintf(
		"%s%%s\r\n", strings.Repeat(" ", indent))
	firstLineMaxLen := maxLen - firstIndent
	restLineMaxLen := maxLen - indent
	isFirstLine := true

	dataLen := len(d)
	remainLen := dataLen
	lineStart := 0
	lineEnd := 0
	lineLen := 0
	lineRemain := 0
	segmentCursor := 0
	lastSegmentCursor := 0

	currentFmt := ""
	currentMaxLen := 0

	for {
		if lineStart >= dataLen {
			return dataLen, nil
		}

		currentBreakPos := i.findFirstLineBreaker(
			d[lineStart : lineStart+remainLen])

		if currentBreakPos == -1 {
			lineEnd = lineStart + remainLen
		} else {
			lineEnd = lineStart + currentBreakPos + 1
		}

		lineLen = lineEnd - lineStart
		lineRemain = lineLen

		lastSegmentCursor = lineStart
		segmentCursor = lineStart

		for {
			if lineRemain <= 0 {
				remainLen -= lineLen
				lineStart = lineEnd

				break
			}

			if isFirstLine {
				currentFmt = firstLineFmt
				currentMaxLen = firstLineMaxLen

				isFirstLine = false
			} else {
				currentFmt = restLineFmt
				currentMaxLen = restLineMaxLen
			}

			if lineRemain > currentMaxLen {
				currentSegmentBreakPos := i.findLastSpace(
					d[lastSegmentCursor : lastSegmentCursor+currentMaxLen])

				if currentSegmentBreakPos == -1 {
					segmentCursor = lastSegmentCursor + currentMaxLen
				} else {
					segmentCursor =
						lastSegmentCursor + currentSegmentBreakPos + 1
				}
			} else {
				segmentCursor = lastSegmentCursor + lineRemain
			}

			// Don't trim if it's the last output line
			if segmentCursor != dataLen {
				fmt.Fprintf(w, currentFmt, bytes.TrimSuffix(
					d[lastSegmentCursor:segmentCursor], newlinesTrim))
			} else {
				fmt.Fprintf(w, currentFmt, d[lastSegmentCursor:segmentCursor])
			}

			lineRemain -= segmentCursor - lastSegmentCursor
			lastSegmentCursor = segmentCursor
		}
	}
}
