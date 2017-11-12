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

package parameter

// Symbol is the parameter parse symbol
type Symbol byte

// Symbols
const (
	SymbolLabel Symbol = 0x01
	SymbolArray Symbol = 0x02
	SymbolBlock Symbol = 0x03
	SymbolValue Symbol = 0x04
)

type token struct {
	start    int
	end      int
	label    *token
	fragment fragment
	symbol   Symbol
	sub      tokens
}

type tokens []*token
