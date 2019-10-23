package client

import "github.com/easymicro/log"

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
