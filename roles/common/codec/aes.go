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
	"sync"
	"time"

	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/codec/aescfb"
	"github.com/reinit/coward/roles/common/codec/aesgcm"
	"github.com/reinit/coward/roles/common/codec/key"
	"github.com/reinit/coward/roles/common/codec/marker"
	"github.com/reinit/coward/roles/common/transceiver"
)

// Vars
var (
	aesPrefixerOptions = []string{
		"Key", "Request-Prefix", "Respond-Prefix",
	}
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

func aesCFB128Builder(configuration []string) transceiver.CodecBuilder {
	cfgStr := aesSettingBuilder(configuration)
	timedKey := key.Timed([]byte(cfgStr), 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func() (rw.Codec, error) {
		return aescfb.AESCFB(timedKey, 16, timedMarkers, timedMarkerLock)
	}
}

func aesCFB256Builder(configuration []string) transceiver.CodecBuilder {
	cfgStr := aesSettingBuilder(configuration)
	timedKey := key.Timed([]byte(cfgStr), 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func() (rw.Codec, error) {
		return aescfb.AESCFB(timedKey, 32, timedMarkers, timedMarkerLock)
	}
}

func aesGCM128Builder(configuration []string) transceiver.CodecBuilder {
	cfgStr := aesSettingBuilder(configuration)
	timedKey := key.Timed([]byte(cfgStr), 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func() (rw.Codec, error) {
		return aesgcm.AESGCM(timedKey, 16, timedMarkers, timedMarkerLock)
	}
}

func aesGCM256Builder(configuration []string) transceiver.CodecBuilder {
	cfgStr := aesSettingBuilder(configuration)
	timedKey := key.Timed([]byte(cfgStr), 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func() (rw.Codec, error) {
		return aesgcm.AESGCM(timedKey, 32, timedMarkers, timedMarkerLock)
	}
}
