package backup

import (
	"reflect"
	"testing"
)

func TestParseBackupItems(t *testing.T) {
	tests := []struct {
		input       string
		expected    []BackupItem
		expectError bool
	}{
		{
			input: "path1:dest1,path2:dest2",
			expected: []BackupItem{
				{Source: "path1", Destination: "dest1"},
				{Source: "path2", Destination: "dest2"},
			},
		},
		{
			input:       "invalid_format",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseBackupItems(tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("unexpected error status: %v", err)
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("unexpected result: got %v, want %v", result, tt.expected)
			}
		})
	}
}
