package limiter

type RateLimiter interface {
	DoLimit(hostname string, model string, tokens int) bool
	UpdateConfig(config *RateLimitConfig) error
}

type RateLimitConfig struct {
	Hostname        string
	Model           string
	RequestsPerUnit uint32
	Unit            string
}
