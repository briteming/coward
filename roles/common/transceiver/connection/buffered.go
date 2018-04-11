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

package connection

import (
	"bufio"

	"github.com/reinit/coward/roles/common/network"
)

type buffered struct {
	network.Connection

	reader *bufio.Reader
}

func newBuffered(c network.Connection, size int) network.Connection {
	return buffered{
		Connection: c,
		reader:     bufio.NewReaderSize(c, size),
	}
}

func (f buffered) Read(b []byte) (int, error) {
	return f.reader.Read(b)
}