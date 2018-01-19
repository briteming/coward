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

package project

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/print"
	"github.com/reinit/coward/common/role"
	"github.com/reinit/coward/roles/common/network"
	tcpconn "github.com/reinit/coward/roles/common/network/connection/tcp"
	"github.com/reinit/coward/roles/common/network/dialer/tcp"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/project/project"
	"github.com/reinit/coward/roles/projector/projection"
)

// ConfigProject Project config
type ConfigProject struct {
	selectedProto  network.Protocol
	ID             projection.ID `json:"id" cfg:"i,-id:Projection ID, must exist on Projector server as well."`
	Host           string        `json:"host" cfg:"h,-host:Host name of the Projection destination."`
	Port           uint16        `json:"port" cfg:"p,-port:Port number of the Projection destination."`
	Protocol       string        `json:"protocol" cfg:"o,-protocol:Protocol type of the Projection destination. Should be the same on the Projector server."`
	MaxConnections uint32        `json:"connections" cfg:"c,-connections:How many connections to the Projection destination can be established at same time.\r\n\r\nNotice this value also effects how many connections will be established to the COWARD Projector server.\r\n\r\nSay if you allowing 1000 Projection connections and having 100 channels per Projector server connection, then we will create 10 connections to the Projector server."`
	Retry          uint8         `json:"retries" cfg:"r,-retries:How many times we will retry when we have failed to connect to the Projection destination."`
	Timeout        uint16        `json:"timeout" cfg:"t,-timeout:The maximum wait time in second for a idle connection to the Projection destination can be maintianed.\r\n\r\nIf the connection remain idle pass this period of time, then it will be closed."`
	RequestTimeout uint16        `json:"request_timeout" cfg:"rt,-request-timeout:The maximum wait time in second for the Projection destination to accept our connect request."`
}

// VerifyHost Verify Host
func (c *ConfigProject) VerifyHost() error {
	if c.Host == "" {
		return errors.New("Host must be defined")
	}

	return nil
}

// VerifyPort Verify Port
func (c *ConfigProject) VerifyPort() error {
	if c.Port <= 0 {
		return errors.New("Port must be defined")
	}

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

// VerifyMaxConnections Verify MaxConnections
func (c *ConfigProject) VerifyMaxConnections() error {
	if c.MaxConnections < 1 {
		return errors.New("Connections must be greater than 0")
	}

	return nil
}

// VerifyRetry Verify Retry
func (c *ConfigProject) VerifyRetry() error {
	if c.Retry < 1 {
		return errors.New("Retries must be greater than 0")
	}

	return nil
}

// VerifyTimeout Verify Timeout
func (c *ConfigProject) VerifyTimeout() error {
	if c.Timeout < 1 {
		return errors.New("Timeout must be greater than 0")
	}

	return nil
}

// VerifyRequestTimeout Verify RequestTimeout
func (c *ConfigProject) VerifyRequestTimeout() error {
	if c.RequestTimeout < 1 {
		return errors.New("Request Timeout must be greater than 0")
	}

	return nil
}

// Verify Verifies
func (c *ConfigProject) Verify() error {
	if c.Protocol == "" || c.selectedProto == network.UnspecifiedProto {
		return fmt.Errorf("Protocol must be defined")
	}

	if c.Host == "" {
		return errors.New("Host must be specified")
	}

	if c.Port < 0 {
		return errors.New("Port must be specified")
	}

	if c.MaxConnections <= 0 {
		return errors.New("Connections must be specified")
	}

	if c.Retry <= 0 {
		return errors.New("Retries must be specified")
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

	return nil
}

// ConfigInput configurations
type ConfigInput struct {
	components     []interface{}
	selectedCodec  transceiver.Codec
	Host           string           `json:"host" cfg:"h,-host:Host name of the remote COWARD Projector server.\r\n\r\nMust matchs the setting on server."`
	Port           uint16           `json:"port" cfg:"p,-port:Registeration port of the remote COWARD Projector server.\r\n\r\nMust matchs the setting on server."`
	Timeout        uint16           `json:"timeout" cfg:"t,-timeout:The maximum idle time in second of the established connection.\r\n\r\nIf a connection is consecutively idle during this period of time, then that connection will be considered as inactive and thus be disconnected.\r\n\r\nIt is recommended to set this value no greater than the related one on the COWARD Projector server setting."`
	RequestTimeout uint16           `json:"request_timeout" cfg:"rt,-request-timeout:The maximum wait time in second for the server to respond the Initial request of a client.\r\n\r\nIf the COWARD Projector server has failed to respond the Initial request within this period of time, the connection will be considered broken and thus be closed.\r\n\r\nIt is recommended to set this value slightly greater than the \"--initial-timeout\" setting on the COWARD Projector server."`
	Channels       uint8            `json:"channels" cfg:"n,-channels:How many requests can be simultaneously opened on a single established connection.\r\n\r\nSet the value greater than 1 so a single connection can be use to transport multiple requests (Multiplexing).\r\n\r\nWARNING:\r\nThis value must matchs or smaller than the related setting on the COWARD Projector server, otherwise the request will be come malformed and thus dropped."`
	Persistent     bool             `json:"persist" cfg:"k,-persist:Whether or not to keep the connection to the COWARD Projector active after all requests on the connection is completed."`
	Projects       []*ConfigProject `json:"projects" cfg:"s,-projects:Pre-defined project destnations.\r\n\r\nMust be exist on the COWARD Projector server."`
	Codec          string           `json:"codec" cfg:"e,-codec:Specify which Codec will be used to encode and decode data payload to and from a connection."`
	CodecSetting   string           `json:"codec_setting" cfg:"es,-codec-cfg:Configuration of the Codec.\r\n\r\nThe actual configuration format of this setting is depend on the Codec of your choosing."`
}

// GetDescription get descriptions
func (c ConfigInput) GetDescription(fieldPath string) string {
	result := ""

	switch fieldPath {
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

// VerifyPort Verify Port
func (c *ConfigInput) VerifyPort() error {
	if c.Port < 1 {
		return errors.New("Port must be greater than 0")
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

// VerifyChannels Verify Channels
func (c *ConfigInput) VerifyChannels() error {
	if c.Channels < 1 {
		return errors.New("Channels must be greater than 0")
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
		return errors.New("Host must be defined")
	}

	if c.Port <= 0 {
		return errors.New("Port must be defined")
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

	if len(c.Projects) <= 0 {
		return errors.New("Must define at least one Project")
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

// Role register
func Role() role.Registration {
	return role.Registration{
		Name: "project",
		Description: "Projects endpoints as an open service that can " +
			"be accessed through the COWARD Projector",
		Configurator: func(components role.Components) interface{} {
			return &ConfigInput{
				components:     components,
				selectedCodec:  transceiver.Codec{},
				Host:           "",
				Port:           0,
				Timeout:        0,
				RequestTimeout: 0,
				Channels:       0,
				Persistent:     false,
				Projects:       []*ConfigProject{},
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

			endpoints := make(Endpoints, len(cfg.Projects))

			for mIdx := range cfg.Projects {
				endpoints[mIdx] = project.Endpoint{
					ID:             projection.ID(cfg.Projects[mIdx].ID),
					Host:           cfg.Projects[mIdx].Host,
					Port:           cfg.Projects[mIdx].Port,
					Protocol:       cfg.Projects[mIdx].selectedProto,
					MaxConnections: cfg.Projects[mIdx].MaxConnections,
					Retry:          cfg.Projects[mIdx].Retry,
					Timeout: time.Duration(
						cfg.Projects[mIdx].Timeout) * time.Second,
					RequestTimeout: time.Duration(
						cfg.Projects[mIdx].RequestTimeout) * time.Second,
				}
			}

			return New(
				cfg.selectedCodec.Build([]byte(cfg.CodecSetting)),
				dialer,
				log,
				Config{
					TransceiverIdleTimeout: time.Duration(
						cfg.Timeout) * time.Second,
					TransceiverInitialTimeout: time.Duration(
						cfg.RequestTimeout) * time.Second,
					TransceiverChannels:             cfg.Channels,
					TransceiverConnectionPersistent: cfg.Persistent,
					Endpoints:                       endpoints,
				}), nil
		},
	}
}
