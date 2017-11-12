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

package parameter

import "fmt"

// SyntaxError is the error happened when we parsing parameter string
type SyntaxError struct {
	ParseErrorBase
	err   error
	index int
	input []byte
}

func newSyntaxErr(err error, index int, input []byte) error {
	return &SyntaxError{
		err:   err,
		index: index,
		input: input,
	}
}

// Is check if inputted error is which current SyntaxError represented
func (s *SyntaxError) Is(err error) bool {
	if s.err != err {
		return false
	}

	return true
}

// Error returns formated error string
func (s *SyntaxError) Error() string {
	return fmt.Sprintf("Syntax error: %s", s.err.Error())
}

// Sample returns a sample if the bad parameter
func (s *SyntaxError) Sample(maxLen int) string {
	return s.PointPosition(s.input, s.index, maxLen)
}
