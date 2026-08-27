package joblog

import (
	"context"
	"errors"
	"fmt"
)

// StoppedEarly reports a run that did not reach every item, so a partial run is
// never recorded as a clean one.
func StoppedEarly(ctx context.Context, done, total int, unit string) error {
	cause := ctx.Err()
	if cause == nil {
		cause = errors.New("workers exited early")
	}
	return fmt.Errorf("stopped after %d of %d %s: %w", done, total, unit, cause)
}
