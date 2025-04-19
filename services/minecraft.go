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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	MinecraftServerName = "MinecraftServer"
)

// MinecraftServer represents the service for handling Minecraft commands.
type MinecraftServer struct {
	MLService
	config       *MinecraftConfig
	name         string
	cmd          *exec.Cmd          // Hold the running command
	stdinPipe    io.WriteCloser     // Pipe to server's stdin
	stdoutPipe   io.ReadCloser      // Pipe from server's stdout
	stderrPipe   io.ReadCloser      // Pipe from server's stderr
	serverCtx    context.Context    // Context specifically for the server process goroutine
	serverCancel context.CancelFunc // Function to cancel the server context
	serverWg     sync.WaitGroup     // WaitGroup for server goroutines
	isRunning    bool               // Flag indicating if the server process is running
	mu           sync.Mutex         // Mutex to protect access to shared resources (cmd, pipes, isRunning)
}

// NewMinecraftServer creates a new MinecraftServer instance with the given context and configuration.
func NewMinecraftServer(ctx context.Context) (Service, error) {
	mc := NewMinecraftConfig()
	globalConf, ok := ctx.Value(MoLingConfigKey).(*MoLingConfig)
	if !ok {
		return nil, fmt.Errorf("MinecraftServer: invalid global config type: %T", ctx.Value(MoLingConfigKey))
	}

	logger, ok := ctx.Value(MoLingLoggerKey).(zerolog.Logger)
	if !ok {
		return nil, fmt.Errorf("MinecraftServer: invalid logger type: %T", ctx.Value(MoLingLoggerKey))
	}

	loggerNameHook := zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, msg string) {
		e.Str("Service", MinecraftServerName)
	})

	// Create a cancellable context for the server process and its monitoring goroutines
	serverCtx, serverCancel := context.WithCancel(context.Background()) // Use Background, manage lifecycle internally

	ms := &MinecraftServer{
		MLService:    NewMLService(ctx, logger.Hook(loggerNameHook), globalConf),
		config:       mc,
		serverCtx:    serverCtx,
		serverCancel: serverCancel,
		isRunning:    false,
	}

	//Init loads config and sets up tools/prompts
	//We defer starting the server until after config is loaded.
	err := ms.init() // Call init explicitly after config load
	if err != nil {
		return nil, err
	}

	return ms, nil
}

// Init initializes the Minecraft server by adding tools and prompts.
// This should be called *after* LoadConfig.
func (ms *MinecraftServer) Init() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	loggerNameHook := zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, msg string) {
		e.Str("Service", string(ms.Name()))
	})
	ms.logger = ms.logger.Hook(loggerNameHook)

	// Add a prompt handler for Minecraft assistance
	pe := PromptEntry{
		prompt: mcp.Prompt{
			Name:        "minecraft_prompt",
			Description: "Get the relevant functions and prompts of the Minecraft Command API.",
		},
		phf: ms.handlePrompt, // Consider loading prompt from file here
	}
	ms.AddPrompt(pe)

	// Register tools (commands)
	ms.registerTools()

	// Start the server process in a goroutine *after* config is loaded and tools are registered
	ms.serverWg.Add(1)
	go func() {
		defer ms.serverWg.Done()
		err := ms.startMinecraftServerProcess() // Renamed internal function
		if err != nil && !errors.Is(err, context.Canceled) {
			ms.logger.Error().Err(err).Msg("Minecraft server process failed")
			ms.mu.Lock()
			ms.isRunning = false
			ms.mu.Unlock()
		}
	}()

	// Wait briefly for the server to potentially start up
	// A more robust check would involve parsing server logs for a "Done" message
	//ms.logger.Info().Msgf("Waiting up to %d seconds for server startup...", ms.config.StartupTimeout)
	//time.Sleep(time.Duration(ms.config.StartupTimeout) * time.Second) // Simple delay
	ms.logger.Info().Msg("Assumed server startup complete (or timeout reached). Ready for commands.")

	return nil
}

// registerTools adds all the Minecraft command tools.
func (ms *MinecraftServer) registerTools() {
	ms.AddTool(mcp.NewTool(
		"minecraft_fill",
		mcp.WithDescription("Fill the specified region with blocks"),
		mcp.WithString("x1", mcp.Description("Starting X coordinate"), mcp.Required()),
		mcp.WithString("y1", mcp.Description("Starting Y coordinate"), mcp.Required()),
		mcp.WithString("z1", mcp.Description("Starting Z coordinate"), mcp.Required()),
		mcp.WithString("x2", mcp.Description("Ending X coordinate"), mcp.Required()),
		mcp.WithString("y2", mcp.Description("Ending Y coordinate"), mcp.Required()),
		mcp.WithString("z2", mcp.Description("Ending Z coordinate"), mcp.Required()),
		mcp.WithString("block", mcp.Description("Block ID (e.g., minecraft:stone)"), mcp.Required()),
		mcp.WithString("oldBlockHandling", mcp.Description("How to handle existing blocks (replace, destroy, keep, hollow, outline) (optional)")),
	), ms.handleFill)

	ms.AddTool(mcp.NewTool(
		"minecraft_setblock",
		mcp.WithDescription("Set a block at the specified position"),
		mcp.WithString("x", mcp.Description("X coordinate"), mcp.Required()),
		mcp.WithString("y", mcp.Description("Y coordinate"), mcp.Required()),
		mcp.WithString("z", mcp.Description("Z coordinate"), mcp.Required()),
		mcp.WithString("block", mcp.Description("Block ID (e.g., minecraft:torch[lit=true])"), mcp.Required()),
		mcp.WithString("oldBlockHandling", mcp.Description("How to handle existing blocks (replace, destroy, keep) (optional)")),
	), ms.handleSetblock)

	ms.AddTool(mcp.NewTool(
		"minecraft_clone",
		mcp.WithDescription("Clone blocks from one region to another"),
		mcp.WithString("x1", mcp.Description("Source starting X coordinate"), mcp.Required()),
		mcp.WithString("y1", mcp.Description("Source starting Y coordinate"), mcp.Required()),
		mcp.WithString("z1", mcp.Description("Source starting Z coordinate"), mcp.Required()),
		mcp.WithString("x2", mcp.Description("Source ending X coordinate"), mcp.Required()),
		mcp.WithString("y2", mcp.Description("Source ending Y coordinate"), mcp.Required()),
		mcp.WithString("z2", mcp.Description("Source ending Z coordinate"), mcp.Required()),
		mcp.WithString("x", mcp.Description("Destination X coordinate"), mcp.Required()),
		mcp.WithString("y", mcp.Description("Destination Y coordinate"), mcp.Required()),
		mcp.WithString("z", mcp.Description("Destination Z coordinate"), mcp.Required()),
		mcp.WithString("maskMode", mcp.Description("Mask mode (replace, masked, filtered) (optional, default: replace)")), // Renamed from filterMode for clarity
		mcp.WithString("cloneMode", mcp.Description("Clone mode (force, move, normal) (optional, default: normal)")),
		mcp.WithString("filterBlock", mcp.Description("Filter block ID (required if maskMode is 'filtered')")),
	), ms.handleClone)

	ms.AddTool(mcp.NewTool(
		"minecraft_summon",
		mcp.WithDescription("Summon an entity at the specified position"),
		mcp.WithString("entity", mcp.Description("Entity ID (e.g., minecraft:pig)"), mcp.Required()),
		mcp.WithString("x", mcp.Description("X coordinate"), mcp.Required()),
		mcp.WithString("y", mcp.Description("Y coordinate"), mcp.Required()),
		mcp.WithString("z", mcp.Description("Z coordinate"), mcp.Required()),
		mcp.WithString("nbt", mcp.Description("NBT data for the entity (optional, JSON format)")), // Renamed from dataTag
	), ms.handleSummon)

	ms.AddTool(mcp.NewTool(
		"minecraft_execute",
		mcp.WithDescription("Execute a command with conditions. Build subcommands using 'as', 'at', 'positioned', 'if', 'unless', etc., ending with 'run <command>'."),
		mcp.WithString("subcommands", mcp.Description("The full execute subcommand chain (e.g., 'as @a at @s if block ~ ~-1 ~ minecraft:grass run say Hello')"), mcp.Required()),
	), ms.handleExecute)

	ms.AddTool(mcp.NewTool(
		"minecraft_give",
		mcp.WithDescription("Give an item to a player"),
		mcp.WithString("target", mcp.Description("Target player selector (e.g., @p, PlayerName)"), mcp.Required()),
		mcp.WithString("item", mcp.Description("Item ID (e.g., minecraft:diamond_sword)"), mcp.Required()),
		mcp.WithNumber("amount", mcp.Description("Amount (optional, default: 1)")),
	), ms.handleGive)

	ms.AddTool(mcp.NewTool(
		"minecraft_teleport",
		mcp.WithDescription("Teleport entities"),
		mcp.WithString("target", mcp.Description("Target entity selector (e.g., @p, PlayerName)"), mcp.Required()),
		mcp.WithString("destination", mcp.Description("Destination coordinates (x y z) or entity selector"), mcp.Required()),
		mcp.WithString("rotation", mcp.Description("Rotation (yaw pitch) (optional)")),
	), ms.handleTeleport)
}

// Helper function for extracting and validating string parameters
func getStringArg(args map[string]interface{}, key string, required bool) (string, error) {
	val, ok := args[key]
	if !ok {
		if required {
			return "", fmt.Errorf("missing required parameter: %s", key)
		}
		return "", nil // Optional parameter not present
	}
	strVal, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string, got %T", key, val)
	}
	if required && strVal == "" {
		return "", fmt.Errorf("required parameter %s cannot be empty", key)
	}
	return strVal, nil
}

// Helper function for extracting and validating coordinate parameters
func getCoordArgs(args map[string]interface{}, keys ...string) ([]string, error) {
	coords := make([]string, len(keys))
	for i, key := range keys {
		coordStr, err := getStringArg(args, key, true)
		if err != nil {
			return nil, err
		}
		// Basic validation: check if it can be parsed as a float (allows integers, decimals, relative coords ~)
		if _, err := strconv.ParseFloat(strings.Replace(coordStr, "~", "0", -1), 64); err != nil && !strings.HasPrefix(coordStr, "~") {
			// Allow pure "~" but reject invalid numbers
			if coordStr != "~" {
				return nil, fmt.Errorf("invalid coordinate format for %s: %s", key, coordStr)
			}
		}
		coords[i] = coordStr
	}
	return coords, nil
}

// Helper function for validating block ID format (basic)
func validateBlockID(blockID string) error {
	if blockID == "" {
		return fmt.Errorf("block ID cannot be empty")
	}
	// Basic check: should ideally contain ':' unless it's a very old format (which we might not support)
	// Allows for block states like minecraft:stone[variant=andesite]
	// This is a weak check, a proper validation requires knowing all block IDs.
	if !strings.Contains(blockID, ":") {
		// Allow simple IDs for now, but log a warning
		// ms.logger.Warn().Str("blockId", blockID).Msg("Block ID does not contain ':', assuming default namespace 'minecraft:'")
		// Alternatively, enforce the colon:
		return fmt.Errorf("invalid block ID format: %s (expected namespace:id, e.g., minecraft:stone)", blockID)
	}
	return nil
}

// Close stops the Minecraft server process gracefully and cleans up resources.
func (ms *MinecraftServer) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.logger.Info().Msg("Closing Minecraft server service...")

	if !ms.isRunning || ms.cmd == nil || ms.cmd.Process == nil {
		ms.logger.Info().Msg("Server process not running or already stopped.")
		ms.serverCancel()  // Ensure context is cancelled even if process wasn't running
		ms.serverWg.Wait() // Wait for any lingering goroutines
		return nil
	}

	// 1. Send shutdown command
	if ms.stdinPipe != nil && ms.config.ShutdownCommand != "" {
		ms.logger.Info().Str("command", ms.config.ShutdownCommand).Msg("Sending shutdown command to Minecraft server")
		_, err := ms.stdinPipe.Write([]byte(ms.config.ShutdownCommand + "\n"))
		if err != nil {
			ms.logger.Error().Err(err).Msg("Failed to write shutdown command to server stdin")
			// Continue with other shutdown steps even if writing fails
		}
		// Give the server a moment to process the stop command
		time.Sleep(2 * time.Second)
	}

	// 2. Close stdin pipe
	if ms.stdinPipe != nil {
		err := ms.stdinPipe.Close()
		if err != nil {
			ms.logger.Warn().Err(err).Msg("Error closing server stdin pipe")
		}
		ms.stdinPipe = nil
	}

	// 3. Cancel the server context (signals monitoring goroutines to stop)
	ms.logger.Debug().Msg("Cancelling server context")
	ms.serverCancel()

	// 4. Wait for the process to exit (optional with timeout)
	// cmd.Wait() is called in the goroutine started by startMinecraftServerProcess
	// We wait for that goroutine (and the I/O goroutines) to finish.
	ms.logger.Debug().Msg("Waiting for server process and I/O goroutines to exit...")
	ms.serverWg.Wait()
	ms.logger.Info().Msg("Server process and goroutines finished.")

	// 5. Release process resources (redundant if Wait succeeded, but good practice)
	if ms.cmd != nil && ms.cmd.Process != nil {
		_ = ms.cmd.Process.Release()
	}

	ms.isRunning = false
	ms.cmd = nil
	ms.logger.Info().Msg("Minecraft server service closed.")
	return nil
}

// Config returns the configuration of the service as a string.
func (ms *MinecraftServer) Config() string {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	cfg, err := json.MarshalIndent(ms.config, "", "  ") // Use Indent for readability
	if err != nil {
		ms.logger.Err(err).Msg("failed to marshal config")
		return "{}"
	}
	return string(cfg)
}

// Name returns the name of the service.
func (ms *MinecraftServer) Name() MoLingServerType {
	return MinecraftServerName
}

// LoadConfig loads the configuration from a JSON object.
func (ms *MinecraftServer) LoadConfig(jsonData map[string]interface{}) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.logger.Info().Msg("Loading MinecraftServer config...")
	err := mergeJSONToStruct(ms.config, jsonData)
	if err != nil {
		return fmt.Errorf("failed to merge JSON config: %w", err)
	}
	err = ms.config.Check()
	if err != nil {
		return fmt.Errorf("invalid Minecraft config: %w", err)
	}
	ms.logger.Info().Msg("MinecraftServer config loaded and checked successfully.")
	// Config is loaded, now Init can be called to start the server process
	return nil
}

// startMinecraftServerProcess starts the actual Minecraft server process.
// This runs in its own goroutine managed by Init.
func (ms *MinecraftServer) startMinecraftServerProcess() error {
	ms.mu.Lock()
	if ms.isRunning {
		ms.mu.Unlock()
		ms.logger.Warn().Msg("Attempted to start server process, but it is already running.")
		return fmt.Errorf("server already running")
	}

	ms.logger.Info().Msg("Attempting to start Minecraft server process...")
	ms.logger.Info().Str("path", ms.config.ServerRootPath).Msg("Changing directory")
	err := os.Chdir(ms.config.ServerRootPath)
	if err != nil {
		ms.mu.Unlock()
		ms.logger.Err(err).Str("path", ms.config.ServerRootPath).Msg("Failed to change directory")
		return fmt.Errorf("failed to change directory to %s: %w", ms.config.ServerRootPath, err)
	}

	// Prepare command arguments
	javaArgs := strings.Fields(ms.config.JvmMemoryArgs) // Split memory args string
	args := append(javaArgs, "-jar", ms.config.ServerJarFile)
	// Add "nogui" if not already present? Often needed for server jars.
	hasNoGui := false
	for _, arg := range args {
		if arg == "nogui" {
			hasNoGui = true
			break
		}
	}
	if !hasNoGui {
		args = append(args, "nogui")
	}

	ms.logger.Info().Str("java", ms.config.JavaPath).Strs("args", args).Msg("Preparing server command")
	cmd := exec.CommandContext(ms.serverCtx, ms.config.JavaPath, args...)
	cmd.Dir = ms.config.ServerRootPath // Ensure command runs in the correct directory

	// Get pipes
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		ms.mu.Unlock()
		ms.logger.Err(err).Msg("Failed to get stdin pipe")
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	ms.stdinPipe = stdinPipe

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		ms.mu.Unlock()
		ms.logger.Err(err).Msg("Failed to get stdout pipe")
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	ms.stdoutPipe = stdoutPipe

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		ms.mu.Unlock()
		ms.logger.Err(err).Msg("Failed to get stderr pipe")
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}
	ms.stderrPipe = stderrPipe

	ms.cmd = cmd
	ms.mu.Unlock() // Unlock before starting potentially long-running operations

	// Start the process
	ms.logger.Info().Msg("Starting server process...")
	err = cmd.Start()
	if err != nil {
		ms.mu.Lock()
		ms.isRunning = false // Ensure flag is false if start fails
		ms.mu.Unlock()
		ms.logger.Err(err).Msg("Failed to start Minecraft server process")
		// Clean up pipes? StdinPipe.Close() maybe?
		return fmt.Errorf("failed to start server process: %w", err)
	}

	ms.mu.Lock()
	ms.isRunning = true
	pid := cmd.Process.Pid
	ms.logger.Info().Int("pid", pid).Msg("Minecraft server process started successfully.")
	ms.mu.Unlock()

	// Goroutine to log stdout
	ms.serverWg.Add(1)
	go ms.logPipe("stdout", ms.stdoutPipe)

	// Goroutine to log stderr
	ms.serverWg.Add(1)
	go ms.logPipe("stderr", ms.stderrPipe)

	// REMOVED: Goroutine forwarding os.Stdin - Interaction is now via WriteCommand only
	// go func() {
	// 	_, err := io.Copy(stdinPipe, os.Stdin) // DO NOT DO THIS
	// 	if err != nil {
	// 		ms.logger.Error().Err(err).Msg("Error forwarding input")
	// 	}
	// }()

	// Wait for the process to exit in this goroutine
	err = cmd.Wait()

	ms.mu.Lock()
	ms.isRunning = false // Mark as not running once Wait() returns
	ms.mu.Unlock()

	if err != nil {
		// Check if the error is due to context cancellation (expected during shutdown)
		select {
		case <-ms.serverCtx.Done():
			ms.logger.Info().Msg("Minecraft server process stopped via context cancellation.")
			return context.Canceled // Return a specific error for cancellation
		default:
			// Process exited with an actual error
			ms.logger.Error().Err(err).Int("pid", pid).Msg("Minecraft server process exited with error")
			return fmt.Errorf("server process exited with error: %w", err)
		}
	}

	ms.logger.Info().Int("pid", pid).Msg("Minecraft server process exited successfully.")
	return nil
}

// logPipe reads from a pipe (stdout/stderr) and logs it line by line.
// Runs in its own goroutine managed by serverWg.
func (ms *MinecraftServer) logPipe(pipeName string, pipe io.ReadCloser) {
	defer ms.serverWg.Done()
	defer pipe.Close() // Ensure the pipe is closed when done

	scanner := bufio.NewScanner(pipe)
	logger := ms.logger.With().Str("pipe", pipeName).Logger()
	logger.Debug().Msg("Started logging pipe")

	for scanner.Scan() {
		line := scanner.Text()
		// Log server output - adjust level as needed (e.g., Info or Debug)
		logger.Info().Msg(line) // Using Info to make server logs visible by default

		// TODO: Add basic log parsing here if needed for specific events (e.g., "Done loading" or errors)
		// if strings.Contains(line, "[Server thread/INFO]: Done") { ... }
	}

	if err := scanner.Err(); err != nil {
		// Don't log error if it's due to context cancellation closing the pipe
		select {
		case <-ms.serverCtx.Done():
			logger.Debug().Msg("Pipe closed due to context cancellation.")
		default:
			logger.Error().Err(err).Msg("Error reading from pipe")
		}
	}
	logger.Debug().Msg("Stopped logging pipe")
}

// WriteCommand writes a command to the Minecraft server's standard input.
func (ms *MinecraftServer) WriteCommand(command string) (*mcp.CallToolResult, error) {
	ms.mu.Lock() // Lock to ensure exclusive access to stdinPipe and isRunning flag
	defer ms.mu.Unlock()

	ms.logger.Debug().Str("command", command).Msg("Attempting to write command to Minecraft server")

	if !ms.isRunning || ms.stdinPipe == nil {
		ms.logger.Error().Msg("Cannot write command: Minecraft server is not running or stdin pipe is nil")
		return mcp.NewToolResultError("Minecraft server is not running"), nil
	}

	// Add newline character required by Minecraft console
	commandWithNewline := command + "\n"

	_, err := ms.stdinPipe.Write([]byte(commandWithNewline))
	if err != nil {
		ms.logger.Error().Err(err).Str("command", command).Msg("Failed to write command to server stdin")
		// Check if the error is because the pipe is closed (server might have crashed)
		if errors.Is(err, os.ErrClosed) || strings.Contains(err.Error(), "pipe is closed") {
			ms.isRunning = false // Mark as not running if pipe is closed
			return mcp.NewToolResultError("Failed to write command: Server connection lost (pipe closed)"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to write command: %v", err)), nil
	}

	ms.logger.Info().Str("command", command).Msg("Command sent successfully to Minecraft server")

	// IMPORTANT: Lack of Feedback
	// This function currently returns success immediately after writing.
	// It does NOT wait for or parse the server's response from stdout/stderr.
	// A more robust implementation would:
	// 1. Read stdout/stderr after sending the command.
	// 2. Parse the output for success messages (e.g., "Filled ... blocks") or error messages.
	// 3. Return success/failure based on the parsed output.
	// This requires knowledge of Minecraft's specific log formats and is more complex.
	// For now, we assume success if the write operation doesn't fail. Check server logs for actual results.
	return mcp.NewToolResultText(fmt.Sprintf("Command '%s' sent to server. Check server logs for results.", command)), nil
}

func init() {
	RegisterServ(MinecraftServerName, NewMinecraftServer)
}
