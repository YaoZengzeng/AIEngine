package limiter

type RateLimiter interface {
	DoLimit(model string, tokens int) bool
	UpdateConfig(config *RateLimitConfig) error
}

type RateLimitConfig struct {
	Model           string
	RequestsPerUnit uint32
	Unit            string
}
