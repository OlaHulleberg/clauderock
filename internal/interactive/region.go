package interactive

import (
	"fmt"

	"github.com/OlaHulleberg/clauderock/internal/awsutil"
)

// SelectRegionWithSearch provides an interactive region selector with real-time filtering
func SelectRegionWithSearch(currentRegion string) (string, error) {
	allRegions := awsutil.GetRegions()

	// Convert regions to SelectOptions
	options := make([]SelectOption, len(allRegions))
	for i, r := range allRegions {
		options[i] = SelectOption{
			ID:      r.ID,
			Display: fmt.Sprintf("%s - %s", r.ID, r.Name),
		}
	}

	return InteractiveSelect(
		"Filter AWS Regions",
		"Type to filter regions...",
		options,
		currentRegion,
	)
}
