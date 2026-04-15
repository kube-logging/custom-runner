// Copyright © 2022 Cisco Systems, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"encoding/json"
)

const (
	ErrAlreadyRunning = "process already running: %s"
	ErrNoProcFound    = "process not found: %s"
	ErrUNKCommand     = "unknown API command: %s"
)

type APICommandProto func(key string, args []byte) ApiResult

type ApiResult struct {
	Success  bool        `json:"success" yaml:"success"`
	Error    error       `json:"error,omitempty" yaml:"error,omitempty"`
	Response interface{} `json:"response,omitempty" yaml:"response,omitempty"`
}

func (a ApiResult) MarshalJSON() ([]byte, error) {
	am := a
	if a.Error != nil {
		am.Error = MarshalableError{a.Error}
	}

	type ar ApiResult

	return json.Marshal(ar(am))
}

type MarshalableError struct {
	error
}

func (m MarshalableError) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Error())
}
