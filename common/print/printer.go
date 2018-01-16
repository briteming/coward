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

import "io"

// printer implements Printer
type printer struct {
	base
	to       io.Writer
	maxWdith func() int
}

// Write writes data to io.Writer
func (p *printer) Write(b []byte) (int, error) {
	if p.to == nil {
		return 0, ErrPrinterAlreadyClosed
	}

	return p.to.Write(b)
}

// Writeln writes a line with indents to io.Writer
func (p *printer) Writeln(
	b []byte, firstIndent int, indent int, endPadding int) (int, error) {
	if p.to == nil {
		return 0, ErrPrinterAlreadyClosed
	}

	maxWidth := p.maxWdith() - endPadding
	maxIndent := firstIndent

	if maxIndent < indent {
		maxIndent = indent
	}

	if maxWidth <= 0 || maxWidth < maxIndent {
		return 0, ErrPrintWidthTooNarrow
	}

	return p.writeIndent(p.to, b, firstIndent, indent, maxWidth)
}

// MaxLen get max line width limit
func (p *printer) MaxLen() int {
	return p.maxWdith()
}

// Close shutdown printer
func (p *printer) Close() error {
	if p.to == nil {
		return ErrPrinterAlreadyClosed
	}

	p.to = nil

	return nil
}
