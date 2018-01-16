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

package role

import (
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/print"
)

// Components is some data needed by role
type Components []interface{}

// Generater creates a new role
type Generater func(
	screenOut print.Common,
	cfg interface{},
	log logger.Logger,
) (Role, error)

// Configurator creates new configuration for a role
type Configurator func(components Components) interface{}

// Registration use to declare the information needed for role registration
type Registration struct {
	Name         string
	Description  string
	Configurator Configurator
	Generater    Generater
}

// Registrations is a slice of Registration
type Registrations []Registration

// Registered contains information of registered roles
type Registered struct {
	Name        string
	Description string
	configuator Configurator
	generater   Generater
}

// Roles contains all registered roles
type Roles map[string]Registered
