/*
 * Copyright 2025 CFC4N <cfc4n.cs@gmail.com>. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Repository: https://github.com/gojue/moling
 */

package services

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleGameRule implements the /gamerule command.
func (ms *MinecraftServer) handleGameRule(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ruleName, err := getStringArg(request.Params.Arguments, "rule", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Value is optional for querying the current rule value
	value, valuePresent := request.Params.Arguments["value"]
	command := fmt.Sprintf("/gamerule %s", ruleName)

	if valuePresent {
		// Convert to appropriate string based on type
		switch v := value.(type) {
		case bool:
			command = fmt.Sprintf("%s %t", command, v)
		case float64:
			command = fmt.Sprintf("%s %d", command, int(v))
		case string:
			command = fmt.Sprintf("%s %s", command, v)
		default:
			return mcp.NewToolResultError(fmt.Sprintf("invalid value type for gamerule: %T", value)), nil
		}
	}

	return ms.WriteCommand(command)
}

// handleTime implements the /time command.
func (ms *MinecraftServer) handleTime(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	subcommand, err := getStringArg(request.Params.Arguments, "subcommand", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	validSubcommands := map[string]bool{"set": true, "add": true, "query": true}
	if !validSubcommands[subcommand] {
		return mcp.NewToolResultError(fmt.Sprintf("invalid time subcommand: %s", subcommand)), nil
	}

	// Different subcommands need different parameters
	var command string
	if subcommand == "query" {
		timeSpec, err := getStringArg(request.Params.Arguments, "timeSpec", true)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		validTimeSpecs := map[string]bool{"daytime": true, "gametime": true, "day": true}
		if !validTimeSpecs[timeSpec] {
			return mcp.NewToolResultError(fmt.Sprintf("invalid time specification: %s", timeSpec)), nil
		}
		command = fmt.Sprintf("/time %s %s", subcommand, timeSpec)
	} else {
		// For "set" and "add"
		value, err := getStringArg(request.Params.Arguments, "value", true)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		command = fmt.Sprintf("/time %s %s", subcommand, value)
	}

	return ms.WriteCommand(command)
}

// handleWeather implements the /weather command.
func (ms *MinecraftServer) handleWeather(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	weatherType, err := getStringArg(request.Params.Arguments, "type", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	validTypes := map[string]bool{"clear": true, "rain": true, "thunder": true}
	if !validTypes[weatherType] {
		return mcp.NewToolResultError(fmt.Sprintf("invalid weather type: %s", weatherType)), nil
	}

	command := fmt.Sprintf("/weather %s", weatherType)

	// Duration is optional (in seconds)
	if durationVal, ok := request.Params.Arguments["duration"]; ok {
		duration, ok := durationVal.(float64)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("duration must be a number, got %T", durationVal)), nil
		}
		if duration < 1 || duration != float64(int(duration)) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid duration: %v (must be a positive integer)", duration)), nil
		}
		command = fmt.Sprintf("%s %d", command, int(duration))
	}

	return ms.WriteCommand(command)
}

// handleEffect implements the /effect command.
func (ms *MinecraftServer) handleEffect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	subcommand, err := getStringArg(request.Params.Arguments, "subcommand", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	target, err := getStringArg(request.Params.Arguments, "target", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if subcommand == "clear" {
		command := fmt.Sprintf("/effect clear %s", target)

		// Effect is optional for 'clear'
		effect, _ := getStringArg(request.Params.Arguments, "effect", false)
		if effect != "" {
			command = fmt.Sprintf("%s %s", command, effect)
		}

		return ms.WriteCommand(command)
	} else if subcommand == "give" {
		effect, err := getStringArg(request.Params.Arguments, "effect", true)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Format command
		command := fmt.Sprintf("/effect give %s %s", target, effect)

		// Add optional parameters
		seconds := 0
		amplifier := 0
		hideParticles := false

		// Duration in seconds (optional)
		if secondsVal, ok := request.Params.Arguments["seconds"]; ok {
			secondsFloat, ok := secondsVal.(float64)
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("seconds must be a number, got %T", secondsVal)), nil
			}
			if secondsFloat < 1 || secondsFloat != float64(int(secondsFloat)) {
				return mcp.NewToolResultError(fmt.Sprintf("invalid seconds: %v (must be a positive integer)", secondsFloat)), nil
			}
			seconds = int(secondsFloat)
			command = fmt.Sprintf("%s %d", command, seconds)

			// Amplifier (optional, but requires seconds)
			if amplifierVal, ok := request.Params.Arguments["amplifier"]; ok {
				amplifierFloat, ok := amplifierVal.(float64)
				if !ok {
					return mcp.NewToolResultError(fmt.Sprintf("amplifier must be a number, got %T", amplifierVal)), nil
				}
				if amplifierFloat < 0 || amplifierFloat > 255 || amplifierFloat != float64(int(amplifierFloat)) {
					return mcp.NewToolResultError(fmt.Sprintf("invalid amplifier: %v (must be an integer between 0 and 255)", amplifierFloat)), nil
				}
				amplifier = int(amplifierFloat)
				command = fmt.Sprintf("%s %d", command, amplifier)

				// Hide particles (optional, but requires seconds and amplifier)
				if hideParticlesVal, ok := request.Params.Arguments["hideParticles"]; ok {
					var ok bool
					hideParticles, ok = hideParticlesVal.(bool)
					if !ok {
						return mcp.NewToolResultError(fmt.Sprintf("hideParticles must be a boolean, got %T", hideParticlesVal)), nil
					}
					command = fmt.Sprintf("%s %v", command, hideParticles)
				}
			}
		}

		return ms.WriteCommand(command)
	} else {
		return mcp.NewToolResultError(fmt.Sprintf("invalid effect subcommand: %s", subcommand)), nil
	}
}

// handleDifficulty implements the /difficulty command.
func (ms *MinecraftServer) handleDifficulty(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	difficulty, err := getStringArg(request.Params.Arguments, "difficulty", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	validDifficulties := map[string]bool{
		"peaceful": true, "easy": true, "normal": true, "hard": true,
		"0": true, "1": true, "2": true, "3": true,
	}
	if !validDifficulties[difficulty] {
		return mcp.NewToolResultError(fmt.Sprintf("invalid difficulty: %s", difficulty)), nil
	}

	command := fmt.Sprintf("/difficulty %s", difficulty)
	return ms.WriteCommand(command)
}

// handleSpawnpoint implements the /spawnpoint command.
func (ms *MinecraftServer) handleSpawnpoint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	target, _ := getStringArg(request.Params.Arguments, "target", false) // Optional, defaults to the command executor

	command := "/spawnpoint"
	if target != "" {
		command = fmt.Sprintf("%s %s", command, target)
	}

	// Position is optional
	if _, ok := hasAllCoordArgs(request.Params.Arguments, "x", "y", "z"); ok {
		coords, err := getCoordArgs(request.Params.Arguments, "x", "y", "z")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		command = fmt.Sprintf("%s %s %s %s", command, coords[0], coords[1], coords[2])
	}

	return ms.WriteCommand(command)
}

// Helper function to check if a set of coordinates are all present
func hasAllCoordArgs(args map[string]interface{}, keys ...string) (bool, bool) {
	for _, key := range keys {
		if _, ok := args[key]; !ok {
			return false, false
		}
	}
	return true, true
}

// Helper function for extracting and validating boolean parameters
func getBoolArg(args map[string]interface{}, key string, required bool) (bool, error) {
	val, ok := args[key]
	if !ok {
		if required {
			return false, fmt.Errorf("missing required parameter: %s", key)
		}
		return false, nil // Optional parameter not present
	}
	boolVal, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("parameter %s must be a boolean, got %T", key, val)
	}
	return boolVal, nil
}
