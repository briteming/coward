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

package config

import (
	"reflect"

	"github.com/reinit/coward/common/parameter"
)

type parseItem struct {
	Tag    string
	Parent valueReflect
	Param  *parameter.Value
	Config *reflect.Value
	Fields fields
}

type sliceRefer struct {
	Indirect bool
	Name     string
	Tag      string
	Parent   valueReflect
	Label    *parameter.Value
	Field    *reflect.Value
	Slice    []*reflect.Value
}

type parsedConfig struct {
	Value valueReflect
	Tag   string
	Start int
	End   int
}

// Parse parameter string and fillin configuration
func (c *configurator) Parse(parameters []byte) error {
	param, paramErr := parameter.New(parameters, 3)

	if paramErr != nil {
		return paramErr
	}

	rootFieldVerifier := c.config.MethodByName("CheckValue")
	slices := make([]sliceRefer, 0, 256)
	items := make([]parseItem, 0, 256)
	parsedConfigs := make([]parsedConfig, 0, 256)

	items = append(items, parseItem{
		Tag:    "",
		Parent: valueReflect{},
		Param:  param.Value(),
		Config: &c.config,
		Fields: c.fields,
	})

	for {
		if len(items) <= 0 {
			break
		}

		currentItem := items[0]
		items = items[1:]

		currentParam := currentItem.Param
		currentCfgPtr := newValueReflect(*currentItem.Config).PointerValue()
		currentConfig := currentCfgPtr.DirectElem()
		currentFields := currentItem.Fields
		currentParentTag := currentItem.Tag

		if currentItem.Parent.IsValid() {
			initer := currentCfgPtr.MethodByName("Init")

			if initer.IsValid() {
				initer.Call([]reflect.Value{
					currentItem.Parent.Value,
				})
			}
		}

		for _, label := range currentParam.Labels() {
			currentLabel := label.Label()

			if currentLabel == nil {
				return newParseError(
					ErrConfigurationWithoutLabel, "",
					label.Current().Start(), label.Current().End(), parameters)
			}

			currentFieldTag := string(currentLabel.Data())

			currentField, fieldGetErr := currentFields.GetByTag(
				currentFieldTag)

			if fieldGetErr != nil {
				return newParseError(
					ErrUndefinedParameter, currentFieldTag,
					currentLabel.Start(), currentLabel.End(), parameters)
			}

			configFieldRef := currentConfig.FieldByName(currentField.Name)

			if !configFieldRef.IsValid() {
				return newParseError(
					ErrInvalidField, currentFieldTag,
					currentLabel.Start(), currentLabel.End(), parameters)
			}

			// Get the value itself instead of ptr
			fieldRefl := newValueReflect(configFieldRef).DirectElem()

			sliceItemLimit := -1

			switch fieldRefl.Kind() {
			case reflect.Array:
				sliceItemLimit = fieldRefl.Len()

				fallthrough
			case reflect.Slice:
				values := label.Values()

				fieldTypeElem := fieldRefl.Type().Elem()
				fieldTypeElemType := newTypeReflect(fieldTypeElem).DirectElem()

				if sliceItemLimit > -1 && len(values) != sliceItemLimit {
					return newParseError(
						ErrInsufficientArray, currentFieldTag,
						currentLabel.Start(), currentLabel.End(), parameters)
				}

				fieldReflRaw := fieldRefl.Extract()
				newSliceRefer := sliceRefer{
					Indirect: fieldTypeElem.Kind() == reflect.Ptr,
					Name:     currentField.Name,
					Tag:      currentFieldTag,
					Parent:   currentCfgPtr,
					Label:    currentLabel,
					Field:    &fieldReflRaw,
					Slice:    []*reflect.Value{},
				}

				switch fieldTypeElemType.Kind() {
				case reflect.Array:
					fallthrough
				case reflect.Slice:
					items = append(items, parseItem{
						Tag:    currentFieldTag,
						Parent: currentCfgPtr,
						Param:  label.Current(),
						Fields: currentField.Sub,
						Config: &fieldReflRaw,
					})

				case reflect.Struct:
					for _, value := range label.Values() {
						newSliceItem := reflect.New(
							fieldTypeElemType.Extract())

						newSliceRefer.Slice = append(newSliceRefer.Slice,
							&newSliceItem)

						items = append(items, parseItem{
							Tag:    currentFieldTag,
							Parent: currentCfgPtr,
							Param:  value,
							Fields: currentField.Sub,
							Config: &newSliceItem,
						})
					}

					slices = append(slices, newSliceRefer)

				default:
					bareValueSetErr := fieldRefl.SetBareSlice(label.Values())

					if bareValueSetErr != nil {
						return newParseError(
							bareValueSetErr, currentFieldTag,
							currentLabel.Start(), currentLabel.End(),
							parameters)
					}
				}

			case reflect.Struct:
				// Struct? Create one and add it for next loop
				for _, value := range label.Values() {
					items = append(items, parseItem{
						Tag:    currentFieldTag,
						Parent: currentCfgPtr,
						Param:  value,
						Fields: currentField.Sub,
						Config: &configFieldRef,
					})
				}

			default:
				bareValueSetErr := fieldRefl.SetBareValue(label.Values())

				if bareValueSetErr != nil {
					return newParseError(
						bareValueSetErr, currentFieldTag,
						currentLabel.Start(), currentLabel.End(),
						parameters)
				}

				if rootFieldVerifier.IsValid() {
					verifyErr := rootFieldVerifier.Interface().(func(
						string, interface{}) error)(
						fieldRefl.Type().Name(), fieldRefl.Interface())

					if verifyErr != nil {
						return newParseError(
							verifyErr, currentFieldTag,
							currentLabel.Start(), currentLabel.End(),
							parameters)
					}
				}

				verifier := currentCfgPtr.MethodByName(
					"Verify" + currentField.Name)

				if !verifier.IsValid() {
					continue
				}

				verifyErr := verifier.Interface().(func() error)()

				if verifyErr == nil {
					continue
				}

				return newParseError(
					verifyErr, currentFieldTag,
					currentLabel.Start(), currentLabel.End(),
					parameters)
			}
		}

		parsedConfigs = append(parsedConfigs, parsedConfig{
			Value: currentCfgPtr,
			Tag:   currentParentTag,
			Start: currentParam.Start(),
			End:   currentParam.End(),
		})
	}

	for _, sliceRefers := range slices {
		sliceData := sliceRefers.Field.Slice(0, sliceRefers.Field.Len())

		for _, sliceReferSlice := range sliceRefers.Slice {
			if sliceRefers.Indirect {
				sliceData = reflect.Append(sliceData, *sliceReferSlice)

				continue
			}

			sliceData = reflect.Append(sliceData, sliceReferSlice.Elem())
		}

		sliceRefers.Field.Set(sliceData)

		verifier := sliceRefers.Parent.MethodByName(
			"Verify" + sliceRefers.Name)

		if !verifier.IsValid() {
			continue
		}

		verifyErr := verifier.Interface().(func() error)()

		if verifyErr == nil {
			continue
		}

		return newParseError(
			verifyErr, sliceRefers.Tag,
			sliceRefers.Label.Start(), sliceRefers.Label.End(),
			parameters)
	}

	for afterChkIdx := len(parsedConfigs) - 1; afterChkIdx >= 0; afterChkIdx-- {
		verifier := parsedConfigs[afterChkIdx].Value.MethodByName("Verify")

		if !verifier.IsValid() {
			continue
		}

		verifyErr := verifier.Interface().(func() error)()

		if verifyErr == nil {
			continue
		}

		return newParseError(
			verifyErr, parsedConfigs[afterChkIdx].Tag,
			parsedConfigs[afterChkIdx].Start, parsedConfigs[afterChkIdx].End,
			parameters)
	}

	return nil
}
