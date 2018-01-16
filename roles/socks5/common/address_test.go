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

package common

import (
	"bytes"
	"testing"
)

type addressTestCase struct {
	Data         []byte
	ExpectedType AType
	ExpectedRLen int64
	ExpectedAddr []byte
	ExpectedPort uint16
}

func TestAddressReadFrom(t *testing.T) {
	testCase := []addressTestCase{
		addressTestCase{
			Data: []byte{
				3, 7, 111, 97, 116, 115, 46, 112, 119, 31, 152},
			ExpectedType: ATypeHost,
			ExpectedAddr: []byte{111, 97, 116, 115, 46, 112, 119},
			ExpectedPort: 8088,
		},
		addressTestCase{
			Data: []byte{
				1, 127, 0, 0, 1, 31, 152},
			ExpectedType: ATypeIPv4,
			ExpectedAddr: []byte{127, 0, 0, 1},
			ExpectedPort: 8088,
		},
		addressTestCase{
			Data: []byte{
				4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 31, 152},
			ExpectedType: ATypeIPv6,
			ExpectedAddr: []byte{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
			ExpectedPort: 8088,
		},
	}

	for v := range testCase {
		addr := Address{}

		rLen, rErr := addr.ReadFrom(bytes.NewBuffer(testCase[v].Data))

		if rErr != nil {
			t.Error("Failed to ReadFrom source due to error:", rErr)

			return
		}

		if rLen != int64(len(testCase[v].Data)) {
			t.Errorf("Expect readed length to be %d, got %d",
				len(testCase[v].ExpectedAddr), rLen)

			return
		}

		if addr.AType != testCase[v].ExpectedType {
			t.Errorf("Expect Address Type to be %d, got %d",
				testCase[v].ExpectedType, addr.AType)

			return
		}

		if !bytes.Equal(addr.Address, testCase[v].ExpectedAddr) {
			t.Errorf("Expect Address Data to be %d, got %d",
				testCase[v].ExpectedAddr, addr.Address)

			return
		}

		if addr.Port != testCase[v].ExpectedPort {
			t.Errorf("Expect Address Port to be %d, got %d",
				testCase[v].ExpectedPort, addr.Port)

			return
		}
	}
}
