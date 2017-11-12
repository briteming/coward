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

package common

import (
	"errors"
	"math"

	"github.com/reinit/coward/roles/common/network"
)

// Errors
var (
	ErrMappingMapNotFound = errors.New(
		"Can't found the map item")
)

// MapID Map OD
type MapID uint8

// Consts
const (
	MaxMapID = math.MaxUint8
)

// Mapped Item
type Mapped struct {
	Protocol network.Protocol
	Host     string
	Port     uint16
}

// Mapping contains Mapped Items
type Mapping [256]*Mapped

// Get returns the Mapped Item
func (m Mapping) Get(id MapID) (*Mapped, error) {
	if id > MaxMapID || m[id] == nil {
		return nil, ErrMappingMapNotFound
	}

	return m[id], nil
}
