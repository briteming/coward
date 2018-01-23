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

package codec

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/reinit/coward/roles/common/codec/aescfb"
	"github.com/reinit/coward/roles/common/codec/aesgcm"
	"github.com/reinit/coward/roles/common/codec/key"
	"github.com/reinit/coward/roles/common/codec/marker"
	"github.com/reinit/coward/roles/common/codec/prefixer"
	"github.com/reinit/coward/roles/common/transceiver"
)

// Vars
var (
	aesPrefixerOptions = []string{
		"Key", "Request-Prefix", "Respond-Prefix",
	}
)

// Consts
const (
	aesPrefixerUsageErr = "You must define an option " +
		"before configuring it. Available options: %s.\r\n\r\nExample:" +
		"\r\n\r\nKey: 434F57415244204170706C69636174696F6E\r\nRequest-Prefix:" +
		"436C69656E74\r\nRespond-Prefix: 536572766572\r\n\r\nNotice:\r\n\r\n" +
		"* One single line can only contain one option\r\n" +
		"* The value of \"Request-Prefix\" and " +
		"\"Respond-Prefix\" option must be swapped at the opponent " +
		"side accordingly"
)

// AESCFB128 return a AESCFB128 Transceiver Codec
func AESCFB128() transceiver.Codec {
	return transceiver.Codec{
		Name: "aes-cfb-128-hmac",
		Usage: "Input a string of letters as shared key (passphrase), " +
			"multiple lines will be combined into a single line " +
			"according to order",
		Build:  aesCFB128Builder,
		Verify: aesVerifier,
	}
}

// AESCFB128QoS return a prefixable AESCFB128 Transceiver Codec for better QoS
// compatibility
func AESCFB128QoS() transceiver.Codec {
	return transceiver.Codec{
		Name:   "aes-cfb-128-hmac-q",
		Usage:  "QoS Prefixer configuration is required",
		Build:  aesPrefixCFB128Builder,
		Verify: aesPrefixVerifier,
	}
}

// AESCFB256 return a AESCFB256 Transceiver Codec
func AESCFB256() transceiver.Codec {
	return transceiver.Codec{
		Name: "aes-cfb-256-hmac",
		Usage: "Input a string of letters as shared key (passphrase), " +
			"multiple lines will be combined into a single line " +
			"according to order",
		Build:  aesCFB256Builder,
		Verify: aesVerifier,
	}
}

// AESCFB256QoS return a prefixable AESCFB256 Transceiver Codec for better QoS
// compatibility
func AESCFB256QoS() transceiver.Codec {
	return transceiver.Codec{
		Name:   "aes-cfb-256-hmac-q",
		Usage:  "QoS Prefixer configuration is required",
		Build:  aesPrefixCFB256Builder,
		Verify: aesPrefixVerifier,
	}
}

// AESGCM128 return a AESGCM128 Transceiver Codec
func AESGCM128() transceiver.Codec {
	return transceiver.Codec{
		Name: "aes-gcm-128",
		Usage: "Input a string of letters as shared key (passphrase), " +
			"multiple lines will be combined into a single line " +
			"according to order",
		Build:  aesGCM128Builder,
		Verify: aesVerifier,
	}
}

// AESGCM128QoS return a prefixable AESGCM128 Transceiver Codec for better QoS
// compatibility
func AESGCM128QoS() transceiver.Codec {
	return transceiver.Codec{
		Name:   "aes-gcm-128-q",
		Usage:  "QoS Prefixer configuration is required",
		Build:  aesPrefixGCM128Builder,
		Verify: aesPrefixVerifier,
	}
}

// AESGCM256 return a AESGCM128 Transceiver Codec
func AESGCM256() transceiver.Codec {
	return transceiver.Codec{
		Name: "aes-gcm-256",
		Usage: "Input a string of letters as shared key (passphrase), " +
			"multiple lines will be combined into a single line " +
			"according to order",
		Build:  aesGCM256Builder,
		Verify: aesVerifier,
	}
}

// AESGCM256QoS return a prefixable AESGCM256 Transceiver Codec for better QoS
// compatibility
func AESGCM256QoS() transceiver.Codec {
	return transceiver.Codec{
		Name:   "aes-gcm-256-q",
		Usage:  "QoS Prefixer configuration is required",
		Build:  aesPrefixGCM256Builder,
		Verify: aesPrefixVerifier,
	}
}

func aesSettingBuilder(configuration []string) []byte {
	var connectedLines string

	for cIdx := range configuration {
		connectedLines += configuration[cIdx]
	}

	return []byte(connectedLines)
}

func aesVerifier(configuration []string) error {
	cfgStr := aesSettingBuilder(configuration)

	if len(cfgStr) < 16 {
		return errors.New("Shared Key was too short. " +
			"Make it at least 16 characters long")
	}

	return nil
}

func aesPrefixBuilder(configuration []string) prefixerSetting {
	return prefixerSettingBuilder(configuration, aesPrefixerOptions)
}

func aesPrefixVerifier(configuration []string) error {
	usageErr := fmt.Errorf(
		aesPrefixerUsageErr, strings.Join(aesPrefixerOptions, ", "))

	return prefixerSettingVerifier(
		configuration, func(c prefixerSetting) error {
			if len(configuration) <= 0 {
				return usageErr
			}

			key, keyFound := c["Key"]

			if !keyFound || len(key) < 16 {
				return errors.New("Option \"Key\" must be defined and " +
					"at least 16 characters long.\r\n\r\nNotice you're " +
					"inputting a hex string which uses 2 characters to " +
					"represent 1 byte of data, so you must input at least 32 " +
					"characters to get a string of 16 bytes data")
			}

			return nil
		}, aesPrefixerOptions, usageErr)
}

func aesCFB128Builder(configuration []string) transceiver.CodecBuilder {
	cfgStr := aesSettingBuilder(configuration)
	timedKey := key.Timed([]byte(cfgStr), 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aescfb.AESCFB(conn, timedKey, 16, timedMarkers, timedMarkerLock)
	}
}

func aesPrefixCFB128Builder(configuration []string) transceiver.CodecBuilder {
	cfg := aesPrefixBuilder(configuration)
	timedKey := key.Timed(cfg["Key"], 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aescfb.AESCFB(prefixer.New(
			conn, cfg["Request-Prefix"], cfg["Respond-Prefix"]),
			timedKey, 16, timedMarkers, timedMarkerLock)
	}
}

func aesCFB256Builder(configuration []string) transceiver.CodecBuilder {
	cfgStr := aesSettingBuilder(configuration)
	timedKey := key.Timed([]byte(cfgStr), 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aescfb.AESCFB(conn, timedKey, 32, timedMarkers, timedMarkerLock)
	}
}

func aesPrefixCFB256Builder(configuration []string) transceiver.CodecBuilder {
	cfg := aesPrefixBuilder(configuration)
	timedKey := key.Timed(cfg["Key"], 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aescfb.AESCFB(prefixer.New(
			conn, cfg["Request-Prefix"], cfg["Respond-Prefix"]),
			timedKey, 32, timedMarkers, timedMarkerLock)
	}
}

func aesGCM128Builder(configuration []string) transceiver.CodecBuilder {
	cfgStr := aesSettingBuilder(configuration)
	timedKey := key.Timed([]byte(cfgStr), 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aesgcm.AESGCM(conn, timedKey, 16, timedMarkers, timedMarkerLock)
	}
}

func aesPrefixGCM128Builder(configuration []string) transceiver.CodecBuilder {
	cfg := aesPrefixBuilder(configuration)
	timedKey := key.Timed(cfg["Key"], 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aesgcm.AESGCM(prefixer.New(
			conn, cfg["Request-Prefix"], cfg["Respond-Prefix"]),
			timedKey, 16, timedMarkers, timedMarkerLock)
	}
}

func aesGCM256Builder(configuration []string) transceiver.CodecBuilder {
	cfgStr := aesSettingBuilder(configuration)
	timedKey := key.Timed([]byte(cfgStr), 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aesgcm.AESGCM(conn, timedKey, 32, timedMarkers, timedMarkerLock)
	}
}

func aesPrefixGCM256Builder(configuration []string) transceiver.CodecBuilder {
	cfg := aesPrefixBuilder(configuration)
	timedKey := key.Timed(cfg["Key"], 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aesgcm.AESGCM(prefixer.New(
			conn, cfg["Request-Prefix"], cfg["Respond-Prefix"]),
			timedKey, 32, timedMarkers, timedMarkerLock)
	}
}
