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
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/reinit/coward/common/parameter"
)

// Value Reflect errors
var (
	ErrValueReflectInvalidBoolString = errors.New(
		"Invalid bool string")

	ErrValueReflectUnsupportedType = errors.New(
		"Unsupported data type")

	ErrValueReflectAssignArrayValueToNonSliceType = errors.New(
		"Can't assign array to a non-slice type")
)

type reflectType struct {
	reflect.Type
}

func newTypeReflect(t reflect.Type) reflectType {
	return reflectType{
		Type: t,
	}
}

func (vt reflectType) Extract() reflect.Type {
	return vt.Type
}

func (vt reflectType) DirectElem() reflectType {
	currentReflect := vt.Type

	for {
		if currentReflect.Kind() != reflect.Ptr {
			break
		}

		currentReflect = currentReflect.Elem()
	}

	return newTypeReflect(currentReflect)
}

type valueReflect struct {
	reflect.Value
}

func newValueReflect(v reflect.Value) valueReflect {
	return valueReflect{
		Value: v,
	}
}

func (vr valueReflect) PointerValue() valueReflect {
	if vr.Value.Kind() == reflect.Ptr {
		return newValueReflect(vr.Value)
	}

	ptr := reflect.New(vr.Value.Type())

	ptr.Elem().Set(vr.Value)

	return newValueReflect(ptr)
}

func (vr valueReflect) isSlice() bool {
	vrType := vr.DirectElem().DirectType()

	if vrType.Kind() != reflect.Slice && vrType.Kind() != reflect.Array {
		return false
	}

	return true
}

func (vr valueReflect) setInt(b []byte, bitWise int) error {
	intVal, intValErr := strconv.ParseInt(string(b), 10, bitWise)

	if intValErr != nil {
		return intValErr
	}

	vr.DirectElem().SetInt(intVal)

	return nil
}

func (vr valueReflect) setInts(values []*parameter.Value, bitWise int) error {
	if !vr.isSlice() && len(values) > 1 {
		return ErrValueReflectAssignArrayValueToNonSliceType
	}

	for _, v := range values {
		setErr := vr.setInt(v.Data(), bitWise)

		if setErr == nil {
			continue
		}

		return setErr
	}

	return nil
}

func (vr valueReflect) setUint(b []byte, bitWise int) error {
	uintVal, uintValErr := strconv.ParseUint(string(b), 10, bitWise)

	if uintValErr != nil {
		return uintValErr
	}

	vr.DirectElem().SetUint(uintVal)

	return nil
}

func (vr valueReflect) setUints(
	values []*parameter.Value, bitWise int) error {
	if !vr.isSlice() && len(values) > 1 {
		return ErrValueReflectAssignArrayValueToNonSliceType
	}

	for _, v := range values {
		setErr := vr.setUint(v.Data(), bitWise)

		if setErr == nil {
			continue
		}

		return setErr
	}

	return nil
}

func (vr valueReflect) SetStringBytes(b []byte) error {
	vr.DirectElem().SetString(string(b))

	return nil
}

func (vr valueReflect) SetStringsParamValues(values []*parameter.Value) error {
	if !vr.isSlice() && len(values) > 1 {
		return ErrValueReflectAssignArrayValueToNonSliceType
	}

	for _, v := range values {
		setErr := vr.SetStringBytes(v.Data())

		if setErr == nil {
			continue
		}

		return setErr
	}

	return nil
}

func (vr valueReflect) SetBoolBytes(b []byte) error {
	tValue := strings.ToLower(string(b))

	if tValue == "yes" || tValue == "true" {
		vr.DirectElem().SetBool(true)

		return nil
	}

	if tValue == "no" || tValue == "false" {
		vr.DirectElem().SetBool(false)

		return nil
	}

	return ErrValueReflectInvalidBoolString
}

func (vr valueReflect) SetBoolsParamValues(values []*parameter.Value) error {
	if !vr.isSlice() && len(values) > 1 {
		return ErrValueReflectAssignArrayValueToNonSliceType
	}

	for _, v := range values {
		setErr := vr.SetBoolBytes(v.Data())

		if setErr == nil {
			continue
		}

		return setErr
	}

	return nil
}

func (vr valueReflect) SetFloat32Bytes(b []byte) error {
	floatVal, floatValErr := strconv.ParseFloat(string(b), 32)

	if floatValErr != nil {
		return floatValErr
	}

	vr.DirectElem().SetFloat(floatVal)

	return nil
}

func (vr valueReflect) SetFloat32sParamValues(values []*parameter.Value) error {
	if !vr.isSlice() && len(values) > 1 {
		return ErrValueReflectAssignArrayValueToNonSliceType
	}

	for _, v := range values {
		setErr := vr.SetFloat32Bytes(v.Data())

		if setErr == nil {
			continue
		}

		return setErr
	}

	return nil
}

func (vr valueReflect) SetFloat64Bytes(b []byte) error {
	floatVal, floatValErr := strconv.ParseFloat(string(b), 64)

	if floatValErr != nil {
		return floatValErr
	}

	vr.DirectElem().SetFloat(floatVal)

	return nil
}

func (vr valueReflect) SetFloat64sParamValues(values []*parameter.Value) error {
	if !vr.isSlice() && len(values) > 1 {
		return ErrValueReflectAssignArrayValueToNonSliceType
	}

	for _, v := range values {
		setErr := vr.SetFloat64Bytes(v.Data())

		if setErr == nil {
			continue
		}

		return setErr
	}

	return nil
}

func (vr valueReflect) SetIntBytes(b []byte) error {
	return vr.setInt(b, 64)
}

func (vr valueReflect) SetIntsParamValues(v []*parameter.Value) error {
	return vr.setInts(v, 64)
}

func (vr valueReflect) SetInt8Bytes(b []byte) error {
	return vr.setInt(b, 8)
}

func (vr valueReflect) SetInt8sParamValues(v []*parameter.Value) error {
	return vr.setInts(v, 8)
}

func (vr valueReflect) SetInt16Bytes(b []byte) error {
	return vr.setInt(b, 16)
}

func (vr valueReflect) SetInt16sParamValues(v []*parameter.Value) error {
	return vr.setInts(v, 16)
}

func (vr valueReflect) SetInt32Bytes(b []byte) error {
	return vr.setInt(b, 32)
}

func (vr valueReflect) SetInt32sParamValues(v []*parameter.Value) error {
	return vr.setInts(v, 32)
}

func (vr valueReflect) SetInt64Bytes(b []byte) error {
	return vr.setInt(b, 64)
}

func (vr valueReflect) SetInt64sParamValues(v []*parameter.Value) error {
	return vr.setInts(v, 64)
}

func (vr valueReflect) SetUintBytes(b []byte) error {
	return vr.setUint(b, 64)
}

func (vr valueReflect) SetUintsParamValues(v []*parameter.Value) error {
	return vr.setUints(v, 64)
}

func (vr valueReflect) SetUint8Bytes(b []byte) error {
	return vr.setUint(b, 8)
}

func (vr valueReflect) SetUint8sParamValues(v []*parameter.Value) error {
	return vr.setUints(v, 8)
}

func (vr valueReflect) SetUint16Bytes(b []byte) error {
	return vr.setUint(b, 16)
}

func (vr valueReflect) SetUint16sParamValues(v []*parameter.Value) error {
	return vr.setUints(v, 16)
}

func (vr valueReflect) SetUint32Bytes(b []byte) error {
	return vr.setUint(b, 32)
}

func (vr valueReflect) SetUint32sParamValues(v []*parameter.Value) error {
	return vr.setUints(v, 32)
}

func (vr valueReflect) SetUint64Bytes(b []byte) error {
	return vr.setUint(b, 64)
}

func (vr valueReflect) SetUint64sParamValues(v []*parameter.Value) error {
	return vr.setUints(v, 64)
}

func (vr *valueReflect) Replace(new reflect.Value) {
	vr.Value = new
}

func (vr valueReflect) Extract() reflect.Value {
	return vr.Value
}

func (vr valueReflect) DirectSlice(i, j int) valueReflect {
	return newValueReflect(vr.DirectElem().Slice(i, j))
}

func (vr valueReflect) DirectField(index int) valueReflect {
	return newValueReflect(vr.DirectElem().Field(index))
}

func (vr valueReflect) DirectType() reflectType {
	return newTypeReflect(vr.Value.Type()).DirectElem()
}

func (vr valueReflect) DirectElem() valueReflect {
	currentReflect := vr.Value

	for {
		if currentReflect.Kind() != reflect.Ptr {
			break
		}

		if currentReflect.IsNil() {
			newElm := reflect.New(currentReflect.Type().Elem())

			currentReflect.Set(newElm)

			continue
		}

		currentReflect = currentReflect.Elem()
	}

	return newValueReflect(currentReflect)
}

func (vr valueReflect) SetBareValue(vs []*parameter.Value) error {
	valueElem := vr.DirectElem()

	switch valueElem.Kind() {
	case reflect.String:
		return vr.SetStringsParamValues(vs)

	case reflect.Bool:
		return vr.SetBoolsParamValues(vs)

	case reflect.Float32:
		return vr.SetFloat32sParamValues(vs)
	case reflect.Float64:
		return vr.SetFloat64sParamValues(vs)

	case reflect.Int:
		return vr.SetIntsParamValues(vs)
	case reflect.Int8:
		return vr.SetInt8sParamValues(vs)
	case reflect.Int16:
		return vr.SetInt16sParamValues(vs)
	case reflect.Int32:
		return vr.SetInt32sParamValues(vs)
	case reflect.Int64:
		return vr.SetInt64sParamValues(vs)

	case reflect.Uint:
		return vr.SetUintsParamValues(vs)
	case reflect.Uint8:
		return vr.SetUint8sParamValues(vs)
	case reflect.Uint16:
		return vr.SetUint16sParamValues(vs)
	case reflect.Uint32:
		return vr.SetUint32sParamValues(vs)
	case reflect.Uint64:
		return vr.SetUint64sParamValues(vs)
	}

	return ErrValueReflectUnsupportedType
}

func (vr valueReflect) appendBareSliceItems(
	vs []*parameter.Value,
	setter func(vrRef *valueReflect, v *parameter.Value) error,
	sliceValue *valueReflect,
	sliceElemType reflect.Type,
	indirect bool,
) error {
	for _, v := range vs {
		valueRef := newValueReflect(reflect.New(sliceElemType))

		setErr := setter(&valueRef, v)

		if setErr != nil {
			return setErr
		}

		if indirect {
			sliceValue.Replace(reflect.Append(
				sliceValue.Extract(), valueRef.Extract()))
		} else {
			sliceValue.Replace(reflect.Append(
				sliceValue.Extract(), valueRef.Extract().Elem()))
		}
	}

	return nil
}

func (vr valueReflect) SetBareSlice(vs []*parameter.Value) error {
	var result error

	vrElem := vr.DirectElem()
	fSlice := vrElem.DirectSlice(0, vrElem.Len())
	fSliceTypeElm := fSlice.DirectType().Elem()
	fSliceTypeElmType := newTypeReflect(fSliceTypeElm).DirectElem().Extract()

	switch fSliceTypeElmType.Kind() {
	case reflect.String:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetStringBytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Bool:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetBoolBytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Float32:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetFloat32Bytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Float64:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetFloat64Bytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Int:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetIntBytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Int8:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetInt8Bytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Int16:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetInt16Bytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Int32:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetInt32Bytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Int64:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetInt64Bytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Uint:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetUintBytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Uint8:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetUint8Bytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Uint16:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetUint16Bytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Uint32:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetUint32Bytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	case reflect.Uint64:
		result = vr.appendBareSliceItems(
			vs, func(fRef *valueReflect, v *parameter.Value) error {
				return fRef.SetUint64Bytes(v.Data())
			}, &fSlice, fSliceTypeElmType, fSliceTypeElm.Kind() == reflect.Ptr)

	default:
		return ErrValueReflectUnsupportedType
	}

	vrElem.Set(fSlice.Extract())

	return result
}
