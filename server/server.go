package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/masonchen2014/easymicro/discovery"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/protocol"
	"github.com/masonchen2014/easymicro/share"
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()
var ErrServerClosed = errors.New("http: Server closed")

const (
	// Defaults used by HandleHTTP
	DefaultRPCPath   = "/_goRPC_"
	DefaultDebugPath = "/debug/rpc"
)

type Server struct {
	ln              net.Listener
	serviceMapMu    sync.RWMutex
	serviceMap      map[string]*service
	maxConnIdleTime int64
	mu              sync.RWMutex
	doneChan        chan struct{}
	discovery       discovery.Publisher
	name            string
	advertiseUrl    string
	wg              sync.WaitGroup
	useGateWay      bool
}

var (
	DefaultServer = NewServer()
	contextType   = reflect.TypeOf((*context.Context)(nil)).Elem()
)

type workerJob struct {
	ctx  context.Context
	conn *easyConn
	req  *protocol.Message
}

// NewServer returns a new Server.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		maxConnIdleTime: defaultOption.MaxConnIdleTime,
	}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			log.Panicf("NewServer failed for error %v", err)
		}
	}

	return s
}

//Register a service
func (server *Server) Register(rcvr interface{}) error {
	return server.register(rcvr, "", false)
}

//Register a service using a name
func (server *Server) RegisterName(name string, rcvr interface{}) error {
	return server.register(rcvr, name, true)
}

func (server *Server) register(rcvr interface{}, name string, useName bool) error {
	server.serviceMapMu.Lock()
	defer server.serviceMapMu.Unlock()
	if server.serviceMap == nil {
		server.serviceMap = make(map[string]*service)
	}

	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	server.name = sname
	if useName {
		sname = name
		server.name = name
	}
	if sname == "" {
		s := "rpc.Register: no service name for type " + s.typ.String()
		return errors.New(s)
	}
	if !isExported(sname) && !useName {
		s := "rpc.Register: type " + sname + " is not exported"
		return errors.New(s)
	}
	s.name = sname

	// Install the methods
	s.method = suitableMethods(s.typ, true)

	if len(s.method) == 0 {
		str := ""

		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PtrTo(s.typ), false)
		if len(method) != 0 {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type"
		}
		return errors.New(str)
	}
	server.serviceMap[strings.ToLower(sname)] = s
	return nil
}

// suitableMethods returns suitable Rpc methods of typ, it will report
// error using log if reportErr is true.
func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs four ins: receiver, context.Context, *args, *reply.
		if mtype.NumIn() != 4 {
			if reportErr {
				log.Errorf("rpc.Register: method %q has %d input parameters; needs exactly three\n", mname, mtype.NumIn())
			}
			continue
		}

		ctxType := mtype.In(1)
		if !ctxType.Implements(contextType) {
			if reportErr {
				log.Errorf("rpc.Register: context type of method %q is not implemented: %q\n", mname, ctxType)
			}
		}

		// First arg need not be a pointer.
		argType := mtype.In(2)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				log.Errorf("rpc.Register: argument type of method %q is not exported: %q\n", mname, argType)
			}
			continue
		}
		// Second arg must be a pointer.
		replyType := mtype.In(3)
		if replyType.Kind() != reflect.Ptr {
			if reportErr {
				log.Errorf("rpc.Register: reply type of method %q is not a pointer: %q\n", mname, replyType)
			}
			continue
		}
		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			if reportErr {
				log.Errorf("rpc.Register: reply type of method %q is not exported: %q\n", mname, replyType)
			}
			continue
		}
		// Method needs one out.
		if mtype.NumOut() != 1 {
			if reportErr {
				log.Errorf("rpc.Register: method %q has %d output parameters; needs exactly one\n", mname, mtype.NumOut())
			}
			continue
		}
		// The return type of the method must be error.
		if returnType := mtype.Out(0); returnType != typeOfError {
			if reportErr {
				log.Errorf("rpc.Register: return type of method %q is %q, must be error\n", mname, returnType)
			}
			continue
		}
		methods[strings.ToLower(mname)] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
	}
	return methods
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

func (server *Server) Serve(network, address string) {
	ln, err := net.Listen(network, address)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	if server.discovery != nil {
		if server.name == "" || server.advertiseUrl == "" {
			log.Fatal("no server name or advertise client url to register ")
		}
		//register discovery
		go func() {
			defer func() {
				log.Infof("goroutine discovery exit")
				server.wg.Done()
			}()
			server.wg.Add(1)
			server.discovery.Register(&discovery.ServiceInfo{
				Network: network,
				Name:    strings.ToLower(server.name),
				Addr:    server.advertiseUrl,
			})
		}()
	}
	go server.handleSignal()

	if server.useGateWay {
		ln = server.startGateway(ln)
	}
	server.serve(ln)
}

func (server *Server) Close() {
	server.mu.Lock()
	if server.doneChan != nil {
		close(server.doneChan)
	}
	if server.discovery != nil {
		server.discovery.UnRegister()
	}
	server.mu.Unlock()
	server.wg.Wait()
	log.Infof("server is closed")
	os.Exit(0)
}

func (server *Server) serve(l net.Listener) error {
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		rw, e := l.Accept()
		if e != nil {
			select {
			case <-server.getDoneChan():
				return ErrServerClosed
			default:
			}
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		if tc, ok := rw.(*net.TCPConn); ok {
			tc.SetKeepAlive(true)
			tc.SetKeepAlivePeriod(3 * time.Minute)
		}

		ec := newEasyConn(server, rw)
		go ec.serveConn()
	}
}

func (server *Server) getDoneChan() <-chan struct{} {
	server.mu.Lock()
	defer server.mu.Unlock()

	if server.doneChan == nil {
		server.doneChan = make(chan struct{})
	}
	return server.doneChan
}

func (server *Server) handleRequest(ctx context.Context, req *protocol.Message) (res *protocol.Message, err error) {
	serviceName := req.ServicePath
	methodName := req.ServiceMethod

	//log.Infof("serviceName %s serviceMethod %s for req %+v", serviceName, methodName, req)
	res = req.Clone()
	res.SetMessageType(protocol.Response)

	server.serviceMapMu.RLock()
	service := server.serviceMap[serviceName]
	server.serviceMapMu.RUnlock()
	if service == nil {
		err = errors.New("easymicro: can't find service " + serviceName)
		return handleError(res, err)
	}
	mtype := service.method[methodName]
	if mtype == nil {
		err = errors.New("easymicro: can't find method " + methodName)
		return handleError(res, err)
	}

	var argv = argsReplyPools.Get(mtype.ArgType)
	codec := share.Codecs[req.SerializeType()]
	if codec == nil {
		argsReplyPools.Put(mtype.ArgType, argv)
		err = fmt.Errorf("can not find codec for %d", req.SerializeType())
		return handleError(res, err)
	}

	//	log.Debugf("codec is %+v", codec)
	err = codec.Decode(req.Payload, argv)
	if err != nil {
		argsReplyPools.Put(mtype.ArgType, argv)
		return handleError(res, err)
	}

	replyv := argsReplyPools.Get(mtype.ReplyType)
	ctx = context.WithValue(ctx, ReplyMessageDataKey{}, res)
	if mtype.ArgType.Kind() != reflect.Ptr {
		err = service.call(ctx, mtype, reflect.ValueOf(argv).Elem(), reflect.ValueOf(replyv))
	} else {
		err = service.call(ctx, mtype, reflect.ValueOf(argv), reflect.ValueOf(replyv))
	}

	if err != nil {
		argsReplyPools.Put(mtype.ArgType, argv)
		argsReplyPools.Put(mtype.ReplyType, replyv)
		return handleError(res, err)
	}

	if !req.IsOneway() {
		data, err := codec.Encode(replyv)
		if err != nil {
			argsReplyPools.Put(mtype.ArgType, argv)
			argsReplyPools.Put(mtype.ReplyType, replyv)
			return handleError(res, err)
		}
		res.Payload = data
	}

	argsReplyPools.Put(mtype.ArgType, argv)
	argsReplyPools.Put(mtype.ReplyType, replyv)
	return res, nil
}

func handleError(res *protocol.Message, err error) (*protocol.Message, error) {
	res.SetMessageStatusType(protocol.Error)
	if res.Metadata == nil {
		res.Metadata = make(map[string]string)
	}
	res.Metadata[protocol.ServiceError] = err.Error()
	return res, err
}

// Can connect to RPC service using HTTP CONNECT to rpcPath.
var connected = "200 Connected to Go RPC"

// ServeHTTP implements an http.Handler that answers RPC requests.
func (server *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "405 must CONNECT\n")
		return
	}
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Errorf("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	io.WriteString(conn, "HTTP/1.0 "+connected+"\n\n")

	ec := newEasyConn(server, conn)
	ec.serveConn()
}

// HandleHTTP registers an HTTP handler for RPC messages on rpcPath,
// and a debugging handler on debugPath.
// It is still necessary to invoke http.Serve(), typically in a go statement.
func (server *Server) HandleHTTP(rpcPath, debugPath string) {
	http.Handle(rpcPath, server)
	//	http.Handle(debugPath, debugHTTP{server})
}

func (server *Server) handleSignal() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for s := range c {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
			server.Close()
			return
		default:
			log.Infof("goroutine signal handler capture a signal %+v", s)
		}
	}
}

// HandleHTTP registers an HTTP handler for RPC messages to DefaultServer
// on DefaultRPCPath and a debugging handler on DefaultDebugPath.
// It is still necessary to invoke http.Serve(), typically in a go statement.
func HandleHTTP() {
	DefaultServer.HandleHTTP(DefaultRPCPath, DefaultDebugPath)
}
