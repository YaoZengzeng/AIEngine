package limiter

type RateLimiter interface {
	DoLimit(model string, tokens int) bool
}
