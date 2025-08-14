// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package blockcontroller

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// TerminalExpect provides expect-like functionality for interactive terminals
type TerminalExpect struct {
	blockId      string
	outputBuffer bytes.Buffer
	promptRegex  *regexp.Regexp
}

// NewTerminalExpect creates a new TerminalExpect instance
func NewTerminalExpect(blockId string) *TerminalExpect {
	// Common prompt patterns:
	// - Standard shell prompts: $ or #
	// - Claude CLI prompts: > or claude>
	// - Application-specific prompts ending with >
	promptPattern := `(?m)(?:[$#>]|claude>|\s>\s*)\s*$`
	
	return &TerminalExpect{
		blockId:     blockId,
		promptRegex: regexp.MustCompile(promptPattern),
	}
}

// WaitForPrompt waits for a command prompt to appear in the terminal output
func (te *TerminalExpect) WaitForPrompt(timeout time.Duration) (bool, string) {
	deadline := time.Now().Add(timeout)
	checkInterval := 50 * time.Millisecond
	
	for time.Now().Before(deadline) {
		output := te.outputBuffer.String()
		
		// Remove ANSI escape sequences for cleaner matching
		cleanOutput := stripANSI(output)
		
		// Check if we have a prompt
		if te.promptRegex.MatchString(cleanOutput) {
			return true, cleanOutput
		}
		
		time.Sleep(checkInterval)
	}
	
	return false, te.outputBuffer.String()
}

// AddOutput adds terminal output to the buffer for analysis
func (te *TerminalExpect) AddOutput(data []byte) {
	te.outputBuffer.Write(data)
	
	// Keep only the last 4KB to avoid memory issues
	if te.outputBuffer.Len() > 4096 {
		buf := te.outputBuffer.Bytes()
		te.outputBuffer.Reset()
		te.outputBuffer.Write(buf[len(buf)-2048:])
	}
}

// Clear clears the output buffer
func (te *TerminalExpect) Clear() {
	te.outputBuffer.Reset()
}

// SendCommandWithExpect sends a command and waits for a prompt
func SendCommandWithExpect(bc *BlockController, command string, waitForPrompt bool) error {
	if bc == nil {
		return fmt.Errorf("block controller is nil")
	}
	
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("command cannot be empty")
	}
	
	// Ensure command ends with \r for proper terminal handling
	if !strings.HasSuffix(command, "\r") && !strings.HasSuffix(command, "\n") {
		command = command + "\r"
	}
	
	// Replace any standalone \n with \r
	command = strings.ReplaceAll(command, "\n", "\r")
	
	// Create input union
	inputUnion := &BlockInputUnion{
		InputData: []byte(command),
	}
	
	// If we should wait for prompt, set up expect handling
	if waitForPrompt {
		// TODO: Integrate with terminal output monitoring
		// This would require hooking into the PTY output stream
		// For now, just add a small delay to let the terminal settle
		time.Sleep(100 * time.Millisecond)
	}
	
	// Send the input
	return bc.SendInput(inputUnion)
}

// stripANSI removes ANSI escape sequences from a string
func stripANSI(str string) string {
	// Pattern to match ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07|\x1b[PX^_].*?\x1b\\|\x1b\[[\?!][0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(str, "")
}

// IsInteractiveCLI checks if the command is likely an interactive CLI
func IsInteractiveCLI(command string) bool {
	// List of known interactive CLIs that need special handling
	interactiveCLIs := []string{
		"claude",
		"python",
		"node",
		"irb",
		"pry",
		"ghci",
		"sqlite3",
		"mysql",
		"psql",
		"redis-cli",
		"mongo",
	}
	
	// Extract the base command (first word)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}
	
	baseCmd := parts[0]
	// Remove path if present
	if idx := strings.LastIndex(baseCmd, "/"); idx >= 0 {
		baseCmd = baseCmd[idx+1:]
	}
	
	// Check if it's a known interactive CLI
	for _, cli := range interactiveCLIs {
		if strings.HasPrefix(baseCmd, cli) {
			return true
		}
	}
	
	return false
}