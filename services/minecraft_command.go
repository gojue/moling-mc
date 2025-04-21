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
	"os"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// handlePrompt loads the prompt from the minecraft.md file.
func (ms *MinecraftServer) handlePrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	promptPath := ms.config.PromptPath
	contentBytes, err := os.ReadFile(promptPath)
	if err != nil {
		ms.logger.Error().Err(err).Str("path", promptPath).Msg("Failed to read minecraft prompt file, used default prompt")
		// Fallback to a basic message if file reading fails
		return &mcp.GetPromptResult{
			Description: "Minecraft Command API - Error loading detailed prompt.",
			Messages: []mcp.PromptMessage{
				{Role: mcp.RoleUser, Content: mcp.TextContent{Type: "text", Text: `
You are now a specialized Minecraft Building Assistant, an expert in Minecraft commands and construction techniques. Your purpose is to help players create amazing structures, understand command syntax, and solve building challenges in Minecraft.

## Your Knowledge and Capabilities
- You have expert knowledge of all Minecraft building commands including /fill, /setblock, /clone, and more
- You understand coordinates, block IDs, data values, and NBT tags
- You can suggest efficient building techniques for various structures
- You can generate command strings for complex builds
- You can troubleshoot command errors and building problems

## How to Respond
1. Always provide complete command syntax when suggesting commands
2. Include coordinates in examples (e.g., /setblock 100 64 100 minecraft:stone)
3. Explain what each part of a command does
4. For complex builds, break down the process into step-by-step instructions
5. Suggest alternative approaches when appropriate
6. Provide both basic and advanced techniques depending on the user's expertise level

## Important Information to Include
- Always specify which Minecraft version your advice applies to when version differences matter
- Include warnings about commands that might lag the game when used on large areas
- Mention common mistakes or pitfalls with certain commands
- Explain the difference between different block handling modes (replace, destroy, keep, etc.)

## Examples You Should Be Ready to Provide
- Command templates for common structures (walls, floors, domes, spheres)
- Ways to use /clone efficiently for repetitive structures
- How to use /execute to create dynamic or conditional builds
- Techniques for creating gradient effects or patterns with blocks
- Solutions for working within command block character limits

When I ask you about building something in Minecraft, provide me with the exact commands I would need to create it, along with clear explanations and any relevant tips.

`}},
			},
		}, nil // Return nil error so the service can still function minimally
	}

	return &mcp.GetPromptResult{
		Description: "Minecraft Command API", // This description might be shown to the user
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser, // Use RoleSystem for instructions
				Content: mcp.TextContent{
					Type: "text",
					Text: string(contentBytes),
				},
			},
		},
	}, nil
}

// handleFill implements the /fill command.
func (ms *MinecraftServer) handleFill(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	coords1, err := getCoordArgs(request.Params.Arguments, "x1", "y1", "z1")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	coords2, err := getCoordArgs(request.Params.Arguments, "x2", "y2", "z2")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	block, err := getStringArg(request.Params.Arguments, "block", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := validateBlockID(block); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	oldBlockHandling, _ := getStringArg(request.Params.Arguments, "oldBlockHandling", false) // Optional

	// Construct the command
	command := fmt.Sprintf("/fill %s %s %s %s %s %s %s", coords1[0], coords1[1], coords1[2], coords2[0], coords2[1], coords2[2], block)

	if oldBlockHandling != "" {
		// Validate handling mode? (replace, destroy, keep, hollow, outline)
		validModes := map[string]bool{"replace": true, "destroy": true, "keep": true, "hollow": true, "outline": true}
		if !validModes[oldBlockHandling] {
			return mcp.NewToolResultError(fmt.Sprintf("invalid oldBlockHandling mode: %s", oldBlockHandling)), nil
		}
		command += " " + oldBlockHandling
	}

	return ms.WriteCommand(command)
}

// handleSetblock implements the /setblock command.
func (ms *MinecraftServer) handleSetblock(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	coords, err := getCoordArgs(request.Params.Arguments, "x", "y", "z")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	block, err := getStringArg(request.Params.Arguments, "block", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := validateBlockID(block); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	oldBlockHandling, _ := getStringArg(request.Params.Arguments, "oldBlockHandling", false) // Optional

	// Construct the command
	command := fmt.Sprintf("/setblock %s %s %s %s", coords[0], coords[1], coords[2], block)

	if oldBlockHandling != "" {
		// Validate handling mode? (replace, destroy, keep)
		validModes := map[string]bool{"replace": true, "destroy": true, "keep": true}
		if !validModes[oldBlockHandling] {
			return mcp.NewToolResultError(fmt.Sprintf("invalid oldBlockHandling mode: %s", oldBlockHandling)), nil
		}
		command += " " + oldBlockHandling
	}

	return ms.WriteCommand(command)
}

// handleClone implements the /clone command.
func (ms *MinecraftServer) handleClone(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	coords1, err := getCoordArgs(request.Params.Arguments, "x1", "y1", "z1")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	coords2, err := getCoordArgs(request.Params.Arguments, "x2", "y2", "z2")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	destCoords, err := getCoordArgs(request.Params.Arguments, "x", "y", "z")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	maskMode, _ := getStringArg(request.Params.Arguments, "maskMode", false)
	cloneMode, _ := getStringArg(request.Params.Arguments, "cloneMode", false)
	filterBlock, _ := getStringArg(request.Params.Arguments, "filterBlock", false) // Required only if maskMode is filtered

	// Construct the command
	command := fmt.Sprintf("/clone %s %s %s %s %s %s %s %s %s",
		coords1[0], coords1[1], coords1[2],
		coords2[0], coords2[1], coords2[2],
		destCoords[0], destCoords[1], destCoords[2])

	// Validate and add optional modes
	validMaskModes := map[string]bool{"replace": true, "masked": true, "filtered": true}
	validCloneModes := map[string]bool{"force": true, "move": true, "normal": true}

	if maskMode != "" {
		if !validMaskModes[maskMode] {
			return mcp.NewToolResultError(fmt.Sprintf("invalid maskMode: %s", maskMode)), nil
		}
		command += " " + maskMode
		if maskMode == "filtered" {
			if filterBlock == "" {
				return mcp.NewToolResultError("filterBlock is required when maskMode is 'filtered'"), nil
			}
			if err := validateBlockID(filterBlock); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid filterBlock: %s", err.Error())), nil
			}
			command += " " + filterBlock
		}
	} else if filterBlock != "" {
		// If filterBlock is provided, maskMode must be specified (usually 'filtered')
		return mcp.NewToolResultError("maskMode must be specified when filterBlock is provided"), nil
	}

	if cloneMode != "" {
		if !validCloneModes[cloneMode] {
			return mcp.NewToolResultError(fmt.Sprintf("invalid cloneMode: %s", cloneMode)), nil
		}
		command += " " + cloneMode
	}

	return ms.WriteCommand(command)
}

// handleSummon implements the /summon command.
func (ms *MinecraftServer) handleSummon(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	entity, err := getStringArg(request.Params.Arguments, "entity", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// Basic entity ID validation (similar to block ID)
	if !strings.Contains(entity, ":") {
		// return mcp.NewToolResultError(fmt.Sprintf("invalid entity ID format: %s (expected namespace:id)", entity)), nil
		ms.logger.Warn().Str("entityId", entity).Msg("Entity ID does not contain ':', assuming default namespace 'minecraft:'")
	}

	coords, err := getCoordArgs(request.Params.Arguments, "x", "y", "z")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	nbt, _ := getStringArg(request.Params.Arguments, "nbt", false) // Optional

	// Construct the command
	command := fmt.Sprintf("/summon %s %s %s %s", entity, coords[0], coords[1], coords[2])

	if nbt != "" {
		// Basic NBT validation: should start with { and end with }
		if !(strings.HasPrefix(nbt, "{") && strings.HasSuffix(nbt, "}")) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid NBT format: %s (must be enclosed in {})", nbt)), nil
		}
		command += " " + nbt
	}

	return ms.WriteCommand(command)
}

// handleExecute implements the /execute command.
func (ms *MinecraftServer) handleExecute(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	subcommands, err := getStringArg(request.Params.Arguments, "subcommands", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Basic validation: ensure it contains "run"
	if !strings.Contains(subcommands, " run ") {
		return mcp.NewToolResultError("execute command must contain 'run' subcommand"), nil
	}

	// Construct the command
	command := "/execute " + subcommands

	return ms.WriteCommand(command)
}

// handleGive implements the /give command.
func (ms *MinecraftServer) handleGive(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	target, err := getStringArg(request.Params.Arguments, "target", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	item, err := getStringArg(request.Params.Arguments, "item", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// Basic item ID validation
	if !strings.Contains(item, ":") {
		// return mcp.NewToolResultError(fmt.Sprintf("invalid item ID format: %s (expected namespace:id)", item)), nil
		ms.logger.Warn().Str("itemId", item).Msg("Item ID does not contain ':', assuming default namespace 'minecraft:'")
	}

	// Optional amount
	amount := 1 // Default amount
	amountVal, amountOk := request.Params.Arguments["amount"]
	if amountOk {
		amountFloat, ok := amountVal.(float64)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("parameter amount must be a number, got %T", amountVal)), nil
		}
		if amountFloat < 1 || amountFloat != float64(int(amountFloat)) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid amount: %v (must be a positive integer)", amountFloat)), nil
		}
		amount = int(amountFloat)
	}

	// Construct the command
	command := fmt.Sprintf("/give %s %s %d", target, item, amount)

	return ms.WriteCommand(command)
}

// handleTeleport implements the /teleport or /tp command.
func (ms *MinecraftServer) handleTeleport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	target, err := getStringArg(request.Params.Arguments, "target", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	destination, err := getStringArg(request.Params.Arguments, "destination", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// Destination could be coords "x y z" or an entity selector "@e[...]"
	// We don't validate the content deeply here, assume LLM provides valid format.

	rotation, _ := getStringArg(request.Params.Arguments, "rotation", false) // Optional "yaw pitch"

	// Construct the command
	command := fmt.Sprintf("/teleport %s %s", target, destination)

	if rotation != "" {
		// Basic validation: should contain two parts (yaw and pitch)
		parts := strings.Fields(rotation)
		if len(parts) != 2 {
			return mcp.NewToolResultError(fmt.Sprintf("invalid rotation format: %s (expected 'yaw pitch')", rotation)), nil
		}
		// Check if parts are numbers or relative ~
		for _, part := range parts {
			if _, err := strconv.ParseFloat(strings.Replace(part, "~", "0", -1), 64); err != nil && !strings.HasPrefix(part, "~") {
				if part != "~" {
					return mcp.NewToolResultError(fmt.Sprintf("invalid rotation value: %s", part)), nil
				}
			}
		}
		command += " " + rotation
	}

	return ms.WriteCommand(command)
}
