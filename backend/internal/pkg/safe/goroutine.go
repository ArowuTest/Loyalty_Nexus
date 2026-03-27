// Package safe provides helpers for safe goroutine execution with panic recovery.
package safe

import (
	"context"
	"log"
	"runtime/debug"
)

// Go launches fn in a goroutine with a deferred recover() so a panic in the
// goroutine is logged rather than crashing the whole server process.
func Go(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC RECOVERED] %v\n%s", r, debug.Stack())
			}
		}()
		fn()
	}()
}

// GoCtx is the same as Go but passes a context so the caller can propagate
// cancellation / deadline into the goroutine.
func GoCtx(ctx context.Context, fn func(ctx context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC RECOVERED] %v\n%s", r, debug.Stack())
			}
		}()
		fn(ctx)
	}()
}
