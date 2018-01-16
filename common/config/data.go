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

import (
	"errors"
	"reflect"
)

var zeroReflect = reflect.Value{}

// Configurator errors
var (
	ErrConfigurationMustBeStructPointer = errors.New(
		"Configuration must be a pointer which point to a struct")

	ErrConfigurationFieldUnsupportedKind = errors.New(
		"Data kind in field %s is unsupported")

	ErrConfigurationFieldUnsupportedDataType = errors.New(
		"Data type in field %s is unsupported")

	ErrConfigurationFieldTagNameConfilcted = errors.New(
		"Tag name of field %s is confilcted with another field")

	ErrConfigurationWithoutLabel = errors.New(
		"Configuration without label")

	ErrUndefinedParameter = errors.New(
		"Parameter is undefined")

	ErrInvalidField = errors.New(
		"Field is invalid")

	ErrInsufficientArray = errors.New(
		"Array Field is insufficient")
)
