package services

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

// MockCommandServer is a mock implementation of CommandServer for testing purposes.
type MockCommandServer struct {
	CommandServer
}

// TestExecuteCommand tests the executeCommand function.
func TestExecuteCommand(t *testing.T) {
	cs := &MockCommandServer{}
	execCmd := "ping -c 4 devel.sankuai.com &> /dev/null | echo 'Hello, World!'"
	// Test a simple command
	output, err := cs.executeCommand(execCmd)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	expectedOutput := "Hello, World!\n"
	if output != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, output)
	}
	t.Logf("Command output: %s", output)
	// Test a command with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sleep", "1")
	err = cmd.Run()
	if err == nil {
		t.Fatalf("Expected timeout error, got nil")
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Errorf("Expected context deadline exceeded error, got %v", ctx.Err())
	}
}
