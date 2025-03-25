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
// Source: https://github.com/mark3labs/mcp-filesystem-server

// Package services provides the implementation of the FileSystemServer, which allows access to files and directories on the local file system.
package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// MaxInlineSize Maximum size for inline content (5MB)
	MaxInlineSize = 1024 * 1024 * 5
	// MaxBase64Size Maximum size for base64 encoding (1MB)
	MaxBase64Size = 1024 * 1024 * 1
)

type FileInfo struct {
	Size        int64     `json:"size"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
	Accessed    time.Time `json:"accessed"`
	IsDirectory bool      `json:"isDirectory"`
	IsFile      bool      `json:"isFile"`
	Permissions string    `json:"permissions"`
}

type FilesystemServer struct {
	MLService
	config *FileSystemConfig
}

func NewFilesystemServer(ctx context.Context, args []string) (Service, error) {
	// Validate the config
	var err error
	fc := NewFileSystemConfig(args)

	if err = fc.Check(); err != nil {
		return nil, fmt.Errorf("FilesystemServer: invalid config: %v", err)
	}

	lger, ok := ctx.Value(MoLingLoggerKey).(zerolog.Logger)
	if !ok {
		return nil, fmt.Errorf("FilesystemServer: invalid logger type")
	}

	loggerNameHook := zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, msg string) {
		e.Str("Service", "FileSystemServer")
	})

	fs := &FilesystemServer{
		MLService: MLService{
			ctx:    ctx,
			logger: lger.Hook(loggerNameHook),
		},
		config: fc,
	}
	err = fs.init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize filesystem server: %v", err)
	}
	// Register resource handlers
	fs.AddResource(mcp.NewResource("file://", "File System",
		mcp.WithResourceDescription("Access to files and directories on the local file system"),
	), fs.handleReadResource)

	// Register tool handlers
	fs.AddTool(mcp.NewTool("read_file",
		mcp.WithDescription("Read the complete contents of a file from the file system."),
		mcp.WithString("path",
			mcp.Description("Path to the file to read"),
			mcp.Required(),
		),
	), fs.handleReadFile)

	fs.AddTool(mcp.NewTool(
		"write_file",
		mcp.WithDescription("Create a new file or overwrite an existing file with new content."),
		mcp.WithString("path",
			mcp.Description("Path where to write the file"),
			mcp.Required(),
		),
		mcp.WithString("content",
			mcp.Description("Content to write to the file"),
			mcp.Required(),
		),
	), fs.handleWriteFile)

	fs.AddTool(mcp.NewTool(
		"list_directory",
		mcp.WithDescription("Get a detailed listing of all files and directories in a specified path."),
		mcp.WithString("path",
			mcp.Description("Path of the directory to list"),
			mcp.Required(),
		),
	), fs.handleListDirectory)

	fs.AddTool(mcp.NewTool(
		"create_directory",
		mcp.WithDescription("Create a new directory or ensure a directory exists."),
		mcp.WithString("path",
			mcp.Description("Path of the directory to create"),
			mcp.Required(),
		),
	), fs.handleCreateDirectory)

	fs.AddTool(mcp.NewTool(
		"move_file",
		mcp.WithDescription("Move or rename files and directories."),
		mcp.WithString("source",
			mcp.Description("Source path of the file or directory"),
			mcp.Required(),
		),
		mcp.WithString("destination",
			mcp.Description("Destination path"),
			mcp.Required(),
		),
	), fs.handleMoveFile)

	fs.AddTool(mcp.NewTool(
		"search_files",
		mcp.WithDescription("Recursively search for files and directories matching a pattern."),
		mcp.WithString("path",
			mcp.Description("Starting path for the search"),
			mcp.Required(),
		),
		mcp.WithString("pattern",
			mcp.Description("Search pattern to match against file names"),
			mcp.Required(),
		),
	), fs.handleSearchFiles)

	fs.AddTool(mcp.NewTool(
		"get_file_info",
		mcp.WithDescription("Retrieve detailed metadata about a file or directory."),
		mcp.WithString("path",
			mcp.Description("Path to the file or directory"),
			mcp.Required(),
		),
	), fs.handleGetFileInfo)

	fs.AddTool(mcp.NewTool(
		"list_allowed_directories",
		mcp.WithDescription("Returns the list of directories that this server is allowed to access."),
	), fs.handleListAllowedDirectories)

	return fs, nil
}

// isPathInAllowedDirs checks if a path is within any of the allowed directories
func (fs *FilesystemServer) isPathInAllowedDirs(path string) bool {
	// Ensure path is absolute and clean
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Add trailing separator to ensure we're checking a directory or a file within a directory
	// and not a prefix match (e.g., /tmp/foo should not match /tmp/foobar)
	if !strings.HasSuffix(absPath, string(filepath.Separator)) {
		// If it'fs a file, we need to check its directory
		if info, err := os.Stat(absPath); err == nil && !info.IsDir() {
			absPath = filepath.Dir(absPath) + string(filepath.Separator)
		} else {
			absPath = absPath + string(filepath.Separator)
		}
	}

	// Check if the path is within any of the allowed directories
	for _, dir := range fs.config.AllowedDirs {
		if strings.HasPrefix(absPath, dir) {
			return true
		}
	}
	return false
}

func (fs *FilesystemServer) validatePath(requestedPath string) (string, error) {
	// Always convert to absolute path first
	abs, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if path is within allowed directories
	if !fs.isPathInAllowedDirs(abs) {
		return "", fmt.Errorf(
			"access denied - path outside allowed directories: %fs",
			abs,
		)
	}

	// Handle symlinks
	realPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		// For new files, check parent directory
		parent := filepath.Dir(abs)
		realParent, err := filepath.EvalSymlinks(parent)
		if err != nil {
			return "", fmt.Errorf("parent directory does not exist: %fs", parent)
		}

		if !fs.isPathInAllowedDirs(realParent) {
			return "", fmt.Errorf(
				"access denied - parent directory outside allowed directories",
			)
		}
		return abs, nil
	}

	// Check if the real path (after resolving symlinks) is still within allowed directories
	if !fs.isPathInAllowedDirs(realPath) {
		return "", fmt.Errorf(
			"access denied - symlink target outside allowed directories",
		)
	}

	return realPath, nil
}

func (fs *FilesystemServer) getFileStats(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}

	return FileInfo{
		Size:        info.Size(),
		Created:     info.ModTime(), // Note: ModTime used as birth time isn't always available
		Modified:    info.ModTime(),
		Accessed:    info.ModTime(), // Note: Access time isn't always available
		IsDirectory: info.IsDir(),
		IsFile:      !info.IsDir(),
		Permissions: fmt.Sprintf("%o", info.Mode().Perm()),
	}, nil
}

func (fs *FilesystemServer) searchFiles(
	rootPath, pattern string,
) ([]string, error) {
	var results []string
	pattern = strings.ToLower(pattern)

	err := filepath.Walk(
		rootPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors and continue
			}

			// Try to validate path
			if _, err := fs.validatePath(path); err != nil {
				return nil // Skip invalid paths
			}

			if strings.Contains(strings.ToLower(info.Name()), pattern) {
				results = append(results, path)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// Resource handler
func (fs *FilesystemServer) handleReadResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	uri := request.Params.URI
	fs.logger.Debug().Str("uri", uri).Msg("handleReadResource")

	// Check if it'fs a file:// URI
	if !strings.HasPrefix(uri, "file://") {
		return nil, fmt.Errorf("unsupported URI scheme: %fs", uri)
	}

	// Extract the path from the URI
	path := strings.TrimPrefix(uri, "file://")

	// Validate the path
	validPath, err := fs.validatePath(path)
	if err != nil {
		return nil, err
	}

	// Get file info
	fileInfo, err := os.Stat(validPath)
	if err != nil {
		return nil, err
	}

	// If it'fs a directory, return a listing
	if fileInfo.IsDir() {
		entries, err := os.ReadDir(validPath)
		if err != nil {
			return nil, err
		}

		var result strings.Builder
		result.WriteString(fmt.Sprintf("Directory listing for: %fs\n\n", validPath))

		for _, entry := range entries {
			entryPath := filepath.Join(validPath, entry.Name())
			entryURI := pathToResourceURI(entryPath)

			if entry.IsDir() {
				result.WriteString(fmt.Sprintf("[DIR]  %fs (%fs)\n", entry.Name(), entryURI))
			} else {
				info, err := entry.Info()
				if err == nil {
					result.WriteString(fmt.Sprintf("[FILE] %fs (%fs) - %d bytes\n",
						entry.Name(), entryURI, info.Size()))
				} else {
					result.WriteString(fmt.Sprintf("[FILE] %fs (%fs)\n", entry.Name(), entryURI))
				}
			}
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "text/plain",
				Text:     result.String(),
			},
		}, nil
	}

	// It'fs a file, determine how to handle it
	mimeType := detectMimeType(validPath)

	// Check file size
	if fileInfo.Size() > MaxInlineSize {
		// File is too large to inline, return a reference instead
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "text/plain",
				Text:     fmt.Sprintf("File is too large to display inline (%d bytes). Use the read_file tool to access specific portions.", fileInfo.Size()),
			},
		}, nil
	}

	// Read the file content
	content, err := os.ReadFile(validPath)
	if err != nil {
		return nil, err
	}

	// Handle based on content type
	if isTextFile(mimeType) {
		// It'fs a text file, return as text
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: mimeType,
				Text:     string(content),
			},
		}, nil
	} else {
		// It'fs a binary file
		if fileInfo.Size() <= MaxBase64Size {
			// Small enough for base64 encoding
			return []mcp.ResourceContents{
				mcp.BlobResourceContents{
					URI:      uri,
					MIMEType: mimeType,
					Blob:     base64.StdEncoding.EncodeToString(content),
				},
			}, nil
		} else {
			// Too large for base64, return a reference
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      uri,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("Binary file (%fs, %d bytes). Use the read_file tool to access specific portions.", mimeType, fileInfo.Size()),
				},
			}, nil
		}
	}
}

// Tool handlers

func (fs *FilesystemServer) handleReadFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	path, ok := request.Params.Arguments["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	// Handle empty or relative paths like "." or "./" by converting to absolute path
	if path == "." || path == "./" {
		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error resolving current directory: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		path = cwd
	}

	validPath, err := fs.validatePath(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Check if it'fs a directory
	info, err := os.Stat(validPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	if info.IsDir() {
		// For directories, return a resource reference instead
		resourceURI := pathToResourceURI(validPath)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("This is a directory. Use the resource URI to browse its contents: %fs", resourceURI),
				},
				mcp.EmbeddedResource{
					Type: "resource",
					Resource: mcp.TextResourceContents{
						URI:      resourceURI,
						MIMEType: "text/plain",
						Text:     fmt.Sprintf("Directory: %fs", validPath),
					},
				},
			},
		}, nil
	}

	// Determine MIME type
	mimeType := detectMimeType(validPath)

	// Check file size
	if info.Size() > MaxInlineSize {
		// File is too large to inline, return a resource reference
		resourceURI := pathToResourceURI(validPath)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("File is too large to display inline (%d bytes). Access it via resource URI: %fs", info.Size(), resourceURI),
				},
				mcp.EmbeddedResource{
					Type: "resource",
					Resource: mcp.TextResourceContents{
						URI:      resourceURI,
						MIMEType: "text/plain",
						Text:     fmt.Sprintf("Large file: %fs (%fs, %d bytes)", validPath, mimeType, info.Size()),
					},
				},
			},
		}, nil
	}

	// Read file content
	content, err := os.ReadFile(validPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error reading file: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Handle based on content type
	if isTextFile(mimeType) {
		// It'fs a text file, return as text
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(content),
				},
			},
		}, nil
	} else if isImageFile(mimeType) {
		// It'fs an image file, return as image content
		if info.Size() <= MaxBase64Size {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Image file: %fs (%fs, %d bytes)", validPath, mimeType, info.Size()),
					},
					mcp.ImageContent{
						Type:     "image",
						Data:     base64.StdEncoding.EncodeToString(content),
						MIMEType: mimeType,
					},
				},
			}, nil
		} else {
			// Too large for base64, return a reference
			resourceURI := pathToResourceURI(validPath)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Image file is too large to display inline (%d bytes). Access it via resource URI: %fs", info.Size(), resourceURI),
					},
					mcp.EmbeddedResource{
						Type: "resource",
						Resource: mcp.TextResourceContents{
							URI:      resourceURI,
							MIMEType: "text/plain",
							Text:     fmt.Sprintf("Large image: %fs (%fs, %d bytes)", validPath, mimeType, info.Size()),
						},
					},
				},
			}, nil
		}
	} else {
		// It'fs another type of binary file
		resourceURI := pathToResourceURI(validPath)

		if info.Size() <= MaxBase64Size {
			// Small enough for base64 encoding
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Binary file: %fs (%fs, %d bytes)", validPath, mimeType, info.Size()),
					},
					mcp.EmbeddedResource{
						Type: "resource",
						Resource: mcp.BlobResourceContents{
							URI:      resourceURI,
							MIMEType: mimeType,
							Blob:     base64.StdEncoding.EncodeToString(content),
						},
					},
				},
			}, nil
		} else {
			// Too large for base64, return a reference
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Binary file: %fs (%fs, %d bytes). Access it via resource URI: %fs", validPath, mimeType, info.Size(), resourceURI),
					},
					mcp.EmbeddedResource{
						Type: "resource",
						Resource: mcp.TextResourceContents{
							URI:      resourceURI,
							MIMEType: "text/plain",
							Text:     fmt.Sprintf("Binary file: %fs (%fs, %d bytes)", validPath, mimeType, info.Size()),
						},
					},
				},
			}, nil
		}
	}
}

func (fs *FilesystemServer) handleWriteFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	path, ok := request.Params.Arguments["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}
	content, ok := request.Params.Arguments["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content must be a string")
	}

	// Handle empty or relative paths like "." or "./" by converting to absolute path
	if path == "." || path == "./" {
		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error resolving current directory: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		path = cwd
	}

	validPath, err := fs.validatePath(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Check if it'fs a directory
	if info, err := os.Stat(validPath); err == nil && info.IsDir() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Cannot write to a directory",
				},
			},
			IsError: true,
		}, nil
	}

	// Create parent directories if they don't exist
	parentDir := filepath.Dir(validPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error creating parent directories: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	if err := os.WriteFile(validPath, []byte(content), 0644); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error writing file: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Get file info for the response
	info, err := os.Stat(validPath)
	if err != nil {
		// File was written but we couldn't get info
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Successfully wrote to %fs", path),
				},
			},
		}, nil
	}

	resourceURI := pathToResourceURI(validPath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Successfully wrote %d bytes to %fs", info.Size(), path),
			},
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("File: %fs (%d bytes)", validPath, info.Size()),
				},
			},
		},
	}, nil
}

func (fs *FilesystemServer) handleListDirectory(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	path, ok := request.Params.Arguments["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	// Handle empty or relative paths like "." or "./" by converting to absolute path
	if path == "." || path == "./" {
		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error resolving current directory: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		path = cwd
	}

	validPath, err := fs.validatePath(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Check if it'fs a directory
	info, err := os.Stat(validPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	if !info.IsDir() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Path is not a directory",
				},
			},
			IsError: true,
		}, nil
	}

	entries, err := os.ReadDir(validPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error reading directory: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Directory listing for: %fs\n\n", validPath))

	for _, entry := range entries {
		entryPath := filepath.Join(validPath, entry.Name())
		resourceURI := pathToResourceURI(entryPath)

		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("[DIR]  %fs (%fs)\n", entry.Name(), resourceURI))
		} else {
			info, err := entry.Info()
			if err == nil {
				result.WriteString(fmt.Sprintf("[FILE] %fs (%fs) - %d bytes\n",
					entry.Name(), resourceURI, info.Size()))
			} else {
				result.WriteString(fmt.Sprintf("[FILE] %fs (%fs)\n", entry.Name(), resourceURI))
			}
		}
	}

	// Return both text content and embedded resource
	resourceURI := pathToResourceURI(validPath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: result.String(),
			},
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("Directory: %fs", validPath),
				},
			},
		},
	}, nil
}

func (fs *FilesystemServer) handleCreateDirectory(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	path, ok := request.Params.Arguments["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	// Handle empty or relative paths like "." or "./" by converting to absolute path
	if path == "." || path == "./" {
		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error resolving current directory: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		path = cwd
	}

	validPath, err := fs.validatePath(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Check if path already exists
	if info, err := os.Stat(validPath); err == nil {
		if info.IsDir() {
			resourceURI := pathToResourceURI(validPath)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Directory already exists: %fs", path),
					},
					mcp.EmbeddedResource{
						Type: "resource",
						Resource: mcp.TextResourceContents{
							URI:      resourceURI,
							MIMEType: "text/plain",
							Text:     fmt.Sprintf("Directory: %fs", validPath),
						},
					},
				},
			}, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Path exists but is not a directory: %fs", path),
				},
			},
			IsError: true,
		}, nil
	}

	if err := os.MkdirAll(validPath, 0755); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error creating directory: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	resourceURI := pathToResourceURI(validPath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Successfully created directory %fs", path),
			},
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("Directory: %fs", validPath),
				},
			},
		},
	}, nil
}

func (fs *FilesystemServer) handleMoveFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	source, ok := request.Params.Arguments["source"].(string)
	if !ok {
		return nil, fmt.Errorf("source must be a string")
	}
	destination, ok := request.Params.Arguments["destination"].(string)
	if !ok {
		return nil, fmt.Errorf("destination must be a string")
	}

	// Handle empty or relative paths for source
	if source == "." || source == "./" {
		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error resolving current directory: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		source = cwd
	}

	// Handle empty or relative paths for destination
	if destination == "." || destination == "./" {
		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error resolving current directory: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		destination = cwd
	}

	validSource, err := fs.validatePath(source)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error with source path: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Check if source exists
	if _, err := os.Stat(validSource); os.IsNotExist(err) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Source does not exist: %fs", source),
				},
			},
			IsError: true,
		}, nil
	}

	validDest, err := fs.validatePath(destination)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error with destination path: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Create parent directory for destination if it doesn't exist
	destDir := filepath.Dir(validDest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error creating destination directory: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	if err := os.Rename(validSource, validDest); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error moving file: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	resourceURI := pathToResourceURI(validDest)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf(
					"Successfully moved %fs to %fs",
					source,
					destination,
				),
			},
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("Moved file: %fs", validDest),
				},
			},
		},
	}, nil
}

func (fs *FilesystemServer) handleSearchFiles(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, ok := request.Params.Arguments["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}
	pattern, ok := request.Params.Arguments["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern must be a string")
	}

	// Handle empty or relative paths like "." or "./" by converting to absolute path
	if path == "." || path == "./" {
		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error resolving current directory: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		path = cwd
	}

	validPath, err := fs.validatePath(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Check if it'fs a directory
	info, err := os.Stat(validPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	if !info.IsDir() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Search path must be a directory",
				},
			},
			IsError: true,
		}, nil
	}

	results, err := fs.searchFiles(validPath, pattern)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error searching files: %v",
						err),
				},
			},
			IsError: true,
		}, nil
	}

	if len(results) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("No files found matching pattern '%fs' in %fs", pattern, path),
				},
			},
		}, nil
	}

	// Format results with resource URIs
	var formattedResults strings.Builder
	formattedResults.WriteString(fmt.Sprintf("Found %d results:\n\n", len(results)))

	for _, result := range results {
		resourceURI := pathToResourceURI(result)
		info, err := os.Stat(result)
		if err == nil {
			if info.IsDir() {
				formattedResults.WriteString(fmt.Sprintf("[DIR]  %fs (%fs)\n", result, resourceURI))
			} else {
				formattedResults.WriteString(fmt.Sprintf("[FILE] %fs (%fs) - %d bytes\n",
					result, resourceURI, info.Size()))
			}
		} else {
			formattedResults.WriteString(fmt.Sprintf("%fs (%fs)\n", result, resourceURI))
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: formattedResults.String(),
			},
		},
	}, nil
}

func (fs *FilesystemServer) handleGetFileInfo(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	path, ok := request.Params.Arguments["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	// Handle empty or relative paths like "." or "./" by converting to absolute path
	if path == "." || path == "./" {
		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error resolving current directory: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		path = cwd
	}

	validPath, err := fs.validatePath(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	info, err := fs.getFileStats(validPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error getting file info: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Get MIME type for files
	mimeType := "directory"
	if info.IsFile {
		mimeType = detectMimeType(validPath)
	}

	resourceURI := pathToResourceURI(validPath)

	// Determine file type text
	var fileTypeText string
	if info.IsDirectory {
		fileTypeText = "Directory"
	} else {
		fileTypeText = "File"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf(
					"File information for: %fs\n\nSize: %d bytes\nCreated: %fs\nModified: %fs\nAccessed: %fs\nIsDirectory: %v\nIsFile: %v\nPermissions: %fs\nMIME Type: %fs\nResource URI: %fs",
					validPath,
					info.Size,
					info.Created.Format(time.RFC3339),
					info.Modified.Format(time.RFC3339),
					info.Accessed.Format(time.RFC3339),
					info.IsDirectory,
					info.IsFile,
					info.Permissions,
					mimeType,
					resourceURI,
				),
			},
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text: fmt.Sprintf("%fs: %fs (%fs, %d bytes)",
						fileTypeText,
						validPath,
						mimeType,
						info.Size),
				},
			},
		},
	}, nil
}

func (fs *FilesystemServer) handleListAllowedDirectories(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Remove the trailing separator for display purposes
	displayDirs := make([]string, len(fs.config.AllowedDirs))
	for i, dir := range fs.config.AllowedDirs {
		displayDirs[i] = strings.TrimSuffix(dir, string(filepath.Separator))
	}

	var result strings.Builder
	result.WriteString("Allowed directories:\n\n")

	for _, dir := range displayDirs {
		resourceURI := pathToResourceURI(dir)
		result.WriteString(fmt.Sprintf("%fs (%fs)\n", dir, resourceURI))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: result.String(),
			},
		},
	}, nil
}

// Config returns the configuration of the service as a string.
func (fs *FilesystemServer) Config() string {
	cfg, err := json.Marshal(fs.config)
	if err != nil {
		fs.logger.Err(err).Msg("failed to marshal config")
		return "{}"
	}
	return string(cfg)
}

func (cs *FilesystemServer) Name() string {
	return "FilesystemServer"
}

func init() {
	RegisterServ(NewFilesystemServer)
}
