//  Crypto-Obscured Forwarder
//
//  Copyright (C) 2018 Rui NI <ranqus@gmail.com>
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

package parameter

import (
	"bytes"
	"fmt"
)

const parseErrorFormat = "%s%s%s\r\n%s%s"
const parseErrorSpace = " "
const parseErrorDots = "..."
const parseErrorDotsLen = 3

// ParseErrorBase contains some common functions which will be used
// by parameter related errors
type ParseErrorBase struct{}

// PointPosition return a two line string:
// Line 1 showing Code sample
// Line 2 showing a pointer points to the position of bad part of the code
func (f *ParseErrorBase) PointPosition(
	input []byte, pos int, maxLen int) string {
	const pointer = "^"

	var headDots []byte
	var tailDots []byte

	var sectionStart int
	var sectionEnd int
	var previewStart int
	var previewEnd int

	inputLen := len(input)
	endBackShift := 0

	headNewLine := bytes.LastIndexByte(input[:pos], '\n')

	if headNewLine < 0 {
		sectionStart = 0
	} else {
		sectionStart = headNewLine + 1
	}

	tailNewLine := bytes.IndexAny(input[pos:], "\r\n")

	if tailNewLine < 0 {
		sectionEnd = inputLen
	} else {
		sectionEnd = pos + tailNewLine
	}

	previewStart = sectionStart
	previewEnd = sectionEnd

	if previewEnd-previewStart > maxLen {
		halfMax := maxLen / 2

		previewHeadGap := pos - previewStart

		if previewHeadGap > halfMax {
			previewStart += previewHeadGap - halfMax
		}

		previewEnd = previewStart + maxLen

		if previewEnd > sectionEnd {
			previewBackShift := previewEnd - sectionEnd

			previewStart -= previewBackShift
			previewEnd -= previewBackShift
		}
	}

	if previewStart > 0 {
		if previewStart+parseErrorDotsLen > pos {
			endBackShift = parseErrorDotsLen
		} else if (previewEnd-previewStart)+parseErrorDotsLen > maxLen {
			previewStart += parseErrorDotsLen
		}

		headDots = []byte(parseErrorDots)
	}

	if previewEnd < inputLen {
		tailDots = []byte(parseErrorDots)

		previewEnd -= parseErrorDotsLen
		previewEnd -= endBackShift
	}

	if previewEnd < previewStart {
		return ""
	}

	return fmt.Sprintf(
		parseErrorFormat,
		headDots, input[previewStart:previewEnd], tailDots,
		bytes.Repeat(
			[]byte(parseErrorSpace),
			(pos-previewStart)+len(headDots)),
		pointer)
}

// MarkSection return a two line string:
// Line 1 showing Code sample
// Line 2 showing a wave line to mark a section of the bad code
func (f *ParseErrorBase) MarkSection(
	input []byte, start int, end int, maxLen int) string {
	const wave = "~"

	var headDots []byte
	var tailDots []byte
	var sectionStart int
	var sectionEnd int
	var previewStart int
	var previewEnd int

	inputLen := len(input)
	codeStart := start
	codeEnd := end
	endBackShift := 0

	headNewLine := bytes.LastIndexByte(input[:codeStart], '\n')

	if headNewLine < 0 {
		sectionStart = 0
	} else {
		sectionStart = headNewLine + 1
	}

	tailNewLine := bytes.IndexAny(input[codeEnd:], "\r\n")

	if tailNewLine < 0 {
		sectionEnd = inputLen
	} else {
		sectionEnd = end + tailNewLine
	}

	midNewLine := bytes.IndexAny(input[codeStart:codeEnd], "\r\n")

	if midNewLine >= 0 {
		sectionEnd = start + midNewLine
		codeEnd = sectionEnd
	}

	previewStart = sectionStart
	previewEnd = sectionEnd

	if previewEnd-previewStart > maxLen {
		halfMax := maxLen / 2

		previewHeadGap := codeStart - previewStart

		if previewHeadGap > halfMax {
			previewStart += previewHeadGap - halfMax
		}

		previewEnd = previewStart + maxLen

		if previewEnd > sectionEnd {
			previewBackShift := previewEnd - sectionEnd

			previewStart -= previewBackShift
			previewEnd -= previewBackShift
		}
	}

	if previewStart > 0 {
		if previewStart+parseErrorDotsLen > codeStart {
			endBackShift = parseErrorDotsLen
		} else if (previewEnd-previewStart)+parseErrorDotsLen > maxLen {
			previewStart += parseErrorDotsLen
		}

		headDots = []byte(parseErrorDots)
	}

	if previewEnd < inputLen {
		tailDots = []byte(parseErrorDots)

		previewEnd -= parseErrorDotsLen
		previewEnd -= endBackShift
	}

	if previewStart > codeStart {
		codeStart = previewStart
	}

	if previewEnd < codeEnd {
		codeEnd = previewEnd
	}

	if previewEnd < previewStart {
		return ""
	}

	waveLineLen := codeEnd - codeStart

	if waveLineLen <= 0 {
		return ""
	}

	return fmt.Sprintf(
		parseErrorFormat,
		headDots, input[previewStart:previewEnd], tailDots,
		bytes.Repeat(
			[]byte(parseErrorSpace),
			(codeStart-previewStart)+len(headDots)),
		bytes.Repeat([]byte(wave), waveLineLen),
	)
}
