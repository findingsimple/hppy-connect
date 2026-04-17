package cmd

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// These tests cover the rating-score validation in `inspections rate-item`
// (P0-4 from the 2026-04-17 review pass). They drive the RunE through a fresh
// cobra wrapper so they exercise the pre-API validation paths only.

func runRateItem(t *testing.T, score float64) error {
	t.Helper()
	wrapper := &cobra.Command{Use: "rate-item-test", RunE: inspectionsRateItemCmd.RunE}
	wrapper.Flags().String("id", "insp-1", "")
	wrapper.Flags().String("section", "Kitchen", "")
	wrapper.Flags().String("item", "Sink", "")
	wrapper.Flags().String("rating-key", "condition", "")
	wrapper.Flags().Float64("rating-score", 0, "")
	wrapper.Flags().String("rating-value", "", "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})

	wrapper.SetArgs([]string{fmt.Sprintf("--rating-score=%g", score)})
	return wrapper.Execute()
}

func TestInspectionsRateItem_RejectsNaNScore(t *testing.T) {
	err := runRateItem(t, math.NaN())
	assert.ErrorContains(t, err, "rating score")
}

func TestInspectionsRateItem_RejectsPositiveInfScore(t *testing.T) {
	err := runRateItem(t, math.Inf(1))
	assert.ErrorContains(t, err, "rating score")
}

func TestInspectionsRateItem_RejectsNegativeInfScore(t *testing.T) {
	err := runRateItem(t, math.Inf(-1))
	assert.ErrorContains(t, err, "rating score")
}

func TestInspectionsRateItem_RejectsNegativeScore(t *testing.T) {
	err := runRateItem(t, -1.0)
	assert.ErrorContains(t, err, "rating score")
}
