package gateway

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juju/ratelimit"
	"github.com/masonchen2014/easymicro/client"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/protocol"
	"github.com/masonchen2014/easymicro/share"
	"github.com/sony/gobreaker"
)

const (
	// Defaults used by HandleHTTP
	DefaultRPCPath      = "/_goRPC_"
	DefaultDebugPath    = "/debug/rpc"
	defHeatBeatInterval = 5
)

// ServerError represents an error that has been returned from
// the remote side of the RPC connection.
type ServerError string

func (e ServerError) Error() string {
	return string(e)
}

var ErrShutdown = errors.New("connection is shutdown")
var ErrTimeOut = errors.New("request timeout")
var connected = "200 Connected to Go RPC"

// Call represents an active RPC.
type Call struct {
	ctx           context.Context
	ServicePath   string
	ServiceMethod string            // The name of the service and method to call.
	Req           *protocol.Message // The argument to the function (*struct).
	Reply         *protocol.Message // The reply from the function (*struct).
	Error         error             // After completion, the error status.
	Done          chan *Call        // Strobes when call is complete.
	//Metadata      map[string]string
	//heartBeat     bool
	//compressType  protocol.CompressType
	//serializeType protocol.SerializeType
	//seq           uint64
	//	BeforeCalls   []CallOption
	//	AfterCalls    []CallOption
}

// Client represents an RPC Client.
// There may be multiple outstanding Calls associated
// with a single Client, and a Client may be used by
// multiple goroutines simultaneously.
type RPCClient struct {
	network     string
	address     string
	servicePath string

	reconnectTryNums int
	//	codec ClientCodec
	conn net.Conn
	//reqMutex sync.Mutex // protects following
	//request  Request
	heartBeatTryNums  int
	heartBeatTimeout  int
	heartBeatInterval int64

	mutex    sync.Mutex // protects following
	seq      uint64
	pending  map[uint64]*Call
	lastSend int64
	status   client.ConnStatus
	suicide  bool
	//shutdown bool // server has told us to stop
	doneChan    chan struct{}
	DialTimeout time.Duration
	breaker     *gobreaker.CircuitBreaker
	bucket      *ratelimit.Bucket
}

func createHeartBeatReq() *protocol.Message {
	req := protocol.GetPooledMsg()
	req.SetHeartbeat(true)
	req.SetSeq(0)
	return req
}

func (client *RPCClient) sendHeartBeat() error {
	tempDelay := 1
	var err error
	timeout := client.heartBeatTimeout
	if timeout <= 0 {
		timeout = 2
	}

	tryNums := client.heartBeatTryNums
	if tryNums <= 0 {
		tryNums = 3
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	for i := 0; i < tryNums; i++ {
		log.Infof("client sendHeartBeat for %d time at time %d", i+1, time.Now().Unix())
		_, err := client.Call(ctx, "", createHeartBeatReq())
		if err == nil {
			return nil
		} else if err == ErrTimeOut {
			continue
		} else {
			time.Sleep(time.Duration(tempDelay) * time.Second)
			tempDelay = 2 * tempDelay
		}
	}
	return err
}

func (c *RPCClient) reconnect() error {
	c.mutex.Lock()
	if c.suicide {
		c.mutex.Unlock()
		return ErrShutdown
	}
	c.status = client.ConnReconnect
	c.mutex.Unlock()

	reconnectTryNums := int(c.reconnectTryNums)
	if reconnectTryNums <= 0 {
		reconnectTryNums = 6
	}

	var conn net.Conn
	var err error
	tempDelay := 1
	for i := 1; i <= reconnectTryNums; i++ {
		log.Infof("client reconnect for %d time at time %d", i, time.Now().Unix())
		if c.network == "tcp" {
			conn, err = Dial(c.network, c.address, c.DialTimeout)
		} else if c.network == "http" {
			conn, err = DialHTTP(c.network, c.address, c.DialTimeout)
		}
		if err != nil {
			log.Errorf("client reconnect error %+v", err)
			if i < reconnectTryNums {
				time.Sleep(time.Duration(tempDelay) * time.Second)
			}
			tempDelay = 2 * tempDelay
			continue
		} else {
			break
		}
	}

	if err != nil {
		c.mutex.Lock()
		c.status = client.ConnReconnectFail
		c.suicide = false
		c.mutex.Unlock()
		return err
	}

	c.mutex.Lock()
	c.conn = conn
	c.status = client.ConnAvailable
	c.suicide = false
	c.mutex.Unlock()
	log.Infof("client reconnect success")
	return nil
}

func (c *RPCClient) send(call *Call) {

	// Register this call.
	c.mutex.Lock()
	if c.status != client.ConnAvailable {
		c.mutex.Unlock()
		call.Error = ErrShutdown
		call.done()
		return
	}

	seq := atomic.AddUint64(&c.seq, 1)
	c.pending[seq] = call
	c.lastSend = time.Now().Unix()
	rawConn := c.conn
	c.mutex.Unlock()

	call.Req.SetSeq(seq)
	_, err := rawConn.Write(call.Req.Encode())
	if err != nil {
		c.mutex.Lock()
		call = c.pending[seq]
		delete(c.pending, seq)
		c.mutex.Unlock()
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

func (client *RPCClient) readResponse(resp *protocol.Message) (*protocol.Message, error) {
	r := bufio.NewReaderSize(client.conn, 1024)
	err := resp.Decode(r)
	return resp, err
}

func (c *RPCClient) keepalive() {
	timer := time.NewTimer(time.Duration(c.heartBeatInterval) * time.Second)
	for {
		select {
		case <-timer.C:
			//examine lastSend
			nextKaInterval := c.heartBeatInterval
			c.mutex.Lock()
			if c.status != client.ConnAvailable {
				c.mutex.Unlock()
				timer.Reset(time.Duration(nextKaInterval) * time.Second)
				continue
			}
			lastSend := c.lastSend
			c.mutex.Unlock()

			now := time.Now().Unix()
			if now-lastSend < c.heartBeatInterval {
				nextKaInterval = lastSend + c.heartBeatInterval - now
			} else {
				//here send heart beat
				if err := c.sendHeartBeat(); err != nil {
					c.mutex.Lock()
					if c.status == client.ConnAvailable {
						c.suicide = true
						c.status = client.ConnClose
						c.conn.Close()
					}
					c.mutex.Unlock()
				}
			}
			timer.Reset(time.Duration(nextKaInterval) * time.Second)
		case <-c.getDoneChan():
			log.Infof("client keepalive goroutine exit")
			return
		}
	}
}

func (client *RPCClient) Close() {
	client.close(ErrShutdown)
}

func (c *RPCClient) close(err error) {
	// Terminate pending calls.
	c.mutex.Lock()
	if c.status != client.ConnClose {
		c.status = client.ConnClose
		for _, call := range c.pending {
			call.Error = err
			call.done()
		}
		c.conn.Close()
		close(c.doneChan)
	}

	c.mutex.Unlock()
}

func (client *RPCClient) input() {
	var err error
	resp := protocol.NewMessage()

	for {
		_, err = client.readResponse(resp)
		if err != nil {
			log.Errorf("readResponse error %+v", err)
			err = client.reconnect()
			if err != nil {
				break
			} else {
				continue
			}
		}

		seq := resp.Seq()
		client.mutex.Lock()
		call := client.pending[seq]
		delete(client.pending, seq)
		client.mutex.Unlock()

		call.Reply = resp
		if resp.IsHeartbeat() {
			call.done()
			continue
		}

		if call != nil {
			err = checkReplyError(resp)
			if err != nil {
				call.Error = err
				call.done()
				continue
			}
			call.done()
		}
	}

	client.close(err)
	log.Infof("client input goroutine exit")
}

func (call *Call) done() {
	select {
	case call.Done <- call:
		// ok
	default:
		// We don't want to block here. It is the caller's responsibility to make
		// sure the channel has enough buffer space. See comment in Go().
		log.Errorf("rpc: discarding Call reply due to insufficient Done chan capacity")

	}
}

func checkReplyError(reply *protocol.Message) error {
	if reply.MessageStatusType() == protocol.Error {
		err := reply.Metadata[protocol.ServiceError]
		return errors.New(err)
	}
	return nil
}

// DialHTTP connects to an HTTP RPC server at the specified network address
// listening on the default HTTP RPC path.
func DialHTTP(network, address string, timeout time.Duration) (net.Conn, error) {
	return DialHTTPPath(network, address, DefaultRPCPath, timeout)
}

// DialHTTPPath connects to an HTTP RPC server
// at the specified network address and path.
func DialHTTPPath(network, address, path string, timeout time.Duration) (net.Conn, error) {
	var conn net.Conn
	var err error
	if timeout > 0 {
		conn, err = net.DialTimeout(network, address, timeout)
	} else {
		conn, err = net.Dial(network, address)
	}
	io.WriteString(conn, "CONNECT "+path+" HTTP/1.0\n\n")

	// Require successful HTTP response
	// before switching to RPC protocol.
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == connected {
		return conn, nil
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	conn.Close()
	return nil, &net.OpError{
		Op:   "dial-http",
		Net:  network + " " + address,
		Addr: nil,
		Err:  err,
	}
}

// Dial connects to an RPC server at the specified network address.
func Dial(network, address string, timeout time.Duration) (net.Conn, error) {
	var conn net.Conn
	var err error
	if timeout > 0 {
		conn, err = net.DialTimeout(network, address, timeout)
	} else {
		conn, err = net.Dial(network, address)
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (client *RPCClient) getDoneChan() <-chan struct{} {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	if client.doneChan == nil {
		client.doneChan = make(chan struct{})
	}
	return client.doneChan
}

// Go invokes the function asynchronously. It returns the Call structure representing
// the invocation. The done channel will signal when the call is complete by returning
// the same Call object. If done is nil, Go will allocate a new channel.
// If non-nil, done must be buffered or Go will deliberately crash.
func (client *RPCClient) Go(ctx context.Context, serviceMethod string, args *protocol.Message, done chan *Call) *Call {
	call := new(Call)
	call.ServicePath = client.servicePath
	call.ServiceMethod = serviceMethod
	call.ctx = ctx
	call.Req = args

	if done == nil {
		done = make(chan *Call, 10) // buffered.
	} else {
		// If caller passes done != nil, it must arrange that
		// done has enough buffer for the number of simultaneous
		// RPCs that will be using that channel. If the channel
		// is totally unbuffered, it's best not to run at all.
		if cap(done) == 0 {
			log.Panic("rpc: done channel is unbuffered")
		}
	}
	call.Done = done
	client.send(call)
	return call
}

// Call invokes the named function, waits for it to complete, and returns its error status.
func (client *RPCClient) Call(ctx context.Context, serviceMethod string, args *protocol.Message) (*protocol.Message, error) {
	select {
	case call := <-client.Go(ctx, serviceMethod, args, make(chan *Call, 1)).Done:
		return call.Reply, call.Error
	case <-ctx.Done():
		return nil, ErrTimeOut
	}
}

func (c *RPCClient) SetCircuitBreaker(s *gobreaker.Settings) {
	if s != nil {
		c.breaker = gobreaker.NewCircuitBreaker(*s)
	}
}

func (c *RPCClient) SetRateLimiter(conf *LimiterConfig) {
	if conf != nil {
		c.bucket = ratelimit.NewBucketWithQuantum(conf.fillInterval, conf.capacity, conf.quantum)
	}
}

// NewClient returns a new Client to handle requests to the
// set of services at the other end of the connection.
// It adds a buffer to the write side of the connection so
// the header and payload are sent as a unit.
//
// The read and write halves of the connection are serialized independently,
// so no interlocking is required. However each half may be accessed
// concurrently so the implementation of conn should protect against
// concurrent reads or concurrent writes.
func NewRPCClient(network, address, servicePath string, dialTimeout time.Duration) (*RPCClient, error) {
	log.Infof("create rpc client for netword %s address %s service %s", network, address, servicePath)
	var conn net.Conn
	var err error
	if network == "tcp" {
		conn, err = Dial(network, address, dialTimeout)
		if err != nil {
			return nil, err
		}
	} else if network == "http" {
		conn, err = DialHTTP(network, address, dialTimeout)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupport network %s", network)
	}

	client := &RPCClient{
		network:           network,
		address:           address,
		servicePath:       strings.ToLower(servicePath),
		conn:              conn,
		pending:           make(map[uint64]*Call),
		heartBeatInterval: defHeatBeatInterval,
		doneChan:          make(chan struct{}),
		DialTimeout:       dialTimeout,
	}

	go client.input()
	go client.keepalive()
	return client, nil
}

func extractMdFromClientCtx(ctx context.Context) map[string]string {
	md := make(map[string]string)
	comMd, ok := ctx.Value(share.ReqMetaDataKey{}).(map[string]string)
	if ok {
		for k, v := range comMd {
			md[k] = v
		}
	}
	spanMd, ok := ctx.Value(share.SpanMetaDataKey{}).(map[string]string)
	if ok {
		for k, v := range spanMd {
			md[k] = v
		}
	}
	return md
}
