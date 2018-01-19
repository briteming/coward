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

package socks5

import (
	"errors"
	"net"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/print"
	"github.com/reinit/coward/common/role"
	tcpconn "github.com/reinit/coward/roles/common/network/connection/tcp"
	"github.com/reinit/coward/roles/common/network/dialer/tcp"
	tcplisten "github.com/reinit/coward/roles/common/network/listener/tcp"
	"github.com/reinit/coward/roles/common/transceiver"
	tclient "github.com/reinit/coward/roles/common/transceiver/client"
)

// ConfigProxy Proxy configurations
type ConfigProxy struct {
	components     []interface{}
	selectedCodec  transceiver.Codec
	Host           string `json:"host" cfg:"h,-host:Host name of the remote COWARD Proxy server.\r\n\r\nMust matchs the setting on server."`
	Port           uint16 `json:"port" cfg:"p,-port:Port number of the remote COWARD Proxy server.\r\n\r\nMust matchs the setting on server."`
	Connections    uint32 `json:"connections" cfg:"c,-connections:The maximum concurrent connections that can be established to a COWARD Proxy server."`
	RequestRetries uint8  `json:"retries" cfg:"r,-retries:How many times a failed Initial request can be retried."`
	Timeout        uint16 `json:"timeout" cfg:"t,-timeout:The maximum idle time in second of the established proxy connection.\r\n\r\nIf the proxy connection consecutively idle during this period of time, then that connection will be considered as inactive and thus be disconnected.\r\n\r\nIt is recommended to set this value no greater than the related one on the COWARD Proxy server setting."`
	RequestTimeout uint16 `json:"request_timeout" cfg:"rt,-request-timeout:The maximum wait time in second for the server to respond the Initial request of a client.\r\n\r\nIf the COWARD Proxy server has failed to respond the Initial request within this period of time, the connection will be considered broken and thus be closed.\r\n\r\nIt is recommended to set this value slightly greater than the \"--initial-timeout\" setting on the COWARD Proxy server."`
	Channels       uint8  `json:"channels" cfg:"n,-channels:How many requests can be simultaneously opened on a single established connection.\r\n\r\nSet the value greater than 1 so a single connection can be use to transport multiple requests (Multiplexing).\r\n\r\nWARNING:\r\nThis value must matchs or smaller than the related setting on the COWARD Proxy server, otherwise the request will be come malformed and thus dropped."`
	Persistent     bool   `json:"persist" cfg:"k,-persist:Whether or not to keep the connection to the COWARD Proxy active after all requests on the connection is completed."`
	Codec          string `json:"codec" cfg:"e,-codec:Specify which Codec will be used to encode and decode data payload to and from a connection."`
	CodecSetting   string `json:"codec_setting" cfg:"es,-codec-cfg:Configuration of the Codec.\r\n\r\nThe actual configuration format of this setting is depend on the Codec of your choosing."`
}

// Init inits the configuration
func (c *ConfigProxy) Init(parent *ConfigInput) {
	c.components = parent.components
}

// VerifyPort Verify Port
func (c *ConfigProxy) VerifyPort() error {
	if c.Port < 1 {
		return errors.New("Port must be greater than 0")
	}

	return nil
}

// VerifyConnections Verify Connections
func (c *ConfigProxy) VerifyConnections() error {
	if c.Connections < 1 {
		return errors.New("Connections must be greater than 0")
	}

	return nil
}

// VerifyRequestRetries Verify RequestRetries
func (c *ConfigProxy) VerifyRequestRetries() error {
	if c.RequestRetries < 1 {
		return errors.New("Retries must be greater than 0")
	}

	return nil
}

// VerifyTimeout Verify Timeout
func (c *ConfigProxy) VerifyTimeout() error {
	if c.Timeout < c.RequestTimeout {
		return errors.New(
			"(Idle) Timeout must be greater than the Request Timeout")
	}

	return nil
}

// VerifyRequestTimeout Verify RequestTimeout
func (c *ConfigProxy) VerifyRequestTimeout() error {
	if c.RequestTimeout > c.Timeout {
		return errors.New(
			"Request Timeout must be smaller than the (Idle) Timeout")
	}

	return nil
}

// VerifyChannels Verify Channels
func (c *ConfigProxy) VerifyChannels() error {
	if c.Channels < 1 {
		return errors.New("Channels must be greater than 0")
	}

	return nil
}

// VerifyCodec Verify Codec
func (c *ConfigProxy) VerifyCodec() error {
	for cIdx := range c.components {
		codecBuilder, isCodecBuilder :=
			c.components[cIdx].(func() transceiver.Codec)

		if !isCodecBuilder {
			continue
		}

		codecInfo := codecBuilder()

		if codecInfo.Name == c.Codec {
			c.selectedCodec = codecInfo

			return nil
		}
	}

	return errors.New("Specified Codec was not found")
}

// VerifyCodecSetting Verify CodecSetting
func (c *ConfigProxy) VerifyCodecSetting() error {
	if c.selectedCodec.Verify == nil {
		return errors.New("Codec must be specified")
	}

	return c.selectedCodec.Verify([]byte(c.CodecSetting))
}

// Verify Verifies
func (c *ConfigProxy) Verify() error {
	if c.Host == "" {
		return errors.New("Host must be defined")
	}

	if c.Port <= 0 {
		return errors.New("Port must be defined")
	}

	if c.Connections <= 0 {
		return errors.New("Connections must be defined")
	}

	if c.RequestRetries <= 0 {
		c.RequestRetries = 3
	}

	if c.Timeout <= 0 {
		return errors.New("(Idle) Timeout must be defined")
	}

	if c.RequestTimeout <= 0 {
		if c.Timeout <= 10 {
			c.RequestTimeout = 1
		} else {
			c.RequestTimeout = c.Timeout / 10
		}
	}

	if c.Channels <= 0 {
		return errors.New("Channels must be defined")
	}

	if c.Codec == "" {
		return errors.New("Codec must be defined")
	}

	if c.selectedCodec.Verify != nil {
		vErr := c.selectedCodec.Verify([]byte(c.CodecSetting))

		if vErr != nil {
			return errors.New("Codec Setting was invalid: " + vErr.Error())
		}
	}

	return nil
}

// ConfigAccount Socks5 accounts
type ConfigAccount struct {
	Username string `json:"username" cfg:"u,-user:Login name of the Socks5 account."`
	Password string `json:"password" cfg:"p,-pass:Password of the Socks5 account."`
}

// VerifyUsername Verify Username
func (c *ConfigAccount) VerifyUsername() error {
	if c.Username == "" {
		return errors.New("Username must be defined")
	}

	return nil
}

// VerifyPassword Verify Password
func (c *ConfigAccount) VerifyPassword() error {
	if c.Password == "" {
		return errors.New("Password must be defined")
	}

	return nil
}

// Verify Verifies
func (c *ConfigAccount) Verify() error {
	if c.Username == "" {
		return errors.New("Username must be defined")
	}

	if c.Password == "" {
		return errors.New("Password must be defined")
	}

	return nil
}

// ConfigInput Configuration
type ConfigInput struct {
	components        []interface{}
	selectedInterface net.IP
	Proxies           []ConfigProxy   `json:"proxies" cfg:"r,-proxies:Specify a set of remote COWARD Proxy servers.\r\n\r\nRequest will be dispatched to one of these proxies automatically."`
	Interface         string          `json:"interface" cfg:"i,-interface:Specify a local network interface to serve the Socks5 server."`
	Port              uint16          `json:"port" cfg:"p,-port:Specify a port to serve the Socks5 server"`
	Timeout           uint16          `json:"timeout" cfg:"t,-timeout:The maximum idle time in second of a Socks5 client connection.\r\n\r\nIf server consecutively receives no data from a connection during this period of time, then that connection will be considered as inactive and thus be disconnected."`
	InitialTimeout    uint16          `json:"initial_timeout" cfg:"it,-initial-timeout:The maximum wait time in second for Socks5 clients to finish Handshake.\r\n\r\nA well balanced value is required: You need to give clients plenty of time to finish the Initial request (Otherwise they may never be able to connect), and also be able defending against malicious accesses (By time them out) at same time."`
	Capacity          uint32          `json:"capicty" cfg:"c,-capacity:The maximum connections this Socks5 server can accept.\r\n\r\nWhen amount of connections reached this limitation, new incoming connection will be dropped."`
	Account           []ConfigAccount `json:"account" cfg:"a,-accounts:Accounts of the Socks5 server.\r\n\r\nOnce defined, the Socks5 server will require user authentication before relaying the request."`
}

// GetDescription gets description
func (c ConfigInput) GetDescription(fieldPath string) string {
	result := ""

	switch fieldPath {
	case "/Interface":
		ifAddrs, ifAddrsErr := net.InterfaceAddrs()

		if ifAddrsErr != nil {
			return ""
		}

		result = "Available network interfaces:\r\n- 0.0.0.0"

		for idIdx := range ifAddrs {
			ifIP, _, ifIPErr := net.ParseCIDR(ifAddrs[idIdx].String())

			if ifIPErr != nil {
				continue
			}

			result += "\r\n- " + ifIP.String()
		}

	case "/Proxies/Codec":
		result = "Available codecs:"

		for cIdx := range c.components {
			codecBuilder, isCodecBuilder :=
				c.components[cIdx].(func() transceiver.Codec)

			if !isCodecBuilder {
				continue
			}

			codecInfo := codecBuilder()

			result += "\r\n- " + codecInfo.Name
		}
	}

	return result
}

// VerifyInterface Verify Interface
func (c *ConfigInput) VerifyInterface() error {
	listenIP := net.ParseIP(c.Interface)

	if listenIP == nil {
		return errors.New("Invalid IP address")
	}

	c.selectedInterface = listenIP

	return nil
}

// VerifyPort Verify Port
func (c *ConfigInput) VerifyPort() error {
	if c.Port <= 0 {
		return errors.New("Port number must be greater than 0")
	}

	return nil
}

// VerifyTimeout Verify Timeout
func (c *ConfigInput) VerifyTimeout() error {
	if c.Timeout < c.InitialTimeout {
		return errors.New(
			"(Idle) Timeout must be greater than the Request Timeout")
	}

	return nil
}

// VerifyInitialTimeout Verify InitialTimeout
func (c *ConfigInput) VerifyInitialTimeout() error {
	if c.InitialTimeout > c.Timeout {
		return errors.New(
			"Request Timeout must be smaller than the (Idle) Timeout")
	}

	return nil
}

// VerifyCapicty Verify Capicty
func (c *ConfigInput) VerifyCapicty() error {
	if c.Capacity <= 0 {
		return errors.New("Capicty must be greater than 0")
	}

	return nil
}

// Verify Verifies
func (c *ConfigInput) Verify() error {
	if len(c.Proxies) <= 0 {
		return errors.New("At least one Proxy is required")
	}

	if c.Interface == "" {
		c.selectedInterface = net.ParseIP("127.0.0.1")
	}

	if c.Timeout <= 0 {
		return errors.New("(Idle) Timeout must be specified")
	}

	if c.InitialTimeout <= 0 {
		if c.Timeout <= 10 {
			c.InitialTimeout = 1
		} else {
			c.InitialTimeout = c.Timeout / 10
		}
	}

	if c.Capacity <= 0 {
		return errors.New("Capicty must be specified")
	}

	return nil
}

// Role register
func Role() role.Registration {
	return role.Registration{
		Name: "socks5",
		Description: "A Socks5 server that will convert Socks5 Requests to " +
			"COWARD Proxy Requests and send them to a COWARD Proxy server",
		Configurator: func(components role.Components) interface{} {
			return &ConfigInput{
				components:        components,
				selectedInterface: nil,
				Proxies:           []ConfigProxy{},
				Interface:         "",
				Port:              0,
				Timeout:           0,
				InitialTimeout:    0,
				Capacity:          0,
			}
		},
		Generater: func(
			w print.Common,
			config interface{},
			log logger.Logger,
		) (role.Role, error) {
			cfg := config.(*ConfigInput)

			listen := tcplisten.New(
				cfg.selectedInterface,
				cfg.Port,
				time.Duration(cfg.InitialTimeout)*time.Second,
				tcpconn.Wrap)

			trReadTimeoutTicker := time.NewTicker(300 * time.Second)
			clients := make([]transceiver.Client, len(cfg.Proxies))

			for cIdx := range cfg.Proxies {
				clentID := transceiver.ClientID(cIdx)

				clients[cIdx] = tclient.New(clentID, log, tcp.New(
					cfg.Proxies[cIdx].Host,
					cfg.Proxies[cIdx].Port,
					time.Duration(cfg.Proxies[cIdx].RequestTimeout)*time.Second,
					tcpconn.Wrap,
				), cfg.Proxies[cIdx].selectedCodec.Build(
					[]byte(cfg.Proxies[cIdx].CodecSetting),
				), trReadTimeoutTicker.C, tclient.Config{
					MaxConcurrent:  cfg.Proxies[cIdx].Connections,
					RequestRetries: cfg.Proxies[cIdx].RequestRetries,
					InitialTimeout: time.Duration(
						cfg.Proxies[cIdx].RequestTimeout) * time.Second,
					IdleTimeout: time.Duration(
						cfg.Proxies[cIdx].Timeout) * time.Second,
					ConnectionPersistent: cfg.Proxies[cIdx].Persistent,
					ConnectionChannels:   cfg.Proxies[cIdx].Channels,
				})
			}

			var accountVerifer Authenticator

			if len(cfg.Account) > 0 {
				accounts := make(map[string]string, len(cfg.Account))

				for aIdx := range cfg.Account {
					accounts[cfg.Account[aIdx].Username] =
						cfg.Account[aIdx].Password
				}

				accountVerifer = func(username, password string) error {
					aPass, aFound := accounts[username]

					if !aFound {
						return errors.New("Socks5 Account was not found")
					}

					if aPass != password {
						return errors.New("Socks5 Account password mismatch")
					}

					return nil
				}
			}

			return New(trReadTimeoutTicker, clients, listen, log, Config{
				Capacity: cfg.Capacity,
				NegotiationTimeout: time.Duration(
					cfg.InitialTimeout) * time.Second,
				ConnectionTimeout: time.Duration(
					cfg.Timeout) * time.Second,
				MaxDestinationRecords: 8192,
				Authenticator:         accountVerifer,
			}), nil
		},
	}
}
