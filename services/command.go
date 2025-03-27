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

// Package services Description: This file contains the implementation of the CommandServer interface for macOS and  Linux.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
)

// CommandServer implements the Service interface and provides methods to execute named commands.
type CommandServer struct {
	MLService
	config    *CommandConfig
	osName    string
	osVersion string
}

// NewCommandServer creates a new CommandServer with the given allowed commands.
func NewCommandServer(ctx context.Context, args []string) (Service, error) {
	var err error
	cc := NewCommandConfig(args)
	lger, ok := ctx.Value(MoLingLoggerKey).(zerolog.Logger)
	if !ok {
		return nil, fmt.Errorf("CommandServer: invalid logger type")
	}

	loggerNameHook := zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, msg string) {
		e.Str("Service", "CommandServer")
	})

	cs := &CommandServer{
		MLService: MLService{
			ctx:    ctx,
			logger: lger.Hook(loggerNameHook),
		},
		config: cc,
	}

	err = cs.init()
	if err != nil {
		return nil, err
	}

	pe := PromptEntry{
		prompt: mcp.Prompt{
			Name:        "command_prompt",
			Description: fmt.Sprintf("You are a command-line tool assistant, using macOS 15.3.3 system commands to help users troubleshoot network issues, system performance, file searching, and statistics, among other things."),
			//Arguments:   make([]mcp.PromptArgument, 0),
		},
		phf: cs.handlePrompt,
	}
	cs.AddPrompt(pe)
	cs.AddTool(mcp.NewTool(
		"execute_command",
		mcp.WithDescription("Execute a named command.Only support command execution on macOS and will strictly follow safety guidelines, ensuring that commands are safe and secure"),
		mcp.WithString("command",
			mcp.Description("The command to execute"),
			mcp.Required(),
		),
	), cs.handleExecuteCommand)

	return cs, nil
}

func (cs *CommandServer) handlePrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: fmt.Sprintf(""),
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: "This is a simple prompt without arguments.",
				},
			},
		},
	}, nil
}

// handleExecuteCommand handles the execution of a named command.
func (cs *CommandServer) handleExecuteCommand(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	command, ok := request.Params.Arguments["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command must be a string")
	}

	// Check if the command is allowed
	if !cs.isCommandAllowed(command) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Command '%s' is not allowed", command),
				},
			},
			IsError: true,
		}, nil
	}

	// Execute the command
	output, err := cs.executeCommand(command)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error executing command: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: output,
			},
		},
	}, nil
}

// isCommandAllowed checks if a command is in the list of allowed commands.
func (cs *CommandServer) isCommandAllowed(command string) bool {
	//return true
	if len(cs.config.AllowedCommands) == 0 {
		return true
	}

	for _, allowed := range cs.config.AllowedCommands {
		if command == allowed {
			return true
		}
	}
	return false
}

// Config returns the configuration of the service as a string.
func (cs *CommandServer) Config() string {
	cfg, err := json.Marshal(cs.config)
	if err != nil {
		cs.logger.Err(err).Msg("failed to marshal config")
		return "{}"
	}
	cs.logger.Info().Str("config", string(cfg)).Msg("CommandServer config")
	return string(cfg)
}

func (cs *CommandServer) Name() string {
	return "CommandServer"
}

func (bs *CommandServer) Close() error {
	// Cancel the context to stop the browser
	return nil
}

func init() {
	RegisterServ(NewCommandServer)
}
