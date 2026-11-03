package cli

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/hmans/means/collection"
)

type tuiWatchError struct{ err error }

type tuiWatcher struct {
	messages chan tea.Msg
	cancel   context.CancelFunc
	stopped  chan struct{}
}

func newTUIWatcher(root string) (*tuiWatcher, error) {
	ctx, cancel := context.WithCancel(context.Background())
	events, err := collection.Watch(ctx, root)
	if err != nil {
		cancel()
		return nil, err
	}
	v := &tuiWatcher{messages: make(chan tea.Msg), cancel: cancel, stopped: make(chan struct{})}
	go func() {
		defer close(v.stopped)
		defer close(v.messages)
		for event := range events {
			if event.Err != nil {
				select {
				case v.messages <- tuiWatchError{event.Err}:
				case <-ctx.Done():
				}
			}
			select {
			case v.messages <- tuiChanged{}:
			case <-ctx.Done():
			}
		}
	}()
	return v, nil
}

func (v *tuiWatcher) close() {
	v.cancel()
	<-v.stopped
}
