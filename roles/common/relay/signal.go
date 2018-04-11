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

package relay

import "strconv"

// Signal represents a Relay Signal
type Signal byte

// Consts for Signal
const (
	SignalData      Signal = 0xFB
	SignalError     Signal = 0xFC
	SignalCompleted Signal = 0xFD
	SignalClose     Signal = 0xFE
	SignalClosed    Signal = 0xFF
)

// String return name of a Signal value
func (s Signal) String() string {
	switch s {
	case SignalData:
		return "Data"
	case SignalError:
		return "Error"
	case SignalCompleted:
		return "Completed"
	case SignalClose:
		return "Close"
	case SignalClosed:
		return "Closed"
	}

	return strconv.FormatUint(uint64(s), 10)
}
