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
	"encoding/hex"
	"fmt"
	"strings"
)

// prefixerSetting Prefixer Setting
type prefixerSetting map[string][]byte

// prefixerSettingParser parse prefixerSetting
func prefixerSettingParser(
	configuration []string,
	prefixerOptions []string,
	usageErr error,
) (prefixerSetting, error) {
	currentSettingContext := ""
	currentSetting := make(prefixerSetting, len(prefixerOptions))

	for pIdx := range prefixerOptions {
		currentSetting[prefixerOptions[pIdx]] = make([]byte, 0, 256)
	}

	for sIdx := range configuration {
		clIdx := strings.Index(configuration[sIdx], ":")

		// Check whether or not to switch setting context
		if clIdx >= 0 {
			currentSettingContext =
				strings.TrimSpace(configuration[sIdx][:clIdx])

			_, scFound := currentSetting[currentSettingContext]

			if !scFound {
				return prefixerSetting{}, fmt.Errorf(
					"Unknown Option \"%s\": %s",
					currentSettingContext, usageErr)
			}

			clIdx++ // Skip ":" symbol
		} else {
			clIdx = 0
		}

		sc, scFound := currentSetting[currentSettingContext]

		if !scFound {
			return prefixerSetting{}, usageErr
		}

		sc = append(sc, configuration[sIdx][clIdx:]...)

		currentSetting[currentSettingContext] = sc
	}

	for sKey := range currentSetting {
		hexData, hexDataErr := hex.DecodeString(
			string(currentSetting[sKey]))

		if hexDataErr != nil {
			return prefixerSetting{}, fmt.Errorf(
				"Invalid value \"%s\" of \"%s\" option: %s. "+
					"It must be a valid string of hex",
				string(currentSetting[sKey]), sKey, hexDataErr)
		}

		currentSetting[sKey] = hexData
	}

	return currentSetting, nil
}

// prefixerSettingBuilder build prefixerSetting
func prefixerSettingBuilder(
	configuration []string, prefixerOptions []string) prefixerSetting {
	prefixer, prefixerErr := prefixerSettingParser(
		configuration, prefixerOptions, nil)

	if prefixerErr != nil {
		panic(fmt.Sprintf("Bad prefixer setting: %s", prefixerErr))
	}

	return prefixer
}

// prefixerSettingVerifier verify prefixerSetting
func prefixerSettingVerifier(
	configuration []string,
	v func(p prefixerSetting) error,
	prefixerOptions []string,
	usageErr error,
) error {
	prefixer, prefixerErr := prefixerSettingParser(
		configuration, prefixerOptions, usageErr)

	if prefixerErr != nil {
		return prefixerErr
	}

	if v == nil {
		return nil
	}

	vErr := v(prefixer)

	if vErr != nil {
		return vErr
	}

	return nil
}
