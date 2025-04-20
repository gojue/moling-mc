/*
 *
 *  Copyright 2025 CFC4N <cfc4n.cs@gmail.com>. All Rights Reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 *  Repository: https://github.com/gojue/moling-minecraft
 *
 */

package client

import (
	"github.com/rs/zerolog"
	"os"
	"testing"
)

func TestClientManager_ListClient(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	mcpConfig := NewMCPServerConfig("MoLing MineCraft UnitTest Description", "moling_test", "MoLing MCP Server")
	cm := NewManager(logger, mcpConfig)
	// Mock client list
	clientLists["TestClient"] = "/path/to/nonexistent/file"

	cm.ListClient()
	// Check logs or other side effects as needed
}

func TestClientManager_checkExist(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	mcpConfig := NewMCPServerConfig("MoLing MineCraft UnitTest Description", "moling_test", "MoLing MCP Server")
	cm := NewManager(logger, mcpConfig)

	// Test with a non-existent file
	exists := cm.checkExist("/path/to/nonexistent/file")
	if exists {
		t.Errorf("Expected file to not exist")
	}

	// Test with an existing file
	file, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(file.Name())
	t.Logf("Created temp file: %s", file.Name())
	exists = cm.checkExist(file.Name())
	if !exists {
		t.Errorf("Expected file to exist")
	}
}
