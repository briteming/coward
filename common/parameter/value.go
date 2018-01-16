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

package parameter

// Value is parsed parameter value
type Value struct {
	token *token
	input []byte
}

func newValue(t *token, input []byte) *Value {
	return &Value{
		token: t,
		input: input,
	}
}

// Symbol returns current symbol
func (v *Value) Symbol() Symbol {
	return v.token.symbol
}

// Data returns a section of inputted data which represented by current
// Value
func (v *Value) Data() []byte {
	return v.token.fragment.unescape(v.input[v.token.start:v.token.end])
}

// Start returns start position of the data
func (v *Value) Start() int {
	return v.token.start
}

// End returns end position of the data
func (v *Value) End() int {
	return v.token.end
}

// Labels return all sub values grouped by lables
func (v *Value) Labels() []*Labelled {
	var currentLabelToken *token
	var currentLabelled *Labelled
	var hasRemain bool

	labels := make([]*Labelled, 0, len(v.token.sub))

	currentLabelled = &Labelled{
		current: v,
		label:   nil,
		values:  []*Value{},
	}

	for _, t := range v.token.sub {
		hasRemain = true

		if t.label == currentLabelToken && t.symbol != SymbolLabel {
			currentLabelled.values = append(
				currentLabelled.values, newValue(t, v.input))

			continue
		}

		if t.symbol == SymbolLabel {
			if len(currentLabelled.values) > 0 ||
				currentLabelled.label != nil {
				labels = append(labels, currentLabelled)
			}

			currentLabelToken = t
			currentLabelVal := newValue(currentLabelToken, v.input)

			currentLabelled = &Labelled{
				current: currentLabelVal,
				label:   currentLabelVal,
				values:  []*Value{},
			}
		} else {
			currentLabelled.values = append(
				currentLabelled.values, newValue(t, v.input))
		}
	}

	if hasRemain {
		labels = append(labels, currentLabelled)
	}

	return labels
}
