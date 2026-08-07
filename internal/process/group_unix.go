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

//go:build unix

package process

import (
	"os/exec"
	"syscall"
)

// Isolate gives the command its own process group, so a kill reaches the
// grandchildren a shell spawns and not just the shell.
func Isolate(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return KillGroup(cmd) }
}

// KillGroup signals the command's whole process group.
func KillGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	// Isolate made the child its group leader, so the negative pid is the group.
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err == nil {
		return nil
	}

	return cmd.Process.Kill()
}
