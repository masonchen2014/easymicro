// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

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

	"github.com/easymicro/log"
	"github.com/easymicro/metadata"
	"github.com/easymicro/protocol"
	"github.com/easymicro/share"
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

var ErrShutdown = errors.New("connection is shut down")
var ErrTimeOut = errors.New("request timeout")
var connected = "200 Connected to Go RPC"

// Call represents an active RPC.
type Call struct {
	ctx           context.Context
	ServicePath   string
	ServiceMethod string      // The name of the service and method to call.
	Args          interface{} // The argument to the function (*struct).
	Reply         interface{} // The reply from the function (*struct).
	Error         error       // After completion, the error status.
	Done          chan *Call  // Strobes when call is complete.
	Metadata      map[string]string
	heartBeat     bool
	compressType  protocol.CompressType
	serializeType protocol.SerializeType
	seq           uint64
	BeforeCalls   []CallOption
	AfterCalls    []CallOption
}

// Client represents an RPC Client.
// There may be multiple outstanding Calls associated
// with a single Client, and a Client may be used by
// multiple goroutines simultaneously.
type Client struct {
	//	codec ClientCodec
	conn net.Conn
	//reqMutex sync.Mutex // protects following
	//request  Request
	heartBeatInterval int64

	mutex    sync.Mutex // protects following
	seq      uint64
	pending  map[uint64]*Call
	lastSend int64
	closing  bool // user has called Close
	shutdown bool // server has told us to stop
	doneChan chan struct{}
}

//create request message from call param
func (client *Client) createRequest(call *Call, seq uint64) (*protocol.Message, error) {
	call.seq = seq

	req := protocol.GetPooledMsg()
	req.ServicePath = call.ServicePath
	req.ServiceMethod = call.ServiceMethod

	if call.heartBeat {
		req.SetHeartbeat(true)
	} else {
		req.SetHeartbeat(false)
	}

	if call.compressType != protocol.None {
		req.SetCompressType(protocol.Gzip)
	}

	req.SetSeq(seq)

	md, b := metadata.FromClientMdContext(call.ctx)
	if b {
		req.Metadata = md
	}

	if call.serializeType != protocol.SerializeNone {
		req.SetSerializeType(call.serializeType)
		codec := share.Codecs[req.SerializeType()]
		if codec == nil {
			err := fmt.Errorf("can not find codec for %d", req.SerializeType())
			return nil, err
		}
		data, err := codec.Encode(call.Args)
		if err != nil {
			return nil, fmt.Errorf("encode ")
		}
		req.Payload = data
	}

	return req, nil
}

func (client *Client) sendHeartBeat() {
	log.Infof("sendHeartBeat at time %d", time.Now().Unix())
	call := new(Call)
	call.ctx = context.Background()
	call.Args = nil
	call.Reply = nil
	call.serializeType = protocol.SerializeNone
	call.Done = make(chan *Call, 1)
	call.heartBeat = true
	client.send(call)
}

func (client *Client) send(call *Call) {

	// Register this call.
	client.mutex.Lock()
	if client.shutdown || client.closing {
		client.mutex.Unlock()
		call.Error = ErrShutdown
		call.done()
		return
	}

	seq := atomic.AddUint64(&client.seq, 1)
	client.pending[seq] = call
	client.lastSend = time.Now().Unix()
	client.mutex.Unlock()

	req, err := client.createRequest(call, seq)
	if err != nil {
		client.mutex.Lock()
		call = client.pending[seq]
		delete(client.pending, seq)
		client.mutex.Unlock()
		if call != nil {
			call.Error = err
			call.done()
		}
		return
	}

	_, err = client.conn.Write(req.Encode())
	if err != nil {
		client.mutex.Lock()
		call = client.pending[seq]
		delete(client.pending, seq)
		client.mutex.Unlock()
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

func (client *Client) readResponse(resp *protocol.Message) (*protocol.Message, error) {
	r := bufio.NewReaderSize(client.conn, 1024)
	//resp := protocol.GetPooledMsg()
	err := resp.Decode(r)
	return resp, err
}

func (client *Client) keepalive() {
	timer := time.NewTimer(time.Duration(client.heartBeatInterval) * time.Second)
	for {
		select {
		case <-timer.C:
			//examine lastSend
			nextKaInterval := client.heartBeatInterval
			client.mutex.Lock()
			lastSend := client.lastSend
			client.mutex.Unlock()

			now := time.Now().Unix()
			if now-lastSend < client.heartBeatInterval {
				nextKaInterval = lastSend + client.heartBeatInterval - now
			} else {
				//here send heart beat
				client.sendHeartBeat()
			}
			timer.Reset(time.Duration(nextKaInterval) * time.Second)
		case <-client.getDoneChan():
			log.Infof("client keepalive goroutine exit")
			break
		}
	}
}

func (client *Client) input() {
	var err error
	resp := protocol.NewMessage()
	for err == nil {
		_, err = client.readResponse(resp)
		if err != nil {
			log.Errorf("input exit for err %v", err)
			break
		}

		seq := resp.Seq()
		client.mutex.Lock()
		call := client.pending[seq]
		delete(client.pending, seq)
		client.mutex.Unlock()

		if resp.IsHeartbeat() {
			call.done()
			continue
		}

		if call != nil {
			codec := share.Codecs[resp.SerializeType()]
			if codec == nil {
				err = fmt.Errorf("can not find codec for %d", resp.SerializeType())
				break
			}
			err = codec.Decode(resp.Payload, call.Reply)
			if err != nil {
				break
			}
			if resp.Metadata != nil {
				call.Metadata = resp.Metadata
			}
			for _, afterCall := range call.AfterCalls {
				afterCall(call)
			}
			call.done()
		}
	}

	// Terminate pending calls.
	client.mutex.Lock()
	client.shutdown = true
	closing := client.closing
	if err == io.EOF {
		if closing {
			err = ErrShutdown
		} else {
			err = io.ErrUnexpectedEOF
		}
	}
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
	client.mutex.Unlock()
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

// DialHTTP connects to an HTTP RPC server at the specified network address
// listening on the default HTTP RPC path.
func DialHTTP(network, address string) (net.Conn, error) {
	return DialHTTPPath(network, address, DefaultRPCPath)
}

// DialHTTPPath connects to an HTTP RPC server
// at the specified network address and path.
func DialHTTPPath(network, address, path string) (net.Conn, error) {
	var err error
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
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
func Dial(network, address string) (net.Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Close calls the underlying codec's Close method. If the connection is already
// shutting down, ErrShutdown is returned.
func (client *Client) Close() error {
	client.mutex.Lock()
	if client.closing {
		client.mutex.Unlock()
		return ErrShutdown
	}
	client.closing = true
	client.mutex.Unlock()
	return client.conn.Close()
}

func (client *Client) getDoneChan() <-chan struct{} {
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
func (client *Client) Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, done chan *Call, options ...BeforeOrAfterCallOption) *Call {
	call := new(Call)
	serviceMethodStrings := strings.Split(serviceMethod, ".")
	if len(serviceMethodStrings) < 2 {
		log.Fatal("invalid service path")
	}
	call.ServicePath = serviceMethodStrings[0]
	call.ServiceMethod = serviceMethodStrings[1]
	call.ctx = ctx
	call.Args = args
	call.Reply = reply
	call.serializeType = protocol.JSON
	for _, opt := range options {
		if opt.after {
			call.AfterCalls = append(call.AfterCalls, opt.option)
		} else {
			opt.option(call)
		}
	}

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
func (client *Client) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, options ...BeforeOrAfterCallOption) error {
	select {
	case call := <-client.Go(ctx, serviceMethod, args, reply, make(chan *Call, 1), options...).Done:
		return call.Error
	case <-ctx.Done():
		return ErrTimeOut
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
func NewClient(network, address string, opts ...ClientOption) (*Client, error) {
	var conn net.Conn
	var err error
	if network == "tcp" {
		conn, err = Dial(network, address)
		if err != nil {
			return nil, err
		}
	} else if network == "http" {
		conn, err = DialHTTP(network, address)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupport network %s", network)
	}

	client := &Client{
		conn:              conn,
		pending:           make(map[uint64]*Call),
		heartBeatInterval: defHeatBeatInterval,
		doneChan:          make(chan struct{}),
	}

	for _, opt := range opts {
		opt(client)
	}
	go client.input()
	go client.keepalive()
	return client, nil
}
