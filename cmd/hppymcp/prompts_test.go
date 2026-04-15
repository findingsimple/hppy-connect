package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDaysBack(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty defaults to 30", "", "30", false},
		{"valid number", "7", "7", false},
		{"max allowed", "365", "365", false},
		{"over max rejected", "366", "", true},
		{"way over max rejected", "999999", "", true},
		{"zero rejected", "0", "", true},
		{"negative rejected", "-5", "", true},
		{"non-numeric rejected", "abc", "", true},
		{"float rejected", "3.5", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateDaysBack(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
