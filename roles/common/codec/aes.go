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
	"io"
	"sync"
	"time"

	"github.com/reinit/coward/roles/common/codec/aescfb"
	"github.com/reinit/coward/roles/common/codec/aesgcm"
	"github.com/reinit/coward/roles/common/codec/key"
	"github.com/reinit/coward/roles/common/codec/marker"
	"github.com/reinit/coward/roles/common/transceiver"
)

// AESCFB128 return a AESCFB128 Transceiver Codec
func AESCFB128() transceiver.Codec {
	return transceiver.Codec{
		Name:   "aes-cfb-128-hmac",
		Usage:  "Input a string of letters as shared key (passphrase)",
		Build:  aesCFB128Builder,
		Verify: aesVerifier,
	}
}

// AESCFB256 return a AESCFB256 Transceiver Codec
func AESCFB256() transceiver.Codec {
	return transceiver.Codec{
		Name:   "aes-cfb-256-hmac",
		Usage:  "Input a string of letters as shared key (passphrase)",
		Build:  aesCFB256Builder,
		Verify: aesVerifier,
	}
}

// AESGCM128 return a AESGCM128 Transceiver Codec
func AESGCM128() transceiver.Codec {
	return transceiver.Codec{
		Name:   "aes-gcm-128",
		Usage:  "Input a string of letters as shared key (passphrase)",
		Build:  aesGCM128Builder,
		Verify: aesVerifier,
	}
}

// AESGCM256 return a AESGCM128 Transceiver Codec
func AESGCM256() transceiver.Codec {
	return transceiver.Codec{
		Name:   "aes-gcm-256",
		Usage:  "Input a string of letters as shared key (passphrase)",
		Build:  aesGCM256Builder,
		Verify: aesVerifier,
	}
}

func aesVerifier(configuration []byte) error {
	if len(configuration) < 16 {
		return errors.New("Shared Key was too short. " +
			"Make it at least 16 charactor long")
	}

	return nil
}

func aesCFB128Builder(configuration []byte) transceiver.CodecBuilder {
	timedKey := key.Timed(configuration, 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aescfb.AESCFB(conn, timedKey, 16, timedMarkers, timedMarkerLock)
	}
}

func aesCFB256Builder(configuration []byte) transceiver.CodecBuilder {
	timedKey := key.Timed(configuration, 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aescfb.AESCFB(conn, timedKey, 32, timedMarkers, timedMarkerLock)
	}
}

func aesGCM128Builder(configuration []byte) transceiver.CodecBuilder {
	timedKey := key.Timed(configuration, 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aesgcm.AESGCM(conn, timedKey, 16, timedMarkers, timedMarkerLock)
	}
}

func aesGCM256Builder(configuration []byte) transceiver.CodecBuilder {
	timedKey := key.Timed(configuration, 10*time.Second, time.Now)
	timedMarkers := marker.Timed(4096, 10*time.Second, time.Now)
	timedMarkerLock := &sync.Mutex{}

	return func(conn io.ReadWriter) (io.ReadWriter, error) {
		return aesgcm.AESGCM(conn, timedKey, 32, timedMarkers, timedMarkerLock)
	}
}
