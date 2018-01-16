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

package application

import "errors"

const (
	aboutBanner = `
 <Name> v.<Version>

 <Copyright>`

	aboutPoweredByBanner = ` Powered by <COWARD:Name> v.<COWARD:Version>`

	helpUsage = "Usage:\r\n\r\n" +
		"%s [Execute Options ...] <Role> [Role Options ...]\r\n"

	helpUsageSlient = `-slient%sDisable output`
	helpUsageDebug  = `-debug %sEnable debug output`
	helpUsageDaemon = `-daemon%sRun as daemon`
	helpUsageLog    = `-log   %sWrite log to a file`
	helpUsageParam  = `-param %sLoad Role Options from a file`
)

// COWARD application errors
var (
	ErrLogFileMustBeSpecified = errors.New(
		"Log file must be specified")

	ErrConfigFileMustBeSpecified = errors.New(
		"Configuration file must be specified")

	ErrUnknownExecuteOption = errors.New(
		"At least one of the Execute Option is unknown")

	ErrExecuteOptionEndedBeforeRoleName = errors.New(
		"Role must be specified at the end of Execute Option")

	ErrFailedToOpenParameterFile = errors.New(
		"Failed to open Parameter File")

	ErrParameterFileEmpty = errors.New(
		"Parameter file is empty")
)

// Changeable declaration data like version etc
var (
	version = "dev"
)
