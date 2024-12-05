package redis

import (
	"fmt"

	"github.com/mediocregopher/radix/v3"

	"AIEngine/internal/limiter"
)

type rateLimitImpl struct {
	client radix.Client
}

func NewRateLimiter() (limiter.RateLimiter, error) {
	poolSize := 10
	address := "redis:6379"
	client, err := radix.NewPool("tcp", address, poolSize)
	if err != nil {
		return nil, fmt.Errorf("failed to construct redis pool: %v", err)
	}

	var pingResponse string
	if err := client.Do(radix.Cmd(&pingResponse, "PING")); err != nil {
		return nil, fmt.Errorf("ping redis failed: %v", err)
	}

	if pingResponse != "PONG" {
		return nil, fmt.Errorf("response of PING is not PONG")
	}

	return &rateLimitImpl{
		client: client,
	}, nil
}

func (*rateLimitImpl) DoLimit(model string, tokens int) bool {
	return true
}
