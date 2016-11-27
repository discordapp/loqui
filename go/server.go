package loqui

import (
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ServerHandler responses to Requests on Conn.
//
// Provided context should be used to request the request and writing
// to the same context will reply to the rest. Not writing to the context
// will result in an empty reply when the function returns.
type ServerHandler interface {
	ServeRequest(ctx RequestContext)
}

// ServerConfig fields are optional except SupportedEncodings.
type ServerConfig struct {
	PingInterval       time.Duration
	SupportedEncodings []string
	Concurrency        int
}

// Server implements http.Handler allowing a specific HTTP route to
// to be upgraded to Loqui.
type Server struct {
	mu      sync.Mutex
	conns   map[*Conn]bool
	handler ServerHandler
	config  ServerConfig
}

// NewServer allocates and returns a new Server.
func NewServer(handler ServerHandler, config ServerConfig) *Server {
	if config.PingInterval == 0 {
		config.PingInterval = time.Millisecond * 30000
	}

	if config.Concurrency == 0 {
		config.Concurrency = 20
	}

	return &Server{
		conns:   make(map[*Conn]bool),
		handler: handler,
		config:  config,
	}
}

func (s *Server) upgrade(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" || !strings.EqualFold(req.Header.Get("Upgrade"), "loqui") {
		w.WriteHeader(http.StatusUpgradeRequired)
		return
	}

	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return
	}
	io.WriteString(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: loqui\r\nConnection: Upgrade\r\n\r\n")

	s.serveConn(conn)
}

func (s *Server) serveConn(conn net.Conn) (err error) {
	c := NewConn(conn, conn, conn, false)
	c.pingInterval = s.config.PingInterval
	c.handler = s.handler
	c.supportedEncodings = s.config.SupportedEncodings

	s.mu.Lock()
	s.conns[c] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.conns, c)
		s.mu.Unlock()
	}()

	return c.Serve(s.config.Concurrency)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		s.handler.ServeRequest(newHTTPRequestContext(w, req))
	} else {
		s.upgrade(w, req)
	}
}
