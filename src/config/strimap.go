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

package config

type Strimap map[string]interface{}

func (s Strimap) GetIn(keys ...interface{}) interface{} {
	if len(keys) == 0 {
		return s
	}
	key, subKeys := keys[0], keys[1:]
	v, ok := key.(string)
	if !ok {
		return nil
	}
	subStore, ok := s[v]
	if !ok {
		return nil
	}
	switch v := subStore.(type) {
	case map[string]interface{}:
		return Strimap(v).GetIn(subKeys...)
	case Strimap:
		return v.GetIn(subKeys...)
	case []interface{}:
		return StriArray(v).GetIn(subKeys...)
	default:
		if len(subKeys) > 0 {
			return nil
		}
		return v
	}
}

type StriArray []interface{}

func (s StriArray) GetIn(keys ...interface{}) interface{} {
	if len(keys) == 0 {
		return []interface{}(s)
	}
	key, subKeys := keys[0], keys[1:]
	intKey, ok := key.(int)
	if !ok {
		return nil
	}

	if len(s) <= intKey || intKey < 0 {
		return nil
	}
	subStore := s[intKey]
	switch v := subStore.(type) {
	case Strimap:
		return v.GetIn(subKeys...)
	case []interface{}:
		return StriArray(v).GetIn(subKeys...)
	default:
		if len(subKeys) > 0 {
			return nil
		}
		return v
	}

}
