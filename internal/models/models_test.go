package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStatus(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		result, err := ValidateStatus("", ValidWorkOrderStatuses)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("valid status normalised to uppercase", func(t *testing.T) {
		result, err := ValidateStatus("open", ValidWorkOrderStatuses)
		require.NoError(t, err)
		assert.Equal(t, []string{"OPEN"}, result)
	})

	t.Run("already uppercase status passes", func(t *testing.T) {
		result, err := ValidateStatus("COMPLETED", ValidWorkOrderStatuses)
		require.NoError(t, err)
		assert.Equal(t, []string{"COMPLETED"}, result)
	})

	t.Run("invalid status returns error", func(t *testing.T) {
		_, err := ValidateStatus("INVALID", ValidWorkOrderStatuses)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "INVALID")
		assert.Contains(t, err.Error(), "must be one of")
	})

	t.Run("inspection statuses validated separately", func(t *testing.T) {
		// OPEN is valid for work orders but not inspections
		_, err := ValidateStatus("OPEN", ValidInspectionStatuses)
		require.Error(t, err)

		result, err := ValidateStatus("SCHEDULED", ValidInspectionStatuses)
		require.NoError(t, err)
		assert.Equal(t, []string{"SCHEDULED"}, result)
	})
}

func TestValidateDateRange(t *testing.T) {
	t.Run("nil dates pass", func(t *testing.T) {
		assert.NoError(t, ValidateDateRange(nil, nil))
	})

	t.Run("only after set passes", func(t *testing.T) {
		after := time.Now()
		assert.NoError(t, ValidateDateRange(&after, nil))
	})

	t.Run("only before set passes", func(t *testing.T) {
		before := time.Now()
		assert.NoError(t, ValidateDateRange(nil, &before))
	})

	t.Run("valid range passes", func(t *testing.T) {
		after := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		before := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		assert.NoError(t, ValidateDateRange(&after, &before))
	})

	t.Run("inverted range fails", func(t *testing.T) {
		after := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		before := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		err := ValidateDateRange(&after, &before)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "created_after must be before created_before")
	})

	t.Run("equal dates rejected", func(t *testing.T) {
		same := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		err := ValidateDateRange(&same, &same)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "created_after must be before created_before")
	})
}
