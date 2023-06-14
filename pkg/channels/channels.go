package channels

import (
	"context"
	"errors"
	"fmt"
)

// Send sends a slice of messages to a channel.
//
// It blocks until every message is sent or the given context is closed.
func Send[T any](ctx context.Context, to chan<- T, msgs ...T) error {
	for i, msg := range msgs {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context closed sending message [%d/%d]: [%w]", i+1, len(msgs), ctx.Err())
		case to <- msg:
		}
	}

	return nil
}

// ForEach passes each message received from a channel to a function.
//
// It blocks until the given context is closed or the function returns an error.
func ForEach[T any](ctx context.Context, in <-chan T, do func(T) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-in:
			if !ok {
				// Channel is closed, msg is zero value.
				// Bail out.
				return nil
			}
			if err := do(msg); err != nil {
				return err
			}
		}
	}
}

// Forward sends messages received from one channel to another.
//
// It blocks until the given context is closed.
func Forward[T any](ctx context.Context, from <-chan T, to chan<- T) error {
	return ForEach(ctx, from, func(msg T) error {
		return Send(ctx, to, msg)
	})
}

// NilOrCanceled returns true IFF an error is either nil or wraps context.Canceled.
func NilOrCanceled(err error) bool {
	return err == nil || errors.Is(err, context.Canceled)
}
