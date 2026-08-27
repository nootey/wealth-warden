package jobs

import (
	"testing"
	"wealth-warden/pkg/config"

	"github.com/stretchr/testify/require"
)

// A typo on either side of the config string is silent: the job just never
// starts on boot.
func TestPeriodicSpecs_ImmediateJobsMapToRunOnStart(t *testing.T) {
	all := []string{
		"asset_history_backfill",
		"balance_backfill",
		"recurring_transactions",
		"asset_price_sync",
	}

	none := periodicSpecs(config.SchedulerConfig{})
	require.Len(t, none, len(all))
	for _, spec := range none {
		require.False(t, spec.runOnStart, "%s should not run on start by default", spec.args.Kind())
	}

	every := periodicSpecs(config.SchedulerConfig{ImmediateJobs: all})
	require.Len(t, every, len(all))
	for _, spec := range every {
		require.True(t, spec.runOnStart, "%s should run on start", spec.args.Kind())
	}
}

func TestPeriodicSpecs_KindsAreUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, spec := range periodicSpecs(config.SchedulerConfig{}) {
		kind := spec.args.Kind()
		require.False(t, seen[kind], "duplicate kind %q", kind)
		seen[kind] = true
	}
}
