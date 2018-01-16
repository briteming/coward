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
	"errors"
	"fmt"
	"strings"

	"github.com/reinit/coward/common/config"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/print"
)

// Roler errors
var (
	ErrExisted = errors.New(
		"Role already existed")

	ErrNameMustBeDeclared = errors.New(
		"Role name must be declared")

	ErrNotExisted = errors.New(
		"Role not existed")

	ErrNoOperation = errors.New(
		"No operation")
)

// Roler is the role manager
type Roler interface {
	Init(
		screenOut print.Common,
		name string,
		configuration interface{},
		log logger.Logger,
	) (Role, error)
	InitParameterString(
		screenOut print.Common,
		name string,
		parameters []byte,
		log logger.Logger,
	) (Role, error)
	List(screenOut print.Common)
	MaxRoleNameLen() int
}

// roler implements Roler
type roler struct {
	roles          Roles
	maxRoleNameLen int
	config         Config
}

// NewRoler creates a new Roler according to setting
func NewRoler(config Config) (Roler, error) {
	r := &roler{
		roles:  Roles{},
		config: config,
	}

	for _, reg := range config.Roles {
		regErr := r.register(reg)

		if regErr == nil {
			continue
		}

		return nil, regErr
	}

	return r, nil
}

func (r *roler) register(register Registration) error {
	if register.Name == "" {
		return ErrNameMustBeDeclared
	}

	_, existed := r.roles[register.Name]

	if existed {
		return ErrExisted
	}

	r.roles[register.Name] = Registered{
		Name:        register.Name,
		Description: register.Description,
		configuator: register.Configurator,
		generater:   register.Generater,
	}

	nameLen := len(register.Name)

	if r.maxRoleNameLen < nameLen {
		r.maxRoleNameLen = nameLen
	}

	return nil
}

func (r *roler) init(
	screenOut print.Common,
	role Registered,
	configuration interface{},
	log logger.Logger,
) (Role, error) {
	newRole, genErr := role.generater(screenOut, configuration, log)

	if genErr != nil {
		return nil, genErr
	}

	return newRole, nil
}

// Init initialize a new Role with configuration
func (r *roler) Init(
	screenOut print.Common,
	name string,
	configuration interface{},
	log logger.Logger,
) (Role, error) {
	role, existed := r.roles[name]

	if !existed {
		r.config.OnUndefined(screenOut, name, r.maxRoleNameLen, r.roles)

		return nil, ErrNotExisted
	}

	return r.init(screenOut, role, configuration, log)
}

// MaxRoleNameLen returns the length of the longest Role
func (r *roler) MaxRoleNameLen() int {
	return r.maxRoleNameLen
}

// Init initialize a new Role with parameter string
func (r *roler) InitParameterString(
	screenOut print.Common,
	name string,
	parameters []byte,
	log logger.Logger,
) (Role, error) {
	var configuator config.Configurator
	var configuration interface{}
	var configuatorErr error

	role, existed := r.roles[name]

	if !existed {
		r.config.OnUndefined(screenOut, name, r.maxRoleNameLen, r.roles)

		return nil, ErrNotExisted
	}

	if role.configuator != nil {
		configuration = role.configuator(r.config.Components)

		configuator, configuatorErr = config.Import(configuration)

		if configuatorErr != nil {
			return nil, configuatorErr
		}
	}

	switch strings.ToLower(string(parameters)) {
	case "-h":
		fallthrough
	case "-help":
		fallthrough
	case "help":
		fallthrough
	case "?":
		fallthrough
	case "/?":
		fallthrough
	case "-?":
		if configuator != nil {
			screenOut.Writeln([]byte(fmt.Sprintf(
				"\"%s\" has following options:\r\n", name)), 1, 2, 1)

			configuator.Help(screenOut)
		} else {
			screenOut.Writeln([]byte(fmt.Sprintf(
				"\"%s\" requires no option.\r\n", name)), 1, 2, 1)
		}

		return nil, ErrNoOperation

	default:
		if configuator != nil && len(parameters) == 0 {
			screenOut.Writeln([]byte(fmt.Sprintf(
				"\"%s\" needs to be configured by following options:\r\n",
				name)), 1, 2, 1)

			configuator.Help(screenOut)

			return nil, ErrNoOperation
		}
	}

	if configuator != nil {
		cfgParseErr := configuator.Parse(parameters)

		if cfgParseErr != nil {
			return nil, cfgParseErr
		}
	}

	return r.init(screenOut, role, configuration, log)
}

func (r *roler) List(screenOut print.Common) {
	r.config.OnListScreen(screenOut, r.maxRoleNameLen, r.roles)
}
