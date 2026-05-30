package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coyoteXujie/mingsui/internal/client"
	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/relay"
)

func TestSOCKS5ProxyEndToEnd(t *testing.T) {
	targetAddr := startTCPServer(t, func(conn net.Conn) {
		defer conn.Close()
		buf := make([]byte, len("ping"))
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}
		_, _ = conn.Write([]byte("pong"))
	})
	relayAddr := startRelay(t, "secret")
	socksAddr, _ := startClient(t, relayAddr, "secret")

	conn, err := net.DialTimeout("tcp", socksAddr, time.Second)
	if err != nil {
		t.Fatalf("Dial SOCKS5 proxy error = %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

	if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		t.Fatalf("write SOCKS5 hello error = %v", err)
	}
	var methodResp [2]byte
	if _, err := io.ReadFull(conn, methodResp[:]); err != nil {
		t.Fatalf("read SOCKS5 method response error = %v", err)
	}
	if !bytes.Equal(methodResp[:], []byte{0x05, 0x00}) {
		t.Fatalf("method response = %v, want no-auth", methodResp)
	}

	host, port, err := net.SplitHostPort(targetAddr)
	if err != nil {
		t.Fatalf("SplitHostPort() error = %v", err)
	}
	ip := net.ParseIP(host).To4()
	if ip == nil {
		t.Fatalf("target host %q is not IPv4", host)
	}
	portNum, err := net.LookupPort("tcp", port)
	if err != nil {
		t.Fatalf("LookupPort() error = %v", err)
	}

	req := bytes.NewBuffer([]byte{0x05, 0x01, 0x00, 0x01})
	req.Write(ip)
	_ = binary.Write(req, binary.BigEndian, uint16(portNum))
	if _, err := conn.Write(req.Bytes()); err != nil {
		t.Fatalf("write SOCKS5 connect error = %v", err)
	}
	var reply [10]byte
	if _, err := io.ReadFull(conn, reply[:]); err != nil {
		t.Fatalf("read SOCKS5 reply error = %v", err)
	}
	if reply[1] != 0x00 {
		t.Fatalf("SOCKS5 reply = %v, want success", reply)
	}

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write proxied payload error = %v", err)
	}
	resp := make([]byte, len("pong"))
	if _, err := io.ReadFull(conn, resp); err != nil {
		t.Fatalf("read proxied payload error = %v", err)
	}
	if string(resp) != "pong" {
		t.Fatalf("proxied response = %q, want pong", resp)
	}
}

func TestHTTPProxyEndToEnd(t *testing.T) {
	targetAddr := startHTTPServer(t)
	relayAddr := startRelay(t, "secret")
	_, httpProxyAddr := startClient(t, relayAddr, "secret")

	proxyURL := &url.URL{Scheme: "http", Host: httpProxyAddr}
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	defer transport.CloseIdleConnections()
	httpClient := &http.Client{Transport: transport, Timeout: 2 * time.Second}

	resp, err := httpClient.Get("http://" + targetAddr + "/hello")
	if err != nil {
		t.Fatalf("HTTP proxy GET error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(body) != "hello through mingsui" {
		t.Fatalf("body = %q, want proxy response", body)
	}
}

func startRelay(t *testing.T, token string) string {
	t.Helper()

	addr := reserveLocalAddr(t)
	cfg := config.DefaultRelay()
	cfg.ListenAddr = addr
	cfg.Token = token
	cfg.AllowPrivateNetworks = true
	server, err := relay.NewServer(cfg, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ctx)
	}()
	waitForTCP(t, addr, errCh)
	t.Cleanup(func() {
		cancel()
		waitForServerStop(t, errCh)
	})
	return addr
}

func startClient(t *testing.T, relayAddr, token string) (socksAddr, httpAddr string) {
	t.Helper()

	socksAddr = reserveLocalAddr(t)
	httpAddr = reserveLocalAddr(t)
	cfg := config.DefaultClient()
	cfg.LocalAddr = socksAddr
	cfg.HTTPAddr = httpAddr
	cfg.RelayAddr = relayAddr
	cfg.Token = token

	service, err := client.NewService(cfg, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- service.Serve(ctx)
	}()
	waitForTCP(t, socksAddr, errCh)
	waitForTCP(t, httpAddr, errCh)
	t.Cleanup(func() {
		cancel()
		waitForServerStop(t, errCh)
	})
	return socksAddr, httpAddr
}

func startTCPServer(t *testing.T, handle func(net.Conn)) string {
	t.Helper()

	listener := listenLocal(t)
	go acceptLoop(listener, handle)
	t.Cleanup(func() {
		_ = listener.Close()
	})
	return listener.Addr().String()
}

func startHTTPServer(t *testing.T) string {
	t.Helper()

	return startTCPServer(t, func(conn net.Conn) {
		defer conn.Close()
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			return
		}
		defer req.Body.Close()
		if req.URL.Path != "/hello" {
			_, _ = io.WriteString(conn, "HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\n\r\n")
			return
		}
		body := "hello through mingsui"
		_, _ = io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: ")
		_, _ = io.WriteString(conn, strconv.Itoa(len(body)))
		_, _ = io.WriteString(conn, "\r\n\r\n")
		_, _ = io.WriteString(conn, body)
	})
}

func acceptLoop(listener net.Listener, handle func(net.Conn)) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go handle(conn)
	}
}

func reserveLocalAddr(t *testing.T) string {
	t.Helper()

	listener := listenLocal(t)
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return addr
}

func listenLocal(t *testing.T) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if isSocketBlocked(err) {
			t.Skipf("当前环境不允许本机 socket 集成测试: %v", err)
		}
		t.Fatalf("Listen() error = %v", err)
	}
	return listener
}

func waitForTCP(t *testing.T, addr string, errCh <-chan error) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-errCh:
			t.Fatalf("server exited before listen %s: %v", addr, err)
		default:
		}

		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", addr)
}

func waitForServerStop(t *testing.T, errCh <-chan error) {
	t.Helper()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("server stop error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop")
	}
}

func isSocketBlocked(err error) bool {
	if errors.Is(err, net.ErrClosed) {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "operation not permitted")
}
