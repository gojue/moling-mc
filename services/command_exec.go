//go:build !windows

// Copyright 2025 CFC4N <cfc4n.cs@gmail.com>. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Repository: https://github.com/gojue/moling

package services

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// executeCommand executes a command and returns its output.
func (cs *CommandServer) executeCommand(command string) (string, error) {
	var cmd *exec.Cmd
	ctx, cfunc := context.WithTimeout(context.Background(), time.Second*3)
	defer cfunc()
	cmd = exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		switch {
		case errors.Is(err, exec.ErrNotFound):
			// 命令未找到
			return "", errors.New("command not found")
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			// 超时时仅返回输出，不返回错误
			return string(output), nil
		default:
			return string(output), nil
		}
	}

	return string(output), nil
}

// isAllowedCommand checks if the command is allowed based on the configuration.
func (cs *CommandServer) isAllowedCommand(command string) bool {

	// 检查命令是否在允许的列表中
	for _, allowed := range cs.config.AllowedCommands {
		if strings.HasPrefix(command, allowed) {
			return true
		}
	}

	// 如果命令包含管道符，进一步检查每个子命令
	if strings.Contains(command, "|") {
		parts := strings.Split(command, "|")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if !cs.isAllowedCommand(part) {
				return false
			}
		}
		return true
	}

	return false
}
