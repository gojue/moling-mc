package services

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// detectMimeType tries to determine the MIME type of a file
func detectMimeType(path string) string {
	// First try by extension
	ext := filepath.Ext(path)
	if ext != "" {
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			return mimeType
		}
	}

	// If that fails, try to read a bit of the file
	file, err := os.Open(path)
	if err != nil {
		return "application/octet-stream" // Default
	}
	defer file.Close()

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return "application/octet-stream" // Default
	}

	// Use http.DetectContentType
	return http.DetectContentType(buffer[:n])
}

// isTextFile determines if a file is likely a text file based on MIME type
func isTextFile(mimeType string) bool {
	return strings.HasPrefix(mimeType, "text/") ||
		mimeType == "application/json" ||
		mimeType == "application/xml" ||
		mimeType == "application/javascript" ||
		mimeType == "application/x-javascript" ||
		strings.Contains(mimeType, "+xml") ||
		strings.Contains(mimeType, "+json")
}

// isImageFile determines if a file is an image based on MIME type
func isImageFile(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

// pathToResourceURI converts a file path to a resource URI
func pathToResourceURI(path string) string {
	return "file://" + path
}
