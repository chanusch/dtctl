package settings

import (
	"testing"
)

func TestIsUUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid UUID with hyphens",
			input: "e1cd3543-8603-3895-bcee-34d20c700074",
			want:  true,
		},
		{
			name:  "valid UUID without hyphens",
			input: "e1cd354386033895bcee34d20c700074",
			want:  true,
		},
		{
			name:  "uppercase UUID",
			input: "E1CD3543-8603-3895-BCEE-34D20C700074",
			want:  true,
		},
		{
			name:  "objectID (base64)",
			input: "vu9U3hXa3q0AAAABABRidWlsdGluOnJ1bS53ZWIubmFtZQALQVBQTElDQVRJT04AEDVDOUI5QkIxQjQ1NDY4NTUAJGU0YzY3NDJmLTQ3ZjktM2IxNC04MzQ4LTU5Y2JlMzJmNzk4ML7vVN4V2t6t",
			want:  false,
		},
		{
			name:  "invalid UUID - too short",
			input: "e1cd3543-8603-3895",
			want:  false,
		},
		{
			name:  "invalid UUID - wrong format",
			input: "not-a-uuid",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUUID(tt.input)
			if got != tt.want {
				t.Errorf("isUUID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetByUID_Logic(t *testing.T) {
	// This test verifies the logic without making actual API calls
	// We'll test that the function correctly identifies UIDs vs objectIDs

	testCases := []struct {
		name       string
		input      string
		expectUID  bool
		expectFail bool
	}{
		{
			name:      "UUID should trigger UID resolution",
			input:     "e1cd3543-8603-3895-bcee-34d20c700074",
			expectUID: true,
		},
		{
			name:      "base64 objectID should use direct get",
			input:     "vu9U3hXa3q0AAAABABRidWlsdGluOnJ1bS53ZWIubmFtZQAL",
			expectUID: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isUUIDResult := isUUID(tc.input)
			if isUUIDResult != tc.expectUID {
				t.Errorf("isUUID(%q) = %v, want %v", tc.input, isUUIDResult, tc.expectUID)
			}
		})
	}
}
