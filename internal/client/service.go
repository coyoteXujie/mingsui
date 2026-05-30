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
	"sync"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/protocol"
)

type Service struct {
	cfg    config.ClientConfig
	logger *log.Logger
}

func NewService(cfg config.ClientConfig, logger *log.Logger) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Service{cfg: cfg, logger: logger}, nil
}

func (s *Service) Serve(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.cfg.LocalAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	s.logger.Printf("client listening on socks5://%s", listener.Addr())
	s.logger.Printf("relay target %s", s.cfg.RelayAddr)

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
		go s.handleSOCKS(conn)
	}
}

func (s *Service) openRelay(target string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: s.cfg.DialTimeout()}

	var (
		conn net.Conn
		err  error
	)
	if s.cfg.TLS.Enabled {
		tlsCfg, err := s.clientTLSConfig()
		if err != nil {
			return nil, err
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", s.cfg.RelayAddr, tlsCfg)
	} else {
		conn, err = dialer.Dial("tcp", s.cfg.RelayAddr)
	}
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

func proxyBidirectional(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = copyAndCloseWrite(b, a)
	}()
	go func() {
		defer wg.Done()
		_, _ = copyAndCloseWrite(a, b)
	}()

	wg.Wait()
}
