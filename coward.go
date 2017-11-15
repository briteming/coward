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

package main

import (
	"os"

	"github.com/reinit/coward/application"
	"github.com/reinit/coward/roles/common/codec"
	"github.com/reinit/coward/roles/mapper"
	"github.com/reinit/coward/roles/project"
	"github.com/reinit/coward/roles/projector"
	"github.com/reinit/coward/roles/proxy"
	"github.com/reinit/coward/roles/socks5"
)

func main() {
	var execErr error

	app := application.New(application.Config{
		Banner:    "",
		Name:      "",
		Version:   "",
		Copyright: "",
		URL:       "",
		Components: application.Components{
			proxy.Role, socks5.Role, mapper.Role,
			projector.Role, project.Role,
			codec.Plain,
			codec.AESCFB128, codec.AESCFB256,
			codec.AESGCM128, codec.AESGCM256,
		},
	})

	switch len(os.Args) {
	case 0:
	case 1:
		execErr = app.Help()
	default:
		execErr = app.ExecuteArgumentInput(os.Args[1:])
	}

	if execErr != nil {
		os.Exit(1)
	}
}
