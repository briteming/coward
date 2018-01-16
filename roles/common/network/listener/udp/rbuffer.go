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

package udp

import "errors"

var (
	errRBufferNoMoreData = errors.New(
		"No more data in the Reader Buffer")
)

type rbuffer struct {
	buf       []byte
	remain    int
	completed chan struct{}
}

func (b *rbuffer) Read(p []byte) (int, error) {
	if b.remain <= 0 {
		return 0, errRBufferNoMoreData
	}

	copied := copy(p, b.buf[len(b.buf)-b.remain:])

	b.remain -= copied

	if b.remain <= 0 {
		<-b.completed
	}

	return copied, nil
}

func (b *rbuffer) Clear() {
	if b.remain <= 0 {
		return
	}

	b.remain = 0
	<-b.completed
}
