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
	selectProto           network.Protocol
	selectedListenAddress net.IP
	ID                    uint8  `json:"id" cfg:"i,-id:Mapping Item ID.\r\n\r\nMust matchs the setting defined on the COWARD Proxy."`
	Protocol              string `json:"protocol" cfg:"o,-protocol:Protocol type of the remote destination.\r\n\r\nMust matchs the setting defined on the COWARD Proxy."`
	ListenAddress         string `json:"listen_address" cfg:"a,-interface:Specify a local network interface to serve for the mapped destination."`
	ListenPort            uint16 `json:"listen_port" cfg:"p,-port:Specify a local port to serve for the mapped destination."`
	MaxConnections        uint32 `json:"capicty" cfg:"c,-capacity:The maximum connections this Mapping server can accept.\r\n\r\nWhen amount of connections reached this limitation, new incoming connection will be dropped."`
}

// VerifyProtocol Verify Protocol
func (c *ConfigMapping) VerifyProtocol() error {
	protocolErr := c.selectProto.FromString(c.Protocol)

	if protocolErr != nil {
		return protocolErr
	}

	return nil
}

// VerifyListenAddress Verify ListenAddress
func (c *ConfigMapping) VerifyListenAddress() error {
	ipAddr := net.ParseIP(c.ListenAddress)

	if ipAddr == nil {
		return errors.New("Invalid IP address")
	}

	c.selectedListenAddress = ipAddr

	return nil
}

// VerifyMaxConnections Verify MaxConnections
func (c *ConfigMapping) VerifyMaxConnections() error {
	if c.MaxConnections < 1 {
		return errors.New("Capicty must be greater than 0")
	}

	return nil
}

// Verify Verifies
func (c *ConfigMapping) Verify() error {
	if c.Protocol == "" || c.selectProto == network.UnspecifiedProto {
		return fmt.Errorf("Mapping Protocol must be defined")
	}

	if c.ListenAddress == "" {
		return errors.New("Listen Address must be specified")
	}

	if c.ListenPort < 0 {
		return errors.New("Listen Port must be specified")
	}

	if c.MaxConnections <= 0 {
		return errors.New("Capicty must be specified")
	}

	return nil
}

// ConfigInput Configuration
type ConfigInput struct {
	components           []interface{}
	selectedCodec        transceiver.Codec
	ProxyHost            string          `json:"proxy_host" cfg:"ph,-proxy-host:Host name of the remote COWARD Proxy server.\r\n\r\nMust matchs the configuration on the Proxy server."`
	ProxyPort            uint16          `json:"proxy_port" cfg:"pp,-proxy-port:Port number of the remote COWARD Proxy server.\r\n\r\nMust matchs the configuration on the Proxy server."`
	MaxConnections       uint32          `json:"max_connections" cfg:"mc,-connections:The maximum concurrent connections that can be maintained by a single COWARD Proxy Client at same time.\r\n\r\nWhen this limitation has been reached, new requests will has to wait until existing requests is completed before continue requesting."`
	ConnectionPersistent bool            `json:"connection_persist" cfg:"pc,-persist:Whether or not to keep the connection to the COWARD Proxy active after all requests on the connection is completed."`
	RequestRetries       uint8           `json:"request_retries" cfg:"rr,-request-retries:How many times a failed Initial request can be retried."`
	ConnectionChannels   uint8           `json:"channels" cfg:"pm,-channels:How many requests can be simultaneously opened on a single established connection.\r\n\r\nSet the value greater than 1 so a single connection can be use to transport multiple requests (Multiplexing).\r\n\r\nWARNING:\r\nThis value must matchs or smaller than the related setting on the COWARD Proxy server, otherwise the request will be come malformed and thus dropped."`
	IdleTimeout          uint16          `json:"idle_timeout" cfg:"t,-idle-timeout:The maximum idle time in second of a client connection.\r\n\r\nIf a connection is consecutively idle during this period of time, then that connection will be considered as inactive and thus be disconnected.\r\n\r\nIt is recommended to set this value no greater than the related one on the COWARD Proxy server setting."`
	RequestTimeout       uint16          `json:"request_timeout" cfg:"rt,-request-timeout:The maximum wait time in second for client request to be initialized.\r\n\r\nIf the COWARD Proxy server has failed to respond the Initial request within this period of time, the connection will be considered broken and thus be closed.\r\n\r\nIt is recommended to set this value slightly greater than the \"--initial-timeout\" setting on the COWARD Proxy server."`
	Mapping              []ConfigMapping `json:"mapping" cfg:"m,-mapping:Enable and configure mapped remote destinations.\r\n\r\nThis will allow you to map the pre-defined destinations on the Proxy as local servers.\r\n\r\nAll access to these servers will be relayed to their corresponding remote destinations transparently through the COWARD Proxy server."`
	Codec                string          `json:"codec" cfg:"co,-codec:Specifiy which Codec will be used to decode and encode data payload from and to a connection."`
	CodecSetting         string          `json:"codec_setting" cfg:"cs,-codec-cfg:Configuration of the Codec.\r\n\r\nThe actual configuration format of this setting is depend on the Codec of your choosing."`
}

// GetDescription gets description
func (c ConfigInput) GetDescription(fieldPath string) string {
	result := ""

	switch fieldPath {
	case "/Mapping/ListenAddress":
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

// VerifyProxyPort Verify ProxyPort
func (c *ConfigInput) VerifyProxyPort() error {
	if c.ProxyPort <= 0 {
		return errors.New("Proxy Port must be greater than 0")
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

// VerifyConnectionChannels Verify ConnectionChannels
func (c *ConfigInput) VerifyConnectionChannels() error {
	if c.ConnectionChannels <= 0 {
		return errors.New("Channels must be greater than 0")
	}

	return nil
}

// VerifyIdleTimeout Verify IdleTimeout
func (c *ConfigInput) VerifyIdleTimeout() error {
	if c.IdleTimeout < c.RequestTimeout {
		return errors.New(
			"Idle Timeout must be greater than the Request Timeout")
	}

	return nil
}

// VerifyRequestTimeout Verify RequestTimeout
func (c *ConfigInput) VerifyRequestTimeout() error {
	if c.RequestTimeout > c.IdleTimeout {
		return errors.New(
			"Request Timeout must be smaller than the Idle Timeout")
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
	if c.ProxyHost == "" {
		return errors.New("Proxy Host must be specified")
	}

	if c.ProxyPort <= 0 {
		return errors.New("Proxy Port must be specified")
	}

	if c.MaxConnections <= 0 {
		return errors.New("Connections must be specified")
	}

	if c.RequestRetries <= 0 {
		c.RequestRetries = 3
	}

	if c.ConnectionChannels <= 0 {
		return errors.New("Channels must be specified")
	}

	if c.IdleTimeout <= 0 {
		return errors.New("Idle Timeout must be specified")
	}

	if c.RequestTimeout <= 0 {
		if c.IdleTimeout <= 10 {
			c.RequestTimeout = 1
		} else {
			c.RequestTimeout = c.IdleTimeout / 10
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
				components:           components,
				ProxyHost:            "",
				ProxyPort:            0,
				MaxConnections:       0,
				ConnectionPersistent: false,
				RequestRetries:       0,
				ConnectionChannels:   0,
				IdleTimeout:          0,
				RequestTimeout:       0,
				Mapping:              []ConfigMapping{},
				Codec:                "",
				CodecSetting:         "",
			}
		},
		Generater: func(
			w print.Common,
			config interface{},
			log logger.Logger,
		) (role.Role, error) {
			cfg := config.(*ConfigInput)

			dialer := tcp.New(
				cfg.ProxyHost,
				cfg.ProxyPort,
				time.Duration(cfg.RequestTimeout)*time.Second,
				tcpconn.Wrap,
			)

			mapps := make([]Mapped, len(cfg.Mapping))

			for mIdx := range cfg.Mapping {
				mapps[mIdx] = Mapped{
					ID:              common.MapID(cfg.Mapping[mIdx].ID),
					ListenInterface: cfg.Mapping[mIdx].selectedListenAddress,
					ListenPort:      cfg.Mapping[mIdx].ListenPort,
					Protocol:        cfg.Mapping[mIdx].selectProto,
					MaxConnections:  cfg.Mapping[mIdx].MaxConnections,
				}
			}

			return New(
				cfg.selectedCodec.Build([]byte(cfg.CodecSetting)),
				dialer,
				log,
				Config{
					TransceiverMaxConnections: cfg.MaxConnections,
					TransceiverRequestRetries: cfg.RequestRetries,
					TransceiverIdleTimeout: time.Duration(
						cfg.IdleTimeout) * time.Second,
					TransceiverInitialTimeout: time.Duration(
						cfg.RequestTimeout) * time.Second,
					TransceiverConnectionPersistent: cfg.ConnectionPersistent,
					TransceiverChannels:             cfg.ConnectionChannels,
					Mapping:                         mapps,
				}), nil
		},
	}
}
