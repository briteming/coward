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

import (
	"bytes"
	"testing"
)

/*
func TestIndentFindLastSpace(t *testing.T) {
	i := indent{}

	simaple := []byte("This is\ta\r\ntest")

	isbreak, foundPos := i.findLastBreak(simaple)

	if isbreak != true {

	}

	if string(simaple[:foundPos]) != "This is\ta\r" {
		t.Error("Can't get expected last space")

		return
	}
}*/

func TestIndentWriteIndent(t *testing.T) {
	test := []byte("Usage:\r\n\r\n" +
		"coward [Execute Options ...] <Role> [Role Options ...]\r\n")
	expected := " Usage:\r\n" +
		"    \r\n" +
		"    coward [Execute Options ...] <Role> [Role Options ...]\r\n" +
		"\r\n"
	w := bytes.NewBuffer(make([]byte, 0, 1024))
	i := indent{}

	i.writeIndent(w, test, 1, 4, 80)

	if w.String() != expected {
		t.Errorf("Failed to Write expected Indent."+
			"\r\nExpecting:\r\n\"%s\"\r\nGot:\r\n\"%s\"\r\n",
			expected, w.String())

		return
	}
}
