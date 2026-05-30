package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func (s *Service) handleHTTP(conn net.Conn) {
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		writeHTTPError(conn, http.StatusBadRequest, "请求格式不正确")
		s.logger.Printf("http request failed from %s: %v", conn.RemoteAddr(), err)
		return
	}
	defer req.Body.Close()
	if !checkHTTPProxyAuth(req, s.cfg.LocalAuth) {
		writeHTTPProxyAuthRequired(conn)
		s.logger.Printf("http proxy auth failed from %s", conn.RemoteAddr())
		return
	}

	if req.Method == http.MethodConnect {
		s.handleHTTPConnect(conn, req, reader)
		return
	}
	s.handleHTTPRequest(conn, req, reader)
}

func (s *Service) handleHTTPConnect(clientConn net.Conn, req *http.Request, reader *bufio.Reader) {
	target, err := normalizeHostPort(req.Host, "443")
	if err != nil {
		writeHTTPError(clientConn, http.StatusBadRequest, "CONNECT 目标地址不正确")
		s.logger.Printf("http connect target invalid host=%q: %v", req.Host, err)
		return
	}

	relayConn, err := s.openRelay(target)
	if err != nil {
		writeHTTPError(clientConn, http.StatusBadGateway, "relay 连接失败")
		s.logger.Printf("http connect relay failed target=%s: %v", target, err)
		return
	}
	defer relayConn.Close()

	if _, err := io.WriteString(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		s.logger.Printf("http connect response failed target=%s: %v", target, err)
		return
	}
	_ = clientConn.SetDeadline(time.Time{})

	s.logger.Printf("http connect target=%s", target)
	s.proxy(bufferedConn{Conn: clientConn, reader: reader}, relayConn)
}

func (s *Service) handleHTTPRequest(clientConn net.Conn, req *http.Request, reader *bufio.Reader) {
	host := req.URL.Host
	if host == "" {
		host = req.Host
	}
	target, err := normalizeHostPort(host, "80")
	if err != nil {
		writeHTTPError(clientConn, http.StatusBadRequest, "HTTP 目标地址不正确")
		s.logger.Printf("http target invalid host=%q: %v", host, err)
		return
	}

	relayConn, err := s.openRelay(target)
	if err != nil {
		writeHTTPError(clientConn, http.StatusBadGateway, "relay 连接失败")
		s.logger.Printf("http relay failed target=%s: %v", target, err)
		return
	}
	defer relayConn.Close()

	req.RequestURI = ""
	req.URL.Scheme = ""
	req.URL.Host = ""
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Proxy-Authenticate")
	req.Header.Del("Proxy-Authorization")

	if err := req.Write(relayConn); err != nil {
		writeHTTPError(clientConn, http.StatusBadGateway, "请求转发失败")
		s.logger.Printf("http write target failed target=%s: %v", target, err)
		return
	}
	_ = clientConn.SetDeadline(time.Time{})

	s.logger.Printf("http request target=%s method=%s", target, req.Method)
	s.proxy(bufferedConn{Conn: clientConn, reader: reader}, relayConn)
}

func normalizeHostPort(host, defaultPort string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("host is empty")
	}
	if h, p, err := net.SplitHostPort(host); err == nil {
		if strings.TrimSpace(h) == "" || strings.TrimSpace(p) == "" {
			return "", fmt.Errorf("host or port is empty")
		}
		return net.JoinHostPort(h, p), nil
	}

	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return net.JoinHostPort(strings.Trim(host, "[]"), defaultPort), nil
	}

	if strings.Count(host, ":") > 1 {
		return net.JoinHostPort(host, defaultPort), nil
	}
	if strings.Contains(host, ":") {
		h, p, err := net.SplitHostPort(host)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(h) == "" || strings.TrimSpace(p) == "" {
			return "", fmt.Errorf("host or port is empty")
		}
		return net.JoinHostPort(h, p), nil
	}
	return net.JoinHostPort(host, defaultPort), nil
}

func writeHTTPError(w io.Writer, status int, body string) {
	statusText := http.StatusText(status)
	if statusText == "" {
		statusText = "Error"
	}
	_, _ = fmt.Fprintf(w, "HTTP/1.1 %d %s\r\nConnection: close\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %d\r\n\r\n%s", status, statusText, len([]byte(body)), body)
}

func writeHTTPProxyAuthRequired(w io.Writer) {
	body := "代理认证失败"
	_, _ = fmt.Fprintf(w, "HTTP/1.1 407 Proxy Authentication Required\r\nConnection: close\r\nProxy-Authenticate: Basic realm=\"MingSui\"\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %d\r\n\r\n%s", len([]byte(body)), body)
}
