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

package projector

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
	"github.com/reinit/coward/roles/projector/projection"
)

// ConfigProject Configuration of Project
type ConfigProject struct {
	selectedProto     network.Protocol
	selectedInterface net.IP
	ID                uint8  `json:"id" cfg:"i,-id:Projection ID."`
	Interface         string `json:"interface" cfg:"a,-interface:Specify the interface which current Projection server will listening to.\r\n\r\nSet this to \"0.0.0.0\" (or \"::\" for IPv6) to make it publicly accessable, or \"127.0.0.1\" to make it local-only."`
	Port              uint16 `json:"port" cfg:"p,-port:Specify a port for server to listen on.\r\n\r\nNotice you may need special permission in order to listen on a port that higher (smaller in number) than 1024."`
	Protocol          string `json:"protocol" cfg:"o,-protocol:Specify which network protocol this server using."`
	Capacity          uint32 `json:"capacity" cfg:"c,-capacity:The maximum connections this server will handle.\r\n\r\nIf amount of connections has reached this limitation, new incoming connections will be dropped."`
	Retries           uint8  `json:"retries" cfg:"r,-retries:When a request to current Projection has failed, how many times we will going to retry that request before given up."`
}

// VerifyInterface Verify Interface
func (c *ConfigProject) VerifyInterface() error {
	selectedIP := net.ParseIP(c.Interface)

	if selectedIP == nil {
		return errors.New("Invalid IP address")
	}

	c.selectedInterface = selectedIP

	return nil
}

// VerifyProtocol Verify Protocol
func (c *ConfigProject) VerifyProtocol() error {
	protocolErr := c.selectedProto.FromString(c.Protocol)

	if protocolErr != nil {
		return protocolErr
	}

	return nil
}

// VerifyCapacity Verify Capacity
func (c *ConfigProject) VerifyCapacity() error {
	if c.Capacity < 1 {
		return errors.New("Capacity must be greater than 0")
	}

	if c.Capacity > 8000000 {
		return errors.New("Capacity must be smaller than 8,000,000")
	}

	return nil
}

// VerifyRetries Verify Retries
func (c *ConfigProject) VerifyRetries() error {
	if c.Retries <= 0 {
		return errors.New("Retries must be greater than 0")
	}

	return nil
}

// Verify Verifies ConfigProject
func (c *ConfigProject) Verify() error {
	if c.Interface == "" {
		return fmt.Errorf("Project server Interface must be defined")
	}

	if c.Protocol == "" || c.selectedProto == network.UnspecifiedProto {
		return fmt.Errorf("Project server Protocol must be defined")
	}

	if c.Capacity <= 0 {
		return fmt.Errorf("Project server Capacity must be defined")
	}

	if c.Retries <= 0 {
		c.Retries = 1
	}

	return nil
}

// ConfigInput Config
type ConfigInput struct {
	components           []interface{}
	selectedInterface    net.IP
	selectedCodec        transceiver.Codec
	Interface            string           `json:"interface" cfg:"i,-interface:Select a network interface for the Projector Register server to listen on by specify the IP address of that interface.\r\n\r\nSet this to \"0.0.0.0\" (or \"::\" for IPv6) to make it publicly accessable, or \"127.0.0.1\" to make it local-only."`
	Port                 uint16           `json:"port" cfg:"p,-port:Specify a port for the Projector Register server to listen on.\r\n\r\nNotice that on some operating systems, you may not able listen on a \"High Port\" (Usually, that's a port number which smaller than 1025) without root privilege.\r\n\r\nIt's not recommended to run this server with such privilege. So instead, you should get around of this limitation by listen on a lower port (Port number that greater than 1024)."`
	Timeout              uint16           `json:"timeout" cfg:"t,-timeout:The maximum idle time in second of a COWARD Project client connection.\r\n\r\nIf server consecutively receives no data from a connection during this period of time, then that connection will be considered as inactive and thus be disconnected."`
	InitialTimeout       uint16           `json:"initial_timeout" cfg:"it,-initial-timeout:The maximum wait time in second for COWARD Project client to finish Initial request (Or first request)\r\n\r\nA well balanced value is required: You need to give clients plenty of time to finish the Initial request (Otherwise they may never be able to connect), and also be able defending against malicious accesses (By time them out) at same time."`
	Capacity             uint32           `json:"capacity" cfg:"c,-capacity:The maximum connections the Projector register server will handle.\r\n\r\nIf amount of connections has reached this limitation, new incoming connections will be dropped."`
	Channels             uint8            `json:"channels" cfg:"n,-channels:How many requests can be simultaneously opened on a single established connection.\r\n\r\nSet the value greater than 1 so a single connection will be allowed to transport multiple requests (Multiplexing). This is very useful to increase the utility of a stable connection.\r\n\r\nWhen the connection is not stable enough however, too many Connection Channels can reduce overall stabililty."`
	ChannelDispatchDelay uint16           `json:"channel_dispatch_delay" cfg:"cd,-channel-delay:A delay of time in millisecond in between Connection Channel data dispatch operations.\r\n\r\nThe main propose of this setting is to limit the CPU usage of the Connection Channel data dispatch. However, it can also in part be use to control the server's connection bandwidth (Higher the delay, lower the bandwidth and CPU usage)."`
	Projects             []*ConfigProject `json:"projects" cfg:"s,-projects:Pre-defined Projection servers"`
	Codec                string           `json:"codec" cfg:"e,-codec:Specify which Codec will be used to encode and decode data payload to and from a connection."`
	CodecSetting         []string         `json:"codec_setting" cfg:"es,-codec-cfg:Configuration of the Codec as an array of string.\r\n\r\nThe actual configuration format of this setting is depend on the Codec of your choosing."`
}

// GetDescription get descriptions
func (c ConfigInput) GetDescription(fieldPath string) string {
	result := ""

	switch fieldPath {
	case "/Interface":
		fallthrough
	case "/Projects/Interface":
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

	case "/Projects/Protocol":
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
		return errors.New("Capacity must be greater than 0")
	}

	if c.Capacity > 8000000 {
		return errors.New("Capacity must be smaller than 8,000,000")
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

	return c.selectedCodec.Verify(c.CodecSetting)
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
		return errors.New("Capacity must be specified")
	}

	if c.Channels <= 0 {
		return errors.New("Channels must be specified")
	}

	if len(c.Projects) <= 0 {
		return errors.New("Must specify at least one Project")
	}

	if c.Codec == "" {
		return errors.New("Codec must be specified")
	}

	if c.selectedCodec.Verify != nil {
		vErr := c.selectedCodec.Verify(c.CodecSetting)

		if vErr != nil {
			return errors.New("Codec Setting was invalid: " + vErr.Error())
		}
	}

	return nil
}

// Role register
func Role() role.Registration {
	return role.Registration{
		Name: "projector",
		Description: "A reverse proxy that dispatch and relay requests to " +
			"an active COWARD Project client",
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
				Projects:             []*ConfigProject{},
				Codec:                "",
				CodecSetting:         nil,
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
				tcpconn.Wrap)

			projects := make([]Server, len(cfg.Projects))

			for mIdx := range cfg.Projects {
				projects[mIdx] = Server{
					ID:        projection.ID(cfg.Projects[mIdx].ID),
					Interface: cfg.Projects[mIdx].selectedInterface,
					Port:      cfg.Projects[mIdx].Port,
					Protocol:  cfg.Projects[mIdx].selectedProto,
					Capacity:  cfg.Projects[mIdx].Capacity,
					Retries:   cfg.Projects[mIdx].Retries,
				}
			}

			return New(
				listen,
				cfg.selectedCodec.Build(cfg.CodecSetting),
				log,
				Config{
					Servers:  projects,
					Capacity: cfg.Capacity,
					InitialTimeout: time.Duration(
						cfg.InitialTimeout) * time.Second,
					IdleTimeout: time.Duration(
						cfg.Timeout) * time.Second,
					ConnectionChannels: cfg.Channels,
					ChannelDispatchDelay: time.Duration(
						cfg.ChannelDispatchDelay) * time.Millisecond,
				}), nil
		},
	}
}
