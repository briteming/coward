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
	"sync"
	"time"

	"github.com/reinit/coward/common/print"
)

type screen struct {
	context string
	writer  print.Common
	wlocker *sync.Mutex
}

type screenNonDebug struct {
	screen
}

const (
	screenLogFormatStr = "[%-3.3s] %-28.28s %s\r\n%s\r\n"
)

// NewScreen creates a logger that will print errors on screen
func NewScreen(w print.Common) Logger {
	return &screen{
		context: "COWARD",
		writer:  w,
		wlocker: &sync.Mutex{},
	}
}

// NewScreenNonDebug creates a screen printing logger which will not
// print debug message
func NewScreenNonDebug(w print.Common) Logger {
	return &screenNonDebug{
		screen: screen{
			context: "COWARD",
			writer:  w,
			wlocker: &sync.Mutex{},
		},
	}
}

// Context enter a new level of log context
func (s *screen) Context(ctx string) Logger {
	return &screen{
		context: s.context + " > " + ctx,
		writer:  s.writer,
		wlocker: s.wlocker,
	}
}

func (s *screen) print(level Level, msg string, detail []interface{}) {
	s.wlocker.Lock()

	defer s.wlocker.Unlock()

	if len(detail) > 0 {
		s.writer.Writeln([]byte(fmt.Sprintf(screenLogFormatStr,
			level, time.Now().Format(time.UnixDate), s.context,
			fmt.Sprintf(msg, detail...))), 1, 7, 1)

		return
	}

	s.writer.Writeln([]byte(fmt.Sprintf(screenLogFormatStr,
		level, time.Now().Format(time.UnixDate), s.context, msg)), 1, 7, 1)
}

// Debugf prints debug message
func (s *screen) Debugf(msg string, detail ...interface{}) {
	s.print(Debug, msg, detail)
}

// Infof prints general message
func (s *screen) Infof(msg string, detail ...interface{}) {
	s.print(Info, msg, detail)
}

// Warningf prints warning message
func (s *screen) Warningf(msg string, detail ...interface{}) {
	s.print(Warning, msg, detail)
}

// Errorf prints error information
func (s *screen) Errorf(msg string, detail ...interface{}) {
	s.print(Error, msg, detail)
}

// Write prints default information
func (s *screen) Write(b []byte) (int, error) {
	s.print(Default, string(b), nil)

	return len(b), nil
}

// Debugf ditchs debug message
func (s *screenNonDebug) Debugf(msg string, detail ...interface{}) {}

// Context enter a new level of log context
func (s *screenNonDebug) Context(ctx string) Logger {
	return &screenNonDebug{
		screen: screen{
			context: s.context + " > " + ctx,
			writer:  s.writer,
			wlocker: s.wlocker,
		},
	}
}
