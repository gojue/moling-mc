package services

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// handlePrompt loads the prompt from the minecraft.md file.
func (ms *MinecraftServer) handlePrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	promptPath := filepath.Join(ms.MlConfig().BasePath, "prompts", "minecraft.md")
	contentBytes, err := os.ReadFile(promptPath)
	if err != nil {
		ms.logger.Error().Err(err).Str("path", promptPath).Msg("Failed to read minecraft prompt file")
		// Fallback to a basic message if file reading fails
		return &mcp.GetPromptResult{
			Description: "Minecraft Command API - Error loading detailed prompt.",
			Messages: []mcp.PromptMessage{
				{Role: mcp.RoleUser, Content: mcp.TextContent{Type: "text", Text: "You are a Minecraft command assistant."}},
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
