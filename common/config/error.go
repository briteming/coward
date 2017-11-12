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

package config

import (
	"fmt"

	"github.com/reinit/coward/common/parameter"
)

// FieldError indicating a wrong Configuration field
type FieldError struct {
	err   error
	field string
}

// ParseError indicating a field had bad value
type ParseError struct {
	parameter.ParseErrorBase
	err   error
	tag   string
	start int
	end   int
	input []byte
}

func newFieldError(err error, field string) error {
	return &FieldError{
		err:   err,
		field: field,
	}
}

func newParseError(
	err error, tag string, start int, end int, input []byte) error {
	return &ParseError{
		err:   err,
		tag:   tag,
		start: start,
		end:   end,
		input: input,
	}
}

// Is check if the input error is equals to underlying FieldError
func (s *FieldError) Is(err error) bool {
	if s.err != err {
		return false
	}

	return true
}

// Error returns a formated error string
func (s *FieldError) Error() string {
	return fmt.Sprintf(s.err.Error(), s.field)
}

// Is check if the input error is equals to underlying ParseError
func (s *ParseError) Is(err error) bool {
	if s.err != err {
		return false
	}

	return true
}

// Error returns a formated error string
func (s *ParseError) Error() string {
	if s.tag == "" {
		return fmt.Sprintf("Invalid Parameter: %s", s.err.Error())
	}

	return fmt.Sprintf("Invalid Parameter \"-%s\": %s",
		s.tag, s.err.Error())
}

// Sample returns a simple of bad input section which assigned that
// bad value to target field
func (s *ParseError) Sample(maxLen int) string {
	return s.MarkSection(s.input, s.start, s.end, maxLen)
}
