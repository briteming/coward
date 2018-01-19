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

package mapper

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/print"
	"github.com/reinit/coward/common/role"
	"github.com/reinit/coward/roles/common/network"
	tcpconn "github.com/reinit/coward/roles/common/network/connection/tcp"
	"github.com/reinit/coward/roles/common/network/dialer/tcp"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/proxy/common"
)

// ConfigMapping Mapping Configuration
type ConfigMapping struct {
	selectProto       network.Protocol
	selectedInterface net.IP
	ID                uint8  `json:"id" cfg:"i,-id:Mapping Item ID.\r\n\r\nMust matchs the setting defined on the COWARD Proxy."`
	Protocol          string `json:"protocol" cfg:"o,-protocol:Protocol type of the remote destination.\r\n\r\nMust matchs the setting defined on the COWARD Proxy."`
	Interface         string `json:"interface" cfg:"a,-interface:Specify a local network interface to serve for the mapped destination."`
	Port              uint16 `json:"port" cfg:"p,-port:Specify a local port to serve for the mapped destination."`
	Capacity          uint32 `json:"capacity" cfg:"c,-capacity:The maximum connections this Mapping server can accept.\r\n\r\nWhen amount of connections reached this limitation, new incoming connection will be dropped."`
}

// VerifyProtocol Verify Protocol
func (c *ConfigMapping) VerifyProtocol() error {
	protocolErr := c.selectProto.FromString(c.Protocol)

	if protocolErr != nil {
		return protocolErr
	}

	return nil
}

// VerifyInterface Verify Interface
func (c *ConfigMapping) VerifyInterface() error {
	ipAddr := net.ParseIP(c.Interface)

	if ipAddr == nil {
		return errors.New("Invalid IP address")
	}

	c.selectedInterface = ipAddr

	return nil
}

// VerifyCapacity Verify Capacity
func (c *ConfigMapping) VerifyCapacity() error {
	if c.Capacity < 1 {
		return errors.New("Capacity must be greater than 0")
	}

	return nil
}

// Verify Verifies
func (c *ConfigMapping) Verify() error {
	if c.Protocol == "" || c.selectProto == network.UnspecifiedProto {
		return fmt.Errorf("Protocol must be defined")
	}

	if c.Interface == "" {
		return errors.New("Interface must be specified")
	}

	if c.Port < 0 {
		return errors.New("Port must be specified")
	}

	if c.Capacity <= 0 {
		return errors.New("Capacity must be specified")
	}

	return nil
}

// ConfigInput Configuration
type ConfigInput struct {
	components     []interface{}
	selectedCodec  transceiver.Codec
	Host           string          `json:"host" cfg:"h,-host:Host name of the remote COWARD Proxy server.\r\n\r\nMust matchs the configuration on the Proxy server."`
	Port           uint16          `json:"port" cfg:"p,-port:Port number of the remote COWARD Proxy server.\r\n\r\nMust matchs the configuration on the Proxy server."`
	Connections    uint32          `json:"connections" cfg:"c,-connections:The maximum concurrent connections that can be established with a COWARD Proxy Server."`
	Persistent     bool            `json:"persist" cfg:"k,-persist:Whether or not to keep the connection to the COWARD Proxy active even after all requests on the connection is completed."`
	RequestRetries uint8           `json:"retries" cfg:"r,-retries:How many times a failed Initial request can be retried."`
	Channels       uint8           `json:"channels" cfg:"n,-channels:How many requests can be simultaneously opened on a single established connection.\r\n\r\nSet the value greater than 1 so a single connection can be use to transport multiple requests (Multiplexing).\r\n\r\nWARNING:\r\nThis value must matchs or smaller than the related setting on the COWARD Proxy server, otherwise the request will be come malformed and thus dropped."`
	Timeout        uint16          `json:"timeout" cfg:"t,-timeout:The maximum idle time in second of a established proxy connection.\r\n\r\nIf the proxy connection consecutively idle during this period of time, then that connection will be considered as inactive and thus be disconnected.\r\n\r\nIt is recommended to set this value no greater than the related one on the COWARD Proxy server setting."`
	RequestTimeout uint16          `json:"request_timeout" cfg:"rt,-request-timeout:The maximum wait time in second for the server to respond the Initial request of a client.\r\n\r\nIf the COWARD Proxy server has failed to respond the Initial request within this period of time, the connection will be considered broken and thus be closed.\r\n\r\nIt is recommended to set this value slightly greater than the \"--initial-timeout\" setting on the COWARD Proxy server."`
	Mapping        []ConfigMapping `json:"mapping" cfg:"m,-mapping:Enable and configure mapped remote destinations.\r\n\r\nThis will allow you to map the pre-defined destinations on the Proxy as local servers.\r\n\r\nAll access to these servers will be relayed to their corresponding remote destinations transparently through the COWARD Proxy server."`
	Codec          string          `json:"codec" cfg:"e,-codec:Specify which Codec will be used to encode and decode data payload to and from a connection."`
	CodecSetting   string          `json:"codec_setting" cfg:"es,-codec-cfg:Configuration of the Codec.\r\n\r\nThe actual configuration format of this setting is depend on the Codec of your choosing."`
}

// GetDescription gets description
func (c ConfigInput) GetDescription(fieldPath string) string {
	result := ""

	switch fieldPath {
	case "/Mapping/Interface":
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

	case "/Mapping/Protocol":
		result = "Available protocols:\r\n- " +
			strings.Join([]string{"tcp", "udp"}, "\r\n- ")

	case "/Codec":
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

// VerifyPort Verify Port
func (c *ConfigInput) VerifyPort() error {
	if c.Port <= 0 {
		return errors.New("Port must be greater than 0")
	}

	return nil
}

// VerifyRequestRetries Verify RequestRetries
func (c *ConfigInput) VerifyRequestRetries() error {
	if c.RequestRetries <= 0 {
		return errors.New("Request Retries must be greater than 0")
	}

	return nil
}

// VerifyChannels Verify Channels
func (c *ConfigInput) VerifyChannels() error {
	if c.Channels <= 0 {
		return errors.New("Channels must be greater than 0")
	}

	return nil
}

// VerifyTimeout Verify Timeout
func (c *ConfigInput) VerifyTimeout() error {
	if c.Timeout < c.RequestTimeout {
		return errors.New(
			"(Idle) Timeout must be greater than the Request Timeout")
	}

	return nil
}

// VerifyRequestTimeout Verify RequestTimeout
func (c *ConfigInput) VerifyRequestTimeout() error {
	if c.RequestTimeout > c.Timeout {
		return errors.New(
			"Request Timeout must be smaller than the (Idle) Timeout")
	}

	return nil
}

// VerifyCodec Verify Codec
func (c *ConfigInput) VerifyCodec() error {
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
func (c *ConfigInput) VerifyCodecSetting() error {
	if c.selectedCodec.Verify == nil {
		return errors.New("Codec must be specified")
	}

	return c.selectedCodec.Verify([]byte(c.CodecSetting))
}

// Verify Verifies
func (c *ConfigInput) Verify() error {
	if c.Host == "" {
		return errors.New("Host must be specified")
	}

	if c.Port <= 0 {
		return errors.New("Port must be specified")
	}

	if c.Connections <= 0 {
		return errors.New("Connections must be specified")
	}

	if c.RequestRetries <= 0 {
		c.RequestRetries = 3
	}

	if c.Channels <= 0 {
		return errors.New("Channels must be specified")
	}

	if c.Timeout <= 0 {
		return errors.New("(Idle) Timeout must be specified")
	}

	if c.RequestTimeout <= 0 {
		if c.Timeout <= 10 {
			c.RequestTimeout = 1
		} else {
			c.RequestTimeout = c.Timeout / 10
		}
	}

	if len(c.Mapping) <= 0 {
		return errors.New("At least one mapping item is required")
	}

	if c.Codec == "" {
		return errors.New("Codec must be specified")
	}

	if c.selectedCodec.Verify != nil {
		vErr := c.selectedCodec.Verify([]byte(c.CodecSetting))

		if vErr != nil {
			return errors.New("Codec Setting was invalid: " + vErr.Error())
		}
	}

	return nil
}

// Role register
func Role() role.Registration {
	return role.Registration{
		Name: "mapper",
		Description: "Map pre-defined remote destinations on the COWARD " +
			"Proxy as server",
		Configurator: func(components role.Components) interface{} {
			return &ConfigInput{
				components:     components,
				Host:           "",
				Port:           0,
				Connections:    0,
				Persistent:     false,
				RequestRetries: 0,
				Channels:       0,
				Timeout:        0,
				RequestTimeout: 0,
				Mapping:        []ConfigMapping{},
				Codec:          "",
				CodecSetting:   "",
			}
		},
		Generater: func(
			w print.Common,
			config interface{},
			log logger.Logger,
		) (role.Role, error) {
			cfg := config.(*ConfigInput)

			dialer := tcp.New(
				cfg.Host,
				cfg.Port,
				time.Duration(cfg.RequestTimeout)*time.Second,
				tcpconn.Wrap,
			)

			mapps := make([]Mapped, len(cfg.Mapping))

			for mIdx := range cfg.Mapping {
				mapps[mIdx] = Mapped{
					ID:        common.MapID(cfg.Mapping[mIdx].ID),
					Interface: cfg.Mapping[mIdx].selectedInterface,
					Port:      cfg.Mapping[mIdx].Port,
					Protocol:  cfg.Mapping[mIdx].selectProto,
					Capacity:  cfg.Mapping[mIdx].Capacity,
				}
			}

			return New(
				cfg.selectedCodec.Build([]byte(cfg.CodecSetting)),
				dialer,
				log,
				Config{
					TransceiverMaxConnections: cfg.Connections,
					TransceiverRequestRetries: cfg.RequestRetries,
					TransceiverIdleTimeout: time.Duration(
						cfg.Timeout) * time.Second,
					TransceiverInitialTimeout: time.Duration(
						cfg.RequestTimeout) * time.Second,
					TransceiverConnectionPersistent: cfg.Persistent,
					TransceiverChannels:             cfg.Channels,
					Mapping:                         mapps,
				}), nil
		},
	}
}
