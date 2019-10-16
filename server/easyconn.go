package server

import (
	"bufio"
	"context"
	"io"
	"net"
	"runtime"
	"time"

	"github.com/easymicro/protocol"

	"github.com/easymicro/log"
)

type easyConn struct {
	server *Server

	maxIdleTime int64

	// rwc is the underlying network connection.
	rwc net.Conn

	// remoteAddr is rwc.RemoteAddr().String().
	remoteAddr string
}

//TODO可以传入ctx进来
func newEasyConn(s *Server, c net.Conn) *easyConn {
	return &easyConn{
		server:      s,
		rwc:         c,
		maxIdleTime: s.maxConnIdleTime,
	}
}

//read request from conn into message
func (ec *easyConn) readRequest(r io.Reader) (*protocol.Message, error) {
	req := protocol.GetPooledMsg()
	err := req.Decode(r)
	return req, err
}

func (ec *easyConn) writeResponse(m *protocol.Message) error {
	if len(m.Payload) > 1024 && m.CompressType() != protocol.None {
		m.SetCompressType(m.CompressType())
	}
	data := m.Encode()
	_, err := ec.rwc.Write(data)
	return err
}

/*func (ec *easyConn) Close() error {
	return nil
}*/

func (ec *easyConn) serveConn(ctx context.Context) {
	defer func() {
		log.Infof("serveConn exit")
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			ss := runtime.Stack(buf, false)
			if ss > size {
				ss = size
			}
			buf = buf[:ss]
		}
		/*	s.mu.Lock()
			delete(s.activeConn, conn)
			s.mu.Unlock()*/

		ec.rwc.Close()

	}()

	r := bufio.NewReaderSize(ec.rwc, 1024)

	for {
		ec.rwc.SetReadDeadline(time.Now().Add(time.Duration(ec.maxIdleTime) * time.Second))
		req, err := ec.readRequest(r)
		if err != nil {
			if err != io.EOF {
				log.Errorf("readRequest error %v", err)
			}
			return
		}

		if req.IsHeartbeat() {
			log.Infof("server receives heartbeat at time %d", time.Now().Unix())
			req.SetMessageType(protocol.Response)
			ec.writeResponse(req)
			protocol.FreeMsg(req)
			continue
		}
		log.Infof("readRequest req %+v", req)
		ec.server.jobChan <- &workerJob{
			ctx:  ctx,
			conn: ec,
			req:  req,
		}
		/*		ctx = metadata.NewClientMdContext(ctx, req.Metadata)
				res, err := ec.server.handleRequest(ctx, req)
				if err != nil {
					log.Errorf("handleRequest error %v", err)
					protocol.FreeMsg(req)
					return
				}
				ec.writeResponse(res)
				protocol.FreeMsg(req)
				protocol.FreeMsg(res)*/
	}

}
