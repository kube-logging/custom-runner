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
