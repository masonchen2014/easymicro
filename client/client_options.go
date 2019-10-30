package client

import (
	"github.com/easymicro/log"
	"github.com/juju/ratelimit"
	"github.com/sony/gobreaker"
)

//client set function
type ClientOption func(*Client)

func SetSelector(mode SelectMode) ClientOption {
	return func(c *Client) {
		c.selectMode = mode
		switch mode {
		case RandomSelect:
			c.selector = NewRandomSelector()
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

func SetCircuitBreaker(setting gobreaker.Settings) ClientOption {
	return func(c *Client) {
		c.breaker = gobreaker.NewCircuitBreaker(setting)
	}
}

func SetRateLimiter(tb *ratelimit.Bucket) ClientOption {
	return func(c *Client) {
		c.bucket = tb
	}
}
