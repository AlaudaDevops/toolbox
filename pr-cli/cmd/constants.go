/*
Copyright 2025 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import "fmt"

// CommandType represents the type of parsed command
type CommandType int

const (
	// SingleCommand represents a regular single command
	SingleCommand CommandType = iota
	// MultiCommand represents multiple commands in a single comment
	MultiCommand
	// BuiltInCommand represents a built-in system command
	BuiltInCommand
)

// String returns the string representation of CommandType
func (ct CommandType) String() string {
	switch ct {
	case SingleCommand:
		return "SingleCommand"
	case MultiCommand:
		return "MultiCommand"
	case BuiltInCommand:
		return "BuiltInCommand"
	default:
		return fmt.Sprintf("Unknown(%d)", int(ct))
	}
}

// Error message constants for consistency
const (
	ErrUnsupportedCommandType = "unsupported command type: %s"
	ErrUnknownCommandType     = "unknown command type: %s"
)

// ParsedCommand encapsulates a parsed command with its type and data
type ParsedCommand struct {
	Type            CommandType
	Command         string
	Args            []string
	CommandLines    []string // Used only for MultiCommand type (processed)
	RawCommandLines []string // Used only for MultiCommand type (original, unprocessed)
}
