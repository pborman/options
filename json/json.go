// Copyright 2018 Paul Borman
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and

// Package json provides JSON flag decoding for the github.com/pborman/options
// packge.  This package registers itself with the options package as the
// json encoding.  Normal usage is one of:
//
//	options.NewFlags("flags").SetEncoding(json.Decoder)
//
//	Flags options.Flags `getopt:"--flags json encoded command line parameter" encoding:"json"`
//
// The JSON encoded data should look something like:
//
//	{
//		"name": "bob",
//		"v": true,
//		"n": 42
//	}
package json

import (
	"bytes"
	"encoding/json"

	"github.com/pborman/options"
)

// Decoder decodes and returns data or an errort.  Data must be a JSON blob.
// Unlike calling json.Unmarshal, Decoder sets UseNumber() on the decoder so
// numbers are returned as json.Numbers (strings).
func Decoder(data []byte) (map[string]interface{}, error) {
	decoder := json.NewDecoder(bytes.NewBuffer(data))
	decoder.UseNumber()

	m := map[string]interface{}{}
	for decoder.More() {
		if err := decoder.Decode(&m); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func init() {
	options.RegisterEncoding("json", Decoder)
}
