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

package logger

// Logger is the standard log of COWARD, not golang
type Logger interface {
	Context(name string) Logger
	Debugf(msg string, detail ...interface{})
	Infof(msg string, detail ...interface{})
	Warningf(msg string, detail ...interface{})
	Errorf(msg string, detail ...interface{})
	Write(b []byte) (int, error)
}

// Level representing the Level information
type Level byte

// Error level strings
const (
	Default Level = 0x0
	Debug   Level = 0x1
	Info    Level = 0x2
	Warning Level = 0x3
	Error   Level = 0x4
)

// String converts Level to string
func (l Level) String() string {
	switch l {
	case Default:
		return "DEF"

	case Debug:
		return "DBG"

	case Info:
		return "INF"

	case Warning:
		return "WRN"

	case Error:
		return "ERR"
	}

	return "UKN"
}
