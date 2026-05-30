package relay

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/protocol"
)

type Server struct {
	cfg     config.RelayConfig
	logger  *log.Logger
	metrics *metricsRecorder
}

func NewServer(cfg config.RelayConfig, logger *log.Logger) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Server{cfg: cfg, logger: logger, metrics: &metricsRecorder{}}, nil
}

func (s *Server) Metrics() protocol.Metrics {
	if s.metrics == nil {
		return protocol.Metrics{}
	}
	return s.metrics.Snapshot()
}

func (s *Server) Serve(ctx context.Context) error {
	listener, err := s.listen()
	if err != nil {
		return err
	}
	defer listener.Close()

	s.logger.Printf("relay listening on %s", listener.Addr())

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Temporary() {
				s.logger.Printf("temporary accept error: %v", err)
				continue
			}
			return err
		}
		go s.handle(conn)
	}
}

func (s *Server) listen() (net.Listener, error) {
	if !s.cfg.TLS.Enabled {
		return net.Listen("tcp", s.cfg.ListenAddr)
	}

	cert, err := tls.LoadX509KeyPair(s.cfg.TLS.CertFile, s.cfg.TLS.KeyFile)
	if err != nil {
		return nil, err
	}
	return tls.Listen("tcp", s.cfg.ListenAddr, &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	})
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	var req protocol.ConnectRequest
	if err := protocol.ReadJSON(conn, &req); err != nil {
		s.logger.Printf("bad request from %s: %v", conn.RemoteAddr(), err)
		return
	}

	if err := s.validateRequest(req); err != nil {
		_ = protocol.WriteJSON(conn, protocol.ConnectResponse{
			Version: protocol.Version,
			OK:      false,
			Error:   err.Error(),
		})
		s.logger.Printf("rejected request from %s: %v", conn.RemoteAddr(), err)
		return
	}

	if req.EffectiveCommand() == protocol.CommandHealth {
		metrics := s.Metrics()
		_ = protocol.WriteJSON(conn, protocol.ConnectResponse{
			Version: protocol.Version,
			OK:      true,
			Metrics: &metrics,
		})
		s.logger.Printf("health check ok from %s", conn.RemoteAddr())
		return
	}

	if !s.reserveConnection() {
		_ = protocol.WriteJSON(conn, protocol.ConnectResponse{
			Version: protocol.Version,
			OK:      false,
			Error:   "relay active connection limit reached",
		})
		s.logger.Printf("connection limit reached from %s target=%s", conn.RemoteAddr(), req.Address)
		return
	}
	defer s.releaseConnection()

	target, err := net.DialTimeout(req.Network, req.Address, s.cfg.DialTimeout())
	if err != nil {
		_ = protocol.WriteJSON(conn, protocol.ConnectResponse{
			Version: protocol.Version,
			OK:      false,
			Error:   fmt.Sprintf("dial target: %v", err),
		})
		s.logger.Printf("dial failed target=%s: %v", req.Address, err)
		return
	}
	defer target.Close()
	s.commitConnection()

	if err := protocol.WriteJSON(conn, protocol.ConnectResponse{Version: protocol.Version, OK: true}); err != nil {
		s.logger.Printf("response failed target=%s: %v", req.Address, err)
		return
	}
	_ = conn.SetDeadline(time.Time{})

	s.logger.Printf("relay connected target=%s", req.Address)
	s.proxy(conn, target)
}

func (s *Server) reserveConnection() bool {
	if s.metrics == nil {
		return true
	}
	return s.metrics.ReserveConnection(s.cfg.MaxConnections)
}

func (s *Server) commitConnection() {
	if s.metrics != nil {
		s.metrics.CommitConnection()
	}
}

func (s *Server) releaseConnection() {
	if s.metrics != nil {
		s.metrics.CloseConnection()
	}
}

func (s *Server) validateRequest(req protocol.ConnectRequest) error {
	if req.Version != protocol.Version {
		return fmt.Errorf("unsupported protocol version %d", req.Version)
	}
	if req.Token != s.cfg.Token {
		return errors.New("unauthorized")
	}

	switch req.EffectiveCommand() {
	case protocol.CommandHealth:
		return nil
	case protocol.CommandConnect:
	default:
		return fmt.Errorf("unsupported command %q", req.Command)
	}

	if req.Network != "tcp" {
		return fmt.Errorf("unsupported network %q", req.Network)
	}
	if err := validateTarget(req.Address, s.cfg.AllowPrivateNetworks); err != nil {
		return err
	}
	return nil
}

func validateTarget(address string, allowPrivate bool) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("target must be host:port: %w", err)
	}
	if host == "" {
		return errors.New("target host is required")
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum <= 0 || portNum > 65535 {
		return fmt.Errorf("target port is invalid: %s", port)
	}
	if allowPrivate {
		return nil
	}

	ip := net.ParseIP(host)
	var ips []net.IP
	if ip != nil {
		ips = []net.IP{ip}
	} else {
		ips, err = net.LookupIP(host)
		if err != nil {
			return fmt.Errorf("resolve target host: %w", err)
		}
	}

	for _, resolved := range ips {
		if isPrivateOrLocal(resolved) {
			return fmt.Errorf("target resolves to private or local address: %s", resolved)
		}
	}
	return nil
}

func isPrivateOrLocal(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast()
}
