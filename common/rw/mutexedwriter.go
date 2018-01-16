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

import (
	"io"
	"sync"
)

type mutexedWriter struct {
	w    io.Writer
	lock sync.Mutex
}

// NewMutexedWriter just like normal writer, but it the Write method
// is mutexed.
func NewMutexedWriter(w io.Writer) io.Writer {
	return &mutexedWriter{
		w:    w,
		lock: sync.Mutex{},
	}
}

// Write writes data to Mutexed Writer
func (m *mutexedWriter) Write(b []byte) (int, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.w.Write(b)
}
