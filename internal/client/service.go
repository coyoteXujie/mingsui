package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/protocol"
)

type Service struct {
	cfg     config.ClientConfig
	logger  *log.Logger
	metrics *metricsRecorder
}

type RelayHealth struct {
	Metrics *protocol.Metrics `json:"metrics,omitempty"`
}

func NewService(cfg config.ClientConfig, logger *log.Logger) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Service{cfg: cfg, logger: logger, metrics: &metricsRecorder{}}, nil
}

func (s *Service) Metrics() RuntimeMetrics {
	if s.metrics == nil {
		return RuntimeMetrics{}
	}
	return s.metrics.Snapshot()
}

func (s *Service) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)
	go func() {
		errCh <- s.serveTCP(ctx, "socks5", s.cfg.LocalAddr, s.handleSOCKS)
	}()
	if strings.TrimSpace(s.cfg.HTTPAddr) != "" {
		go func() {
			errCh <- s.serveTCP(ctx, "http", s.cfg.HTTPAddr, s.handleHTTP)
		}()
	}

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}
}

func (s *Service) serveTCP(ctx context.Context, name, addr string, handler func(net.Conn)) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	s.logger.Printf("%s proxy listening on %s", name, listener.Addr())
	if name == "socks5" {
		s.logger.Printf("relay target %s", s.cfg.RelayAddr)
	}

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
		go handler(conn)
	}
}

func (s *Service) openRelay(target string) (net.Conn, error) {
	return s.openRelayContext(context.Background(), target)
}

func (s *Service) openRelayContext(ctx context.Context, target string) (net.Conn, error) {
	conn, err := s.dialRelay(ctx)
	if err != nil {
		return nil, err
	}

	req := protocol.ConnectRequest{
		Version: protocol.Version,
		Token:   s.cfg.Token,
		Network: "tcp",
		Address: target,
	}
	if err := conn.SetDeadline(time.Now().Add(s.cfg.DialTimeout())); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := protocol.WriteJSON(conn, req); err != nil {
		_ = conn.Close()
		return nil, err
	}

	var resp protocol.ConnectResponse
	if err := protocol.ReadJSON(conn, &resp); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if resp.Version != protocol.Version {
		_ = conn.Close()
		return nil, fmt.Errorf("relay protocol version mismatch: %d", resp.Version)
	}
	if !resp.OK {
		_ = conn.Close()
		if strings.TrimSpace(resp.Error) == "" {
			return nil, errors.New("relay rejected connection")
		}
		return nil, errors.New(resp.Error)
	}
	return conn, nil
}

func (s *Service) CheckRelay(ctx context.Context) error {
	_, err := s.CheckRelayStatus(ctx)
	return err
}

func (s *Service) CheckRelayStatus(ctx context.Context) (RelayHealth, error) {
	conn, err := s.dialRelay(ctx)
	if err != nil {
		return RelayHealth{}, err
	}
	defer conn.Close()

	req := protocol.ConnectRequest{
		Version: protocol.Version,
		Command: protocol.CommandHealth,
		Token:   s.cfg.Token,
	}
	if err := conn.SetDeadline(time.Now().Add(s.cfg.DialTimeout())); err != nil {
		return RelayHealth{}, err
	}
	if err := protocol.WriteJSON(conn, req); err != nil {
		return RelayHealth{}, err
	}

	var resp protocol.ConnectResponse
	if err := protocol.ReadJSON(conn, &resp); err != nil {
		return RelayHealth{}, err
	}
	if resp.Version != protocol.Version {
		return RelayHealth{}, fmt.Errorf("relay protocol version mismatch: %d", resp.Version)
	}
	if !resp.OK {
		if strings.TrimSpace(resp.Error) == "" {
			return RelayHealth{}, errors.New("relay health check failed")
		}
		return RelayHealth{}, errors.New(resp.Error)
	}
	return RelayHealth{Metrics: resp.Metrics}, nil
}

func (s *Service) dialRelay(ctx context.Context) (net.Conn, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.cfg.DialTimeout())
		defer cancel()
	}

	dialer := &net.Dialer{Timeout: s.cfg.DialTimeout()}
	conn, err := dialer.DialContext(ctx, "tcp", s.cfg.RelayAddr)
	if err != nil {
		return nil, err
	}
	if !s.cfg.TLS.Enabled {
		return conn, nil
	}

	tlsCfg, err := s.clientTLSConfig()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	tlsConn := tls.Client(conn, tlsCfg)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = tlsConn.Close()
		return nil, err
	}
	return tlsConn, nil
}

func (s *Service) clientTLSConfig() (*tls.Config, error) {
	host, _, err := net.SplitHostPort(s.cfg.RelayAddr)
	if err != nil {
		return nil, err
	}

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         host,
		InsecureSkipVerify: s.cfg.TLS.InsecureSkipVerify,
	}
	if s.cfg.TLS.ServerName != "" {
		tlsCfg.ServerName = s.cfg.TLS.ServerName
	}
	if s.cfg.TLS.CAFile != "" {
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		pemData, err := os.ReadFile(s.cfg.TLS.CAFile)
		if err != nil {
			return nil, err
		}
		if ok := pool.AppendCertsFromPEM(pemData); !ok {
			return nil, fmt.Errorf("no certificate found in %s", s.cfg.TLS.CAFile)
		}
		tlsCfg.RootCAs = pool
	}
	return tlsCfg, nil
}
