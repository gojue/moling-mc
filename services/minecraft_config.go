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
	"fmt"
	"path/filepath"
)

// MinecraftConfig represents the configuration for the Minecraft service.
type MinecraftConfig struct {
	// --- Fields for connecting to an EXISTING server (e.g., via RCON - currently NOT used by StartMinecraftServer) ---
	ServerAddress string `json:"server_address"` // Address of the Minecraft server (if connecting, not starting)
	Port          int    `json:"port"`           // Port of the Minecraft server (e.g., RCON port)
	Username      string `json:"username"`       // Username for authentication (if needed)
	Password      string `json:"password"`       // Password for authentication (if needed)

	// --- Fields for STARTING a NEW local server ---
	ServerRootPath  string `json:"serverRootPath"`  // Path to the Minecraft server root directory
	ServerJarFile   string `json:"serverJarFile"`   // Name of the server JAR file (e.g., "minecraft_server.1.20.2.jar")
	JavaPath        string `json:"javaPath"`        // Path to the java executable (default: "java")
	JvmMemoryArgs   string `json:"jvmMemoryArgs"`   // JVM memory arguments (e.g., "-Xms1024M -Xmx2048M")
	ServerLogFile   string `json:"serverLogFile"`   // Path to the server log file (relative to ServerRootPath or absolute)
	StartupTimeout  int    `json:"startupTimeout"`  // Seconds to wait for server startup (approximate)
	ShutdownCommand string `json:"shutdownCommand"` // Command to gracefully stop the server (e.g., "stop")

	GameVersion    string `json:"game_version"`    // Informational, used in prompts
	CommandTimeout int    `json:"command_timeout"` // Timeout for individual command execution (if applicable in future connection methods)
}

// NewMinecraftConfig creates a new MinecraftConfig with default values.
func NewMinecraftConfig() *MinecraftConfig {
	// Sensible defaults, assuming user wants to start a local server
	// User MUST configure ServerRootPath and ServerJarFile in their config file
	mc := &MinecraftConfig{
		ServerAddress:   "localhost",                   // Default, but not used for local start
		Port:            25565,                         // Default, but not used for local start
		Username:        "MoLingMC",                    // Default, but not used for local start
		Password:        "",                            // Default, but not used for local start
		ServerRootPath:  "./minecraft_server/",         // MUST BE SET BY USER CONFIG
		ServerJarFile:   "minecraft_server.1.20.2.jar", // MUST BE SET BY USER CONFIG
		JavaPath:        "java",
		JvmMemoryArgs:   "-Xms1024M -Xmx1024M",
		ServerLogFile:   "minecraft.log", // Default MC log location relative to root
		StartupTimeout:  5,               // 60 seconds default startup wait
		ShutdownCommand: "stop",
		GameVersion:     "1.20.2", // Default, should reflect jar version ideally
		CommandTimeout:  3,
	}

	return mc
}

// Check validates the configuration.
func (mc *MinecraftConfig) Check() error {
	// Validate fields needed for starting a local server
	if mc.ServerRootPath == "" {
		return fmt.Errorf("minecraft config error: serverRootPath cannot be empty")
	}
	if mc.ServerJarFile == "" {
		return fmt.Errorf("minecraft config error: serverJarFile cannot be empty")
	}
	if mc.JavaPath == "" {
		return fmt.Errorf("minecraft config error: javaPath cannot be empty")
	}
	if mc.ShutdownCommand == "" {
		return fmt.Errorf("minecraft config error: shutdownCommand cannot be empty")
	}
	// Basic check for ServerAddress/Port if they were intended for connection (though not used now)
	// if mc.ServerAddress == "" {
	// 	return fmt.Errorf("server_address cannot be empty")
	// }
	// if mc.Port <= 0 {
	// 	return fmt.Errorf("port must be greater than 0")
	// }

	// Set the absolute path for the server log file
	mc.ServerLogFile = filepath.Join(mc.ServerRootPath, filepath.Base(mc.ServerLogFile))

	return nil
}
