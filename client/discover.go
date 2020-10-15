package client

import (
	"github.com/masonchen2014/easymicro/discovery"
)

type AddNodeFunc func(*discovery.ServiceInfo) error

type DelNodeFunc func(*discovery.ServiceInfo) error

