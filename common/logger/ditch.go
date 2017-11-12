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

package logger

type ditch struct{}

// NewDitch creates a logger that will ... ditch all the logs
func NewDitch() Logger {
	return &ditch{}
}

// Context enter a new level of log context
func (s *ditch) Context(ctx string) Logger {
	return &ditch{}
}

// Debugf ditchs debug message
func (s *ditch) Debugf(msg string, detail ...interface{}) {}

// Infof ditchs general message
func (s *ditch) Infof(msg string, detail ...interface{}) {}

// Warningf ditchs warning message
func (s *ditch) Warningf(msg string, detail ...interface{}) {}

// Errorf ditchs error information
func (s *ditch) Errorf(msg string, detail ...interface{}) {}

// Write ditchs default information
func (s *ditch) Write(b []byte) (int, error) { return len(b), nil }
