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

package plain

import (
	"io"

	"github.com/reinit/coward/common/rw"
)

type encoder struct {
	w io.Writer
}

func (e encoder) Write(b []byte) (int, error) {
	return e.WriteAll(b)
}

func (e encoder) WriteAll(b ...[]byte) (int, error) {
	totalWriteLen := 0

	for _, s := range b {
		wLen, wErr := rw.WriteFull(e.w, s)

		totalWriteLen += wLen

		if wErr == nil {
			continue
		}

		return totalWriteLen, wErr
	}

	return totalWriteLen, nil
}
