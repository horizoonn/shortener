package middleware

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/horizoonn/shortener/internal/httpapi/request"
	"github.com/horizoonn/shortener/internal/httpapi/response"
	"golang.org/x/time/rate"
)

const unknownIPKey = "unknown"

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64
}

type IPRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*clientLimiter
	rate     rate.Limit
	burst    int
	stop     chan struct{}
}

func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	rl := &IPRateLimiter{
		limiters: make(map[string]*clientLimiter),
		rate:     rate.Limit(rps),
		burst:    burst,
		stop:     make(chan struct{}),
	}

	go rl.cleanupLoop(10*time.Minute, 5*time.Minute)

	return rl
}

func (rl *IPRateLimiter) Close() {
	close(rl.stop)
}

func (rl *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.RLock()
	v, exists := rl.limiters[ip]
	rl.mu.RUnlock()

	if exists {
		v.lastSeen.Store(time.Now().UnixNano())
		return v.limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists = rl.limiters[ip]
	if exists {
		v.lastSeen.Store(time.Now().UnixNano())
		return v.limiter
	}

	limiter := rate.NewLimiter(rl.rate, rl.burst)
	cl := &clientLimiter{limiter: limiter}
	cl.lastSeen.Store(time.Now().UnixNano())
	rl.limiters[ip] = cl

	return limiter
}

func (rl *IPRateLimiter) cleanupLoop(maxInactive, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	maxInactiveNano := maxInactive.Nanoseconds()

	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixNano()

			rl.mu.Lock()
			for ip, v := range rl.limiters {
				if now-v.lastSeen.Load() > maxInactiveNano {
					delete(rl.limiters, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

var ignoredRateLimitPaths = map[string]struct{}{
	"/healthz":           {},
	"/readyz":            {},
	"/metrics":           {},
	"/docs":              {},
	"/docs/openapi.yaml": {},
}

func RateLimit(rl *IPRateLimiter, ipResolver *request.IPResolver) Middleware {
	if rl == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := ignoredRateLimitPaths[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			ip := ipResolver.Resolve(r)
			if ip == "" {
				ip = unknownIPKey
			}

			limiter := rl.getLimiter(ip)
			if !limiter.Allow() {
				response.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded", "too_many_requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
