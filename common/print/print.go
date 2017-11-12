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

// print implements Print
type print struct {
	cfg Config
}

// New creates a new Print
func New(cfg Config) Print {
	return &print{
		cfg: cfg,
	}
}

// Printer returns a new Printer
func (p *print) Printer(to io.Writer) Printer {
	return &printer{
		to:       to,
		maxWdith: p.cfg.MaxLineWidth,
	}
}

// Buffer returns a new buffer
func (p *print) Buffer(header []byte, footer []byte) Buffer {
	return &buffer{
		buf:      bytes.Buffer{},
		header:   header,
		footer:   footer,
		maxWdith: p.cfg.MaxLineWidth,
	}
}
