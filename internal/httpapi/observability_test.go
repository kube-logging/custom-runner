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

package httpapi

import (
	"strings"
	"testing"
)

func TestReadinessStatus(t *testing.T) {
	testCases := []struct {
		name    string
		setup   func(*Readiness)
		wantOK  bool
		wantWhy string
	}{
		{
			name:    "not started",
			setup:   func(*Readiness) {},
			wantOK:  false,
			wantWhy: "starting up",
		},
		{
			name:   "started and healthy",
			setup:  func(r *Readiness) { r.SetStarted(true) },
			wantOK: true,
		},
		{
			name: "a failed watch keeps it unready",
			setup: func(r *Readiness) {
				r.Degrade("not watching /etc/app/..data: no such file or directory")
				r.SetStarted(true)
			},
			wantOK:  false,
			wantWhy: "not watching /etc/app/..data",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ready := &Readiness{}
			tc.setup(ready)

			gotOK, reasons := ready.Status()
			if gotOK != tc.wantOK {
				t.Fatalf("Status() ok = %v, want %v (reasons %v)", gotOK, tc.wantOK, reasons)
			}
			if tc.wantWhy != "" && !strings.Contains(strings.Join(reasons, "; "), tc.wantWhy) {
				t.Errorf("reasons = %v, want one containing %q", reasons, tc.wantWhy)
			}
		})
	}
}
