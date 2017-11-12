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

package rw

import "io"

// WriteFull keep writing until all data is written
func WriteFull(writer io.Writer, b []byte) (int, error) {
	writtenLen := 0
	totalSendLen := len(b)

	for {
		wLen, wErr := writer.Write(b[writtenLen:totalSendLen])

		writtenLen += wLen

		if wErr != nil {
			return writtenLen, wErr
		}

		if writtenLen < totalSendLen {
			continue
		}

		return writtenLen, nil
	}
}
