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

const treeInitEnd = -1

type tree struct {
	*token

	lastLabel *token
	fragment  fragment
	parent    *tree
	lastStart int
	level     int
}

func (tr *tree) isTail(c byte) bool {
	if tr.fragment.isTail(c) {
		return true
	}

	return false
}

func (tr *tree) isFall(c byte) bool {
	if tr.parent != nil && tr.parent.isTail(c) {
		return true
	}

	return false
}

func (tr *tree) append(t *token) {
	if t.start == t.end {
		return
	}

	tr.token.sub = append(tr.token.sub, t)
}

func (tr *tree) pop() *token {
	tsLen := len(tr.token.sub)

	t := tr.token.sub[tsLen-1]

	tr.token.sub = tr.token.sub[:tsLen-1]

	return t
}

func (tr *tree) last() *token {
	return tr.token.sub[len(tr.token.sub)-1]
}
