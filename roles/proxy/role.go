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

package proxy

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
	"github.com/reinit/coward/roles/common/network/listener/tcp"
	"github.com/reinit/coward/roles/common/transceiver"
)

// ConfigMapping Configuration of Mapping
type ConfigMapping struct {
	selectProto network.Protocol
	ID          uint8  `json:"id" cfg:"i,-id:Mapping Item ID."`
	Host        string `json:"host" cfg:"h,-host:Host name of the remote destination."`
	Port        uint16 `json:"port" cfg:"p,-port:Port number of the remote destination."`
	Protocol    string `json:"protocol" cfg:"o,-protocol:Protocol type of the remote destination."`
}

// VerifyProtocol Verify Protocol
func (c *ConfigMapping) VerifyProtocol() error {
	protocolErr := c.selectProto.FromString(c.Protocol)

	if protocolErr != nil {
		return protocolErr
	}

	return nil
}

// Verify Verify all configrations
func (c *ConfigMapping) Verify() error {
	if c.Host == "" {
		return fmt.Errorf("Mapping Host must be defined")
	}

	if c.Port <= 0 {
		return fmt.Errorf("Mapping Port must be defined")
	}

	if c.Protocol == "" || c.selectProto == network.UnspecifiedProto {
		return fmt.Errorf("Mapping Protocol must be defined")
	}

	return nil
}

// ConfigInput Config
type ConfigInput struct {
	components           []interface{}
	selectedInterface    net.IP
	selectedCodec        transceiver.Codec
	Interface            string          `json:"interface" cfg:"i,-interface:Select a network interface for server to listen on by specify the IP address of that interface.\r\n\r\nSet this to \"0.0.0.0\" (or \"::\" for IPv6) to make it publicly accessable, or \"127.0.0.1\" to make it local-only."`
	Port                 uint16          `json:"port" cfg:"p,-port:Specify a port for server to listen on.\r\n\r\nNotice that on some operating systems, you may not able listen on a \"High Port\" (Usually, that's a port number which smaller than 1025) without root privilege.\r\n\r\nIt's not recommended to run this server with such privilege. So instead, you should get around of this limitation by listen on a lower port (Port number that greater than 1024)."`
	Timeout              uint16          `json:"timeout" cfg:"t,-timeout:The maximum idle time in second of a client connection.\r\n\r\nIf server consecutively receives no data from a connection during this period of time, then that connection will be considered as inactive and thus be disconnected."`
	InitialTimeout       uint16          `json:"initial_timeout" cfg:"it,-initial-timeout:The maximum wait time in second for clients to finish Initial request (Or first request)\r\n\r\nA well balanced value is required: You need to give clients plenty of time to finish the Initial request (Otherwise they may never be able to connect), and also be able defending against malicious accesses (By time them out) at same time."`
	Capacity             uint32          `json:"capacity" cfg:"c,-capacity:The maximum connections this server will handle.\r\n\r\nIf amount of connections has reached this limitation, new incoming connections will be dropped."`
	Channels             uint8           `json:"channels" cfg:"n,-channels:How many requests can be simultaneously opened on a single established connection.\r\n\r\nSet the value greater than 1 so a single connection will be allowed to transport multiple requests (Multiplexing). This is very useful to increase the utility of a stable connection.\r\n\r\nWhen the connection is not stable enough however, too many Connection Channels can reduce overall stabililty."`
	ChannelDispatchDelay uint16          `json:"channel_dispatch_delay" cfg:"cd,-channel-delay:A delay of time in millisecond in between Connection Channel data dispatch operations.\r\n\r\nThe main propose of this setting is to limit the CPU usage of the Connection Channel data dispatch. However, it can also in part be use to control the server's connection bandwidth (Higher the delay, lower the bandwidth and CPU usage)."`
	Mapping              []ConfigMapping `json:"mapping" cfg:"m,-mapping:Pre-defined local and remote destinations.\r\n\r\nYou can define both local and remote destinations as server will not enforce access limitation here (In opposite of the dynamical Connect request, which will deny all local accesses)."`
	Codec                string          `json:"codec" cfg:"e,-codec:Specify which Codec will be used to encode and decode  data payload to and from a connection."`
	CodecSetting         string          `json:"codec_setting" cfg:"es,-codec-cfg:Configuration of the Codec.\r\n\r\nThe actual configuration format of this setting is depend on the Codec of your choosing."`
}

// GetDescription get descriptions
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

// VerifyInterface Verify Interface
func (c *ConfigInput) VerifyInterface() error {
	selectedIP := net.ParseIP(c.Interface)

	if selectedIP == nil {
		return errors.New("Invalid IP address")
	}

	c.selectedInterface = selectedIP

	return nil
}

// VerifyTimeout Verify Timeout
func (c *ConfigInput) VerifyTimeout() error {
	if c.Timeout < c.InitialTimeout {
		return errors.New(
			"(Idle) Timeout must be greater than the Initial Timeout")
	}

	return nil
}

// VerifyInitialTimeout Verify InitialTimeout
func (c *ConfigInput) VerifyInitialTimeout() error {
	if c.InitialTimeout > c.Timeout {
		return errors.New(
			"Initial Timeout must be smaller than the Idle Timeout")
	}

	return nil
}

// VerifyCapacity Verify Capacity
func (c *ConfigInput) VerifyCapacity() error {
	if c.Capacity < 1 {
		return errors.New("Capicty must be greater than 0")
	}

	return nil
}

// VerifyChannels Verify Channels
func (c *ConfigInput) VerifyChannels() error {
	if c.Channels < 1 {
		return errors.New("Channels must be greater than 0")
	}

	return nil
}

// VerifyChannelDispatchDelay Verify ChannelDispatchDelay
func (c *ConfigInput) VerifyChannelDispatchDelay() error {
	if c.ChannelDispatchDelay < 0 {
		return errors.New("Channel Dispatch Delay not smaller than 0")
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

// Verify Verify all settings
func (c *ConfigInput) Verify() error {
	if c.Interface == "" {
		c.selectedInterface = net.ParseIP("127.0.0.1")
	}

	if c.Timeout <= 0 {
		return errors.New("Idle Timeout must be specified")
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

	if c.Channels <= 0 {
		return errors.New("Channels must be specified")
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
		Name: "proxy",
		Description: "A Proxy backend server which will handle and relay " +
			"incoming requests from a COWARD Proxy Client",
		Configurator: func(components role.Components) interface{} {
			return &ConfigInput{
				components:           components,
				Interface:            "",
				Port:                 0,
				Timeout:              0,
				InitialTimeout:       0,
				Capacity:             0,
				Channels:             0,
				ChannelDispatchDelay: 20,
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

			listen := tcp.New(
				cfg.selectedInterface,
				cfg.Port,
				time.Duration(cfg.InitialTimeout)*time.Second,
				tcpconn.Wrap)

			mapps := make([]Mapped, len(cfg.Mapping))

			for mIdx := range cfg.Mapping {
				mapps[mIdx] = Mapped{
					ID:       cfg.Mapping[mIdx].ID,
					Host:     cfg.Mapping[mIdx].Host,
					Port:     cfg.Mapping[mIdx].Port,
					Protocol: cfg.Mapping[mIdx].selectProto,
				}
			}

			return New(
				cfg.selectedCodec.Build([]byte(cfg.CodecSetting)),
				listen,
				log,
				Config{
					Capacity: cfg.Capacity,
					InitialTimeout: time.Duration(
						cfg.InitialTimeout) * time.Second,
					IdleTimeout: time.Duration(
						cfg.Timeout) * time.Second,
					ConnectionChannels: cfg.Channels,
					ChannelDispatchDelay: time.Duration(
						cfg.ChannelDispatchDelay) * time.Millisecond,
					Mapping: mapps,
				}), nil
		},
	}
}
