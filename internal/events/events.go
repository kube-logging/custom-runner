// Copyright © 2026 Kube logging authors
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

package events

// Kind identifies an event. The value is the configuration key it dispatches on,
// so config lookup needs no translation table.
type Kind string

const (
	OnStart Kind = "onStart"
	OnExit  Kind = "onExit"

	OnExec   Kind = "onExec"
	OnFinish Kind = "onFinish"
	OnError  Kind = "onError"

	OnFileCreate Kind = "onFileCreate"
	OnFileWrite  Kind = "onFileWrite"
	OnFileRemove Kind = "onFileRemove"
	OnFileRename Kind = "onFileRename"
	OnFileChmod  Kind = "onFileChmod"
)

// FileKinds lists every path-keyed kind, for collecting watch targets from config.
func FileKinds() []Kind {
	return []Kind{OnFileCreate, OnFileWrite, OnFileRemove, OnFileRename, OnFileChmod}
}

// Event is a single occurrence. Subject is the process key for lifecycle kinds and
// the path for file kinds; it is empty for onStart and onExit.
type Event struct {
	Kind    Kind
	Subject string
	Err     error
}

// New builds an event for a subject: a process key, or a watched path.
func New(kind Kind, subject string) Event {
	return Event{Kind: kind, Subject: subject}
}

func Error(key string, err error) Event {
	return Event{Kind: OnError, Subject: key, Err: err}
}
