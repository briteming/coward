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

package config

import (
	"reflect"
	"strings"

	"github.com/reinit/coward/common/print"
)

type carrier struct {
	Path   string
	Fields *fields
	Type   reflect.Type
}

// Configurator is configuration parser
type Configurator interface {
	Parse(parameters []byte) error
	Help(w print.Common)
}

// configurator implements Configurator
type configurator struct {
	fields fields
	config reflect.Value
}

// Import a struct pointer which point to a configuration struct
func Import(cfg interface{}) (Configurator, error) {
	configRoot := reflect.ValueOf(cfg)

	if configRoot.Kind() != reflect.Ptr {
		return nil, ErrConfigurationMustBeStructPointer
	}

	cfgReflect := configRoot

	configRoot = configRoot.Elem()

	if configRoot.Kind() != reflect.Struct {
		return nil, ErrConfigurationMustBeStructPointer
	}

	rootFields := fields{}

	carriers := make([]carrier, 0, 256)

	carriers = append(carriers, carrier{
		Path:   "",
		Fields: &rootFields,
		Type:   configRoot.Type(),
	})

	for {
		if len(carriers) <= 0 {
			break
		}

		currentCarrier := carriers[0]
		carriers = carriers[1:]

		currentCarrierType := currentCarrier.Type

		for {
			if currentCarrierType.Kind() != reflect.Ptr {
				break
			}

			currentCarrierType = currentCarrierType.Elem()
		}

		if currentCarrierType.Kind() == reflect.Array ||
			currentCarrierType.Kind() == reflect.Slice {

			switch currentCarrierType.Elem().Kind() {
			case reflect.Ptr:
				carriers = append(carriers, carrier{
					Path:   currentCarrier.Path,
					Fields: currentCarrier.Fields,
					Type:   reflect.SliceOf(currentCarrierType.Elem().Elem()),
				})

			case reflect.Array:
				fallthrough
			case reflect.Slice:
				subField := &field{
					Name:        currentCarrierType.Name(),
					Path:        currentCarrier.Path,
					Tag:         "[]",
					Tags:        []string{},
					Description: "A array slice of:",
					Sub:         fields{},
				}

				*currentCarrier.Fields = append(
					*currentCarrier.Fields, subField)

				carriers = append(carriers, carrier{
					Path:   currentCarrier.Path,
					Fields: &subField.Sub,
					Type:   currentCarrierType.Elem(),
				})

			case reflect.Struct:
				carriers = append(carriers, carrier{
					Path:   currentCarrier.Path,
					Fields: currentCarrier.Fields,
					Type:   currentCarrierType.Elem(),
				})
			}

			continue
		}

		if currentCarrierType.Kind() != reflect.Struct {
			return nil, newFieldError(
				ErrConfigurationFieldUnsupportedDataType,
				currentCarrier.Path)
		}

		fieldNum := currentCarrierType.NumField()
		fieldNameMutex := map[string]bool{}

		for fieldIdx := 0; fieldIdx < fieldNum; fieldIdx++ {
			fieldType := currentCarrierType.Field(fieldIdx)

			fieldTag := fieldType.Tag.Get("cfg")

			if fieldTag == "" {
				continue
			}

			fieldTags := []string{}

			fieldDescription := ""
			fieldDescriptionIndex := strings.Index(fieldTag, ":")

			if fieldDescriptionIndex >= 0 {
				fieldDescription = strings.TrimSpace(
					fieldTag[fieldDescriptionIndex+1:])

				fieldDescMethod := configRoot.MethodByName("GetDescription")

				if fieldDescMethod.IsValid() {
					fDesc := fieldDescMethod.Interface().(func(string) string)(
						currentCarrier.Path + "/" + fieldType.Name)

					if fDesc != "" {
						fieldDescription += "\r\n\r\n" + fDesc
					}
				}

				fieldTag = fieldTag[:fieldDescriptionIndex]
			}

			// Just had a litte thought that I should not add
			// tag texts here
			// fieldTags = append(fieldTags, fieldType.Name)
			// fieldNameMutex[fieldType.Name] = true

			for _, fTag := range strings.Split(fieldTag, ",") {
				fTagName := strings.TrimSpace(fTag)

				_, fTagExisted := fieldNameMutex[fTagName]

				if fTagExisted {
					return nil, newFieldError(
						ErrConfigurationFieldTagNameConfilcted,
						currentCarrier.Path+"/"+fieldType.Name)
				}

				fieldTags = append(fieldTags, fTagName)
				fieldNameMutex[fTagName] = true
			}

			fieldTypeType := fieldType.Type

			for {
				if fieldTypeType.Kind() != reflect.Ptr {
					break
				}

				fieldTypeType = fieldTypeType.Elem()
			}

			switch fieldTypeType.Kind() {
			case reflect.Bool:
				fallthrough
			case reflect.String:
				fallthrough
			case reflect.Int:
				fallthrough
			case reflect.Int8:
				fallthrough
			case reflect.Int16:
				fallthrough
			case reflect.Int32:
				fallthrough
			case reflect.Int64:
				fallthrough
			case reflect.Uint:
				fallthrough
			case reflect.Uint8:
				fallthrough
			case reflect.Uint16:
				fallthrough
			case reflect.Uint32:
				fallthrough
			case reflect.Uint64:
				fallthrough
			case reflect.Float32:
				fallthrough
			case reflect.Float64:
				*currentCarrier.Fields = append(
					*currentCarrier.Fields, &field{
						Name:        fieldType.Name,
						Path:        currentCarrier.Path + "/" + fieldType.Name,
						Tag:         "-" + strings.Join(fieldTags, ", -"),
						Tags:        fieldTags,
						Description: fieldDescription,
						Sub:         fields{},
					})

			case reflect.Array:
				fallthrough
			case reflect.Slice:
				subField := &field{
					Name: fieldType.Name,
					Path: currentCarrier.Path + "/" + fieldType.Name,
					Tag: "-" + strings.Join(
						fieldTags, " [], -") + " []",
					Tags:        fieldTags,
					Description: fieldDescription,
					Sub:         fields{},
				}

				*currentCarrier.Fields = append(
					*currentCarrier.Fields, subField)

				carriers = append(carriers, carrier{
					Path:   subField.Path,
					Fields: &subField.Sub,
					Type:   fieldType.Type,
				})

			case reflect.Struct:
				subField := &field{
					Name: fieldType.Name,
					Path: currentCarrier.Path + "/" + fieldType.Name,
					Tag: "-" + strings.Join(
						fieldTags, " {}, -") + " {}",
					Tags:        fieldTags,
					Description: fieldDescription,
					Sub:         fields{},
				}

				*currentCarrier.Fields = append(
					*currentCarrier.Fields, subField)

				// Add this value to the waitingFields
				carriers = append(carriers, carrier{
					Path:   subField.Path,
					Fields: &subField.Sub,
					Type:   fieldType.Type,
				})

			default:
				return nil, newFieldError(
					ErrConfigurationFieldUnsupportedKind,
					currentCarrier.Path+"/"+fieldType.Name)
			}
		}
	}

	return &configurator{
		fields: rootFields,
		config: cfgReflect,
	}, nil
}
