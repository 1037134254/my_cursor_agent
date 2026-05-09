package llm

import (
	"context"
	"sync"
	"time"
)

// rpmLimiter 简单滑动窗口：每分钟最多 max 次请求（RPM）。
type rpmLimiter struct {
	mu   sync.Mutex
	hits []time.Time
	max  int
}

func newRPMLimiter(maxPerMinute int) *rpmLimiter {
	if maxPerMinute <= 0 {
		return nil
	}
	return &rpmLimiter{max: maxPerMinute}
}

func (l *rpmLimiter) Wait(ctx context.Context) error {
	if l == nil {
		return nil
	}
	cutoff := time.Now().Add(-time.Minute)
	for {
		l.mu.Lock()
		var kept []time.Time
		for _, t := range l.hits {
			if t.After(cutoff) {
				kept = append(kept, t)
			}
		}
		l.hits = kept
		if len(l.hits) < l.max {
			l.hits = append(l.hits, time.Now())
			l.mu.Unlock()
			return nil
		}
		var sleep time.Duration
		if len(l.hits) > 0 {
			sleep = time.Until(l.hits[0].Add(time.Minute))
			if sleep < 0 {
				sleep = time.Millisecond * 50
			}
		} else {
			sleep = time.Millisecond * 50
		}
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleep):
		}
		cutoff = time.Now().Add(-time.Minute)
	}
}
