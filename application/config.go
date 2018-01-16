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

// SignalChan is something will be used for sending and
// receiving signals
type SignalChan chan bool

// SignalReceiveChan is read only SignalChan
type SignalReceiveChan chan<- bool

// Config is the COWARD config
type Config struct {
	Banner     string
	Name       string
	Version    string
	Copyright  string
	URL        string
	Components Components
}

// ExecuteConfig is command config of current COWARD application instance
type ExecuteConfig struct {
	Daemom    bool
	Slient    bool
	Debug     bool
	LogFile   string
	ParamFile string
	Shutdown  SignalChan
	Booted    SignalReceiveChan
}
