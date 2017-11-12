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

package print

import (
	"bytes"
	"io"
)

// buffer implements Buffer
type buffer struct {
	base
	buf      bytes.Buffer
	header   []byte
	footer   []byte
	maxWdith func() int
}

// Write writes data to buffer
func (w *buffer) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

// WriteTo export data to a io.Writer
func (w *buffer) WriteTo(to io.Writer) (int64, error) {
	to.Write(w.header)

	defer to.Write(w.footer)

	return w.buf.WriteTo(to)
}

// Writeln writes a line width indent
func (w *buffer) Writeln(
	b []byte, firstIndent int, indent int, endPadding int) (int, error) {
	maxWidth := w.maxWdith() - endPadding
	maxIndent := firstIndent

	if maxIndent < indent {
		maxIndent = indent
	}

	if maxWidth <= 0 || maxWidth < maxIndent {
		return 0, ErrPrintWidthTooNarrow
	}

	return w.writeIndent(&w.buf, b, firstIndent, indent, maxWidth)
}

// MaxLen get max line width limit
func (w *buffer) MaxLen() int {
	return w.maxWdith()
}
