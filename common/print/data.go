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

package print

import (
	"errors"
	"io"
)

// Print is the printer builder
type Print interface {
	Printer(to io.Writer) Printer
	Buffer(header []byte, footer []byte) Buffer
}

// Common is interfaces shared by both Printer and buffer
type Common interface {
	io.Writer
	Writeln(b []byte, firstIndent int, indent int, endPadding int) (int, error)
	MaxLen() int
}

// Printer is the printer interface
type Printer interface {
	Common
	Close() error
}

// Buffer is the buffer interface
type Buffer interface {
	Common
	io.WriterTo
}

// Print errors
var (
	ErrPrintWidthTooNarrow = errors.New(
		"Print width is too narrow")

	ErrPrinterAlreadyClosed = errors.New(
		"Printer already closed")
)
