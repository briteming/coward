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

import "bytes"

type fragment struct {
	head        byte
	tails       []byte
	escapes     []byte
	keepSeeking bool
}

var (
	labelFragment = fragment{
		head:        '-',
		tails:       []byte{' ', '\r', '\n', '\t'},
		escapes:     nil,
		keepSeeking: false,
	}

	valueFragment = fragment{
		head:        0x00,
		tails:       []byte{' ', '\r', '\n', '\t'},
		escapes:     nil,
		keepSeeking: false,
	}

	doubleQuoteFragment = fragment{
		head:        '"',
		tails:       []byte{'"'},
		escapes:     []byte{'"'},
		keepSeeking: true,
	}

	quoteFragment = fragment{
		head:        '\'',
		tails:       []byte{'\''},
		escapes:     []byte{'\''},
		keepSeeking: true,
	}

	blockFragment = fragment{
		head:        '{',
		tails:       []byte{'}'},
		escapes:     nil,
		keepSeeking: false,
	}
)

func (s *fragment) unescape(inputStr []byte) []byte {
	result := make([]byte, len(inputStr))

	copy(result, inputStr)

	for _, char := range s.escapes {
		result = bytes.Replace(result, []byte{ESCAPE, char}, []byte{char}, -1)
	}

	result = bytes.Replace(result, []byte{ESCAPE, ESCAPE}, []byte{ESCAPE}, -1)

	return result
}

func (s *fragment) needEscape(input []byte, currentIndex int) bool {
	isEscaper := false
	isEscape := false

	if currentIndex <= 0 {
		return false
	}

	if s.escapes == nil {
		return false
	}

	for _, escaper := range s.escapes {
		if escaper != input[currentIndex] {
			continue
		}

		isEscaper = true

		break
	}

	if isEscaper {
		for eSeek := currentIndex - 1; eSeek >= 0; eSeek-- {
			if input[eSeek] != ESCAPE {
				break
			}

			if !isEscape {
				isEscape = true
			} else {
				isEscape = false
			}
		}

		return isEscape
	}

	if input[currentIndex] == ESCAPE {
		return true
	}

	return false
}

func (s *fragment) isTail(c byte) bool {
	if s.tails[0] == 0xff {
		return false
	}

	for _, ch := range s.tails {
		if c == ch {
			return true
		}
	}

	return false
}
