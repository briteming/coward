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

package config

import "errors"

var (
	// ErrFieldNotFound is throwed when the field is not found
	ErrFieldNotFound = errors.New(
		"Field not found")
)

type fields []*field

type field struct {
	Name        string
	Path        string
	Tag         string
	Tags        []string
	Description string
	Sub         fields
}

// GetByTag will search and return field by given tag name
func (f *fields) GetByTag(tag string) (*field, error) {
	for _, field := range *f {
		for _, ftag := range field.Tags {
			if ftag != tag {
				continue
			}

			return field, nil
		}
	}

	return nil, ErrFieldNotFound
}

// GetByName will search and return field by given field name
func (f *fields) GetByName(name string) (*field, error) {
	for _, field := range *f {
		if field.Name != name {
			continue
		}

		return field, nil
	}

	return nil, ErrFieldNotFound
}
