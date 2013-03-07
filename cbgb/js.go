//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the
//  License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/robertkrimen/otto"
)

// Originally from github.com/couchbaselabs/walrus, but capitalized.
// TODO: Push this back to walrus.
// Converts an Otto value to a Go value. Handles all JSON-compatible types.
func OttoToGo(value otto.Value) (interface{}, error) {
	if value.IsBoolean() {
		return value.ToBoolean()
	} else if value.IsNull() || value.IsUndefined() {
		return nil, nil
	} else if value.IsNumber() {
		return value.ToFloat()
	} else if value.IsString() {
		return value.ToString()
	} else {
		switch value.Class() {
		case "Array":
			return OttoToGoArray(value.Object())
		}
	}
	return nil, fmt.Errorf("Unsupported Otto value: %v", value)
}

// Originally from github.com/couchbaselabs/walrus, but capitalized.
// TODO: Push this back to walrus.
func OttoToGoArray(array *otto.Object) ([]interface{}, error) {
	lengthVal, err := array.Get("length")
	if err != nil {
		return nil, err
	}
	length, err := lengthVal.ToInteger()
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, length)
	for i := 0; i < int(length); i++ {
		item, err := array.Get(strconv.Itoa(i))
		if err != nil {
			return nil, err
		}
		result[i], err = OttoToGo(item)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func OttoFromGoArray(o *otto.Otto, arr []interface{}) (otto.Value, error) {
	jarr, err := json.Marshal(arr)
	if err != nil {
		return otto.UndefinedValue(),
			fmt.Errorf("could not jsonify arr, err: %v", err)
	}
	oarr, err := o.Object("({v:" + string(jarr) + "})")
	if err != nil {
		return otto.UndefinedValue(),
			fmt.Errorf("could not convert arr, err: %v", err)
	}
	ovarr, err := oarr.Get("v")
	if err != nil {
		return otto.UndefinedValue(),
			fmt.Errorf("could not convert oarr, err: %v", err)
	}
	if ovarr.Class() != "Array" {
		return otto.UndefinedValue(),
			fmt.Errorf("expected ovarr to be array, got: %#v, %v, jarr: %v",
				ovarr, ovarr.Class(), string(jarr))
	}
	return ovarr, nil
}

func OttoNewFunction(o *otto.Otto, f string) (otto.Value, error) {
	fn, err := o.Object("(" + f + ")")
	if err != nil {
		return otto.UndefinedValue(),
			fmt.Errorf("could not eval function, err: %v", err)
	}
	if fn.Class() != "Function" {
		return otto.UndefinedValue(),
			fmt.Errorf("fn not a function, was: %v", fn.Class())
	}
	fnv := fn.Value()
	if fnv.Class() != "Function" {
		return otto.UndefinedValue(),
			fmt.Errorf("fnv not a function, was: %v", fnv.Class())
	}
	return fnv, nil
}