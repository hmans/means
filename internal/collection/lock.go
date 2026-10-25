package collection

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/flock"
)

// tryLock waits up to five seconds for an exclusive cache lock. A timed-out wait
// reports busy; cancellation of ctx itself is returned as a lock error.
func tryLock(ctx context.Context, filename, name, busy string) (*flock.Flock, error) {
	lock := flock.New(filename)
	wait, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := lock.TryLockContext(wait, 25*time.Millisecond); err != nil {
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			return nil, errors.New(busy)
		}
		return nil, fmt.Errorf("lock %s: %w", name, err)
	}
	return lock, nil
}
