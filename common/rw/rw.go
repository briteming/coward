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

package rw

import "io"

// Depleter depletes data from the reader
type Depleter interface {
	Depleted() bool
	Deplete() error
}

// Doner Finish reading without close the reader
type Doner interface {
	Done() error
}

// ReadWriteDepleteDoner io.ReadWriter + Depleter + Doner
type ReadWriteDepleteDoner interface {
	io.ReadWriter
	Depleter
	Doner
}

// Codec encodes and decodes data
type Codec interface {
	Decode(io.Reader) io.Reader
	Encode(io.Writer) WriteWriteAll
}

// WriteMax only write maxmius max bytes data to the writer
type WriteMax interface {
	WriteMax(w io.Writer, max int) (int, error)
}

// Remainer returns the remaining length
type Remainer interface {
	Remain() int
}

// WriteMaxRemainer WriteMax + Remainer
type WriteMaxRemainer interface {
	WriteMax
	Remainer
}

// WriteWriteAll io.Writer + WriteAll
type WriteWriteAll interface {
	io.Writer
	WriteAll
}
