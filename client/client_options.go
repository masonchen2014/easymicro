package client

import (
	"time"

	"github.com/masonchen2014/easymicro/log"
	"github.com/sony/gobreaker"
)

//client set function
type ClientOption func(*Client)

func SetDialTimeout(t time.Duration) ClientOption {
	return func(c *Client) {
		c.DialTimeout = t
	}
}

func SetSelector(mode SelectMode) ClientOption {
	return func(c *Client) {
		c.selectMode = mode
		switch mode {
		case RoundRobin:
			c.selector = NewRoundRobinSelector()
		default:
			log.Panic("unsupport selector mode")
		}
	}
}

func SetRandomSelector() ClientOption {
	return SetSelector(RandomSelect)
}

func SetRoundRobinSelector() ClientOption {
	return SetSelector(RoundRobin)
}

func SetCircuitBreaker(s *gobreaker.Settings) ClientOption {
	return func(c *Client) {
		c.breakerConfig = s
		//	c.breaker = gobreaker.NewCircuitBreaker(setting)
	}
}

func SetRateLimiter(conf *LimiterConfig) ClientOption {
	return func(c *Client) {
		c.limiterConfig = conf
		//	c.bucket = tb
	}
}
