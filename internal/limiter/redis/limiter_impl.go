package redis

import (
	"fmt"
	"sync"

	"github.com/mediocregopher/radix/v3"

	"AIEngine/internal/limiter"
)

type rateLimitImpl struct {
	client radix.Client

	mutex   sync.RWMutex
	configs map[string]*limiter.RateLimitConfig
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

	fmt.Printf("Successfully ping redis\n")

	return &rateLimitImpl{
		client:  client,
		configs: make(map[string]*limiter.RateLimitConfig),
	}, nil
}

func (r *rateLimitImpl) DoLimit(model string, tokens int) bool {
	var limit, unit uint32
	r.mutex.RLock()
	if _, ok := r.configs[model]; !ok {
		r.mutex.RUnlock()
		return true
	} else {
		limit = r.configs[model].RequestsPerUnit
		unit = parseUnit(r.configs[model].Unit)
		r.mutex.RUnlock()
	}

	var cnt int
	if err := r.client.Do(radix.Cmd(&cnt, "GET", model)); err != nil {
		fmt.Printf("Failed to get current count of model %s from redis\n", model)
		return true
	}

	if cnt+tokens > int(limit) {
		return false
	}

	if err := r.client.Do(radix.FlatCmd(nil, "INCRBY", model, tokens)); err != nil {
		fmt.Printf("Failed to update count of model %s from redis\n", model)
		return true
	}

	if err := r.client.Do(radix.FlatCmd(nil, "EXPIRE", model, unit)); err != nil {
		fmt.Printf("Failed to set expire time of model %s from redis\n", model)
		return true
	}

	fmt.Printf("Calling redis rate limit\n")
	return true
}

func (r *rateLimitImpl) UpdateConfig(config *limiter.RateLimitConfig) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.configs[config.Model] = config

	return nil
}

func parseUnit(unit string) uint32 {
	var res uint32
	switch unit {
	case "second":
		res = 1
	case "minute":
		res = 60
	case "hour":
		res = 60 * 60
	case "day":
		res = 60 * 60 * 24
	case "month":
		res = 60 * 60 * 24 * 30
	}

	return res
}
