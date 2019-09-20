package server

type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "easymicro/rpc context value " + k.name }

var (
	// ServerContextKey is a context key. It can be used in HTTP
	// handlers with context.WithValue to access the server that
	// started the handler. The associated value will be of
	// type *Server.
	ServerContextKey = &contextKey{"easymicro-server"}

	ReplyContextKey = &contextKey{"easymicro-reply"}

	// LocalAddrContextKey is a context key. It can be used in
	// HTTP handlers with context.WithValue to access the local
	// address the connection arrived on.
	// The associated value will be of type net.Addr.
	LocalAddrContextKey = &contextKey{"local-addr"}
)
