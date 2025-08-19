package cmd

import (
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		comment     string
		wantCommand string
		wantArgs    []string
		wantErr     bool
	}{
		{
			name:        "single user assignment",
			comment:     "/assign user1",
			wantCommand: "assign",
			wantArgs:    []string{"user1"},
			wantErr:     false,
		},
		{
			name:        "multiple users assignment",
			comment:     "/assign user1 user2 user3",
			wantCommand: "assign",
			wantArgs:    []string{"user1", "user2", "user3"},
			wantErr:     false,
		},
		{
			name:        "multiple users assignment with @ symbol",
			comment:     "/assign @user1 @user2 @user3",
			wantCommand: "assign",
			wantArgs:    []string{"@user1", "@user2", "@user3"},
			wantErr:     false,
		},
		{
			name:        "help command",
			comment:     "/help",
			wantCommand: "help",
			wantArgs:    nil,
			wantErr:     false,
		},
		{
			name:        "lgtm command",
			comment:     "/lgtm",
			wantCommand: "lgtm",
			wantArgs:    nil,
			wantErr:     false,
		},
		{
			name:        "invalid command format",
			comment:     "assign user1",
			wantCommand: "",
			wantArgs:    nil,
			wantErr:     true,
		},
		{
			name:        "unsupported command",
			comment:     "/invalid",
			wantCommand: "",
			wantArgs:    nil,
			wantErr:     true,
		},
	}

	// Create a PROption instance for testing
	prOption := NewPROption()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCommand, gotArgs, err := prOption.parseCommand(tt.comment)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotCommand != tt.wantCommand {
				t.Errorf("parseCommand() gotCommand = %v, want %v", gotCommand, tt.wantCommand)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("parseCommand() gotArgs = %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestParseStringFields(t *testing.T) {
	tests := []struct {
		name                string
		prNumStr            string
		lgtmPermissionsStr  string
		wantPRNum           int
		wantLGTMPermissions []string
		wantErr             bool
	}{
		{
			name:                "valid PR number and permissions",
			prNumStr:            "123",
			lgtmPermissionsStr:  "admin,write",
			wantPRNum:           123,
			wantLGTMPermissions: []string{"admin", "write"},
			wantErr:             false,
		},
		{
			name:                "permissions with spaces",
			prNumStr:            "456",
			lgtmPermissionsStr:  "admin, write, read",
			wantPRNum:           456,
			wantLGTMPermissions: []string{"admin", "write", "read"},
			wantErr:             false,
		},
		{
			name:                "single permission",
			prNumStr:            "789",
			lgtmPermissionsStr:  "admin",
			wantPRNum:           789,
			wantLGTMPermissions: []string{"admin"},
			wantErr:             false,
		},
		{
			name:                "invalid PR number",
			prNumStr:            "invalid",
			lgtmPermissionsStr:  "admin,write",
			wantPRNum:           0,
			wantLGTMPermissions: []string{"admin", "write"},
			wantErr:             true,
		},
		{
			name:                "empty strings",
			prNumStr:            "",
			lgtmPermissionsStr:  "",
			wantPRNum:           0,
			wantLGTMPermissions: []string{"admin", "write"}, // Should keep default
			wantErr:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prOption := NewPROption()
			prOption.prNumStr = tt.prNumStr
			prOption.lgtmPermissionsStr = tt.lgtmPermissionsStr

			err := prOption.parseStringFields()
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStringFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if prOption.Config.PRNum != tt.wantPRNum {
					t.Errorf("parseStringFields() PRNum = %v, want %v", prOption.Config.PRNum, tt.wantPRNum)
				}
				if !reflect.DeepEqual(prOption.Config.LGTMPermissions, tt.wantLGTMPermissions) {
					t.Errorf("parseStringFields() LGTMPermissions = %v, want %v", prOption.Config.LGTMPermissions, tt.wantLGTMPermissions)
				}
			}
		})
	}
}
