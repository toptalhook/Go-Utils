package utils

import (
	"context"
	"time"
)

// ThrottleCfg Throttle's configuration
type ThrottleCfg struct {
	Max, NPerSec int
}

// Throttle current limitor
type Throttle struct {
	*ThrottleCfg
	token      struct{}
	tokensChan chan struct{}
	stopChan   chan struct{}
}

// NewThrottleWithCtx create new Throttle
func NewThrottleWithCtx(ctx context.Context, cfg *ThrottleCfg) *Throttle {
	t := &Throttle{
		ThrottleCfg: cfg,
		token:       struct{}{},
		stopChan:    make(chan struct{}),
	}
	t.tokensChan = make(chan struct{}, t.Max)
	t.runWithCtx(ctx)
	return t
}

// Allow check whether is allowed
func (t *Throttle) Allow() bool {
	select {
	case <-t.tokensChan:
		return true
	default:
		return false
	}
}

// runWithCtx start throttle with context
func (t *Throttle) runWithCtx(ctx context.Context) {
	go func() {
		defer Logger.Info("throttle exit")
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			for i := 0; i < t.NPerSec; i++ {
				select {
				case <-ctx.Done():
					return
				case <-t.stopChan:
					return
				case t.tokensChan <- t.token:
				default:
				}
			}

			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			case <-t.stopChan:
				return
			}
		}
	}()
}

// Close stop throttle
func (t *Throttle) Close() {
	close(t.stopChan)
}

// Stop stop throttle
func (t *Throttle) Stop() {
	t.Close()
}
