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

import (
	"fmt"
	"io"
	"time"
)

type write struct {
	context string
	writer  io.Writer
}

type writeNonDebug struct {
	write
}

const (
	writeFormatStr = "%s [%s] %s | %s\r\n"
)

// NewWrite creates a logger that will write log to a io.Writer
func NewWrite(w io.Writer) Logger {
	return &write{
		context: "COWARD",
		writer:  w,
	}
}

// NewWriteNonDebug creates a writing logger that will not write debug message
func NewWriteNonDebug(w io.Writer) Logger {
	return &writeNonDebug{
		write: write{
			context: "COWARD",
			writer:  w,
		},
	}
}

// Context enter a new level of log context
func (s *write) Context(ctx string) Logger {
	return &write{
		context: s.context + " > " + ctx,
		writer:  s.writer,
	}
}

func (s *write) print(level Level, msg string, detail []interface{}) {
	// Notice that since we only print one line to the file,
	// no need to lock this method
	if len(detail) > 0 {
		fmt.Fprintf(s.writer, writeFormatStr,
			level, time.Now().Format(time.UnixDate), s.context,
			fmt.Sprintf(msg, detail...))

		return
	}

	fmt.Fprintf(s.writer, writeFormatStr,
		level, time.Now().Format(time.UnixDate), s.context, msg)
}

// Debugf writes debug message
func (s *write) Debugf(msg string, detail ...interface{}) {
	s.print(Debug, msg, detail)
}

// Infof writes general message
func (s *write) Infof(msg string, detail ...interface{}) {
	s.print(Info, msg, detail)
}

// Warningf writes warning message
func (s *write) Warningf(msg string, detail ...interface{}) {
	s.print(Warning, msg, detail)
}

// Errorf writes error information
func (s *write) Errorf(msg string, detail ...interface{}) {
	s.print(Error, msg, detail)
}

// Write prints default information
func (s *write) Write(b []byte) (int, error) {
	s.print(Default, string(b), nil)

	return len(b), nil
}

// Debugf ditchs debug message
func (s *writeNonDebug) Debugf(msg string, detail ...interface{}) {}

// Context enter a new level of log context
func (s *writeNonDebug) Context(ctx string) Logger {
	return &writeNonDebug{
		write: write{
			context: s.context + " > " + ctx,
			writer:  s.writer,
		},
	}
}
