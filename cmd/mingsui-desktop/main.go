package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coyoteXujie/mingsui/internal/buildinfo"
	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/desktop"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("mingsui-desktop", flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultClientPath(), "客户端配置文件路径")
	listenAddr := fs.String("listen", "127.0.0.1:18200", "桌面控制台监听地址")
	showVersion := fs.Bool("version", false, "输出版本信息")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Println(buildinfo.String())
		return 0
	}

	logger := log.New(os.Stderr, "mingsui desktop: ", log.LstdFlags)
	app, err := desktop.NewApp(*cfgPath, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化桌面端失败: %v\n", err)
		return 1
	}
	handler, err := desktop.NewHTTPHandler(app)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化桌面端界面失败: %v\n", err)
		return 1
	}

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "监听桌面端地址失败: %v\n", err)
		return 1
	}
	server := &http.Server{Handler: handler}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(os.Stdout, "明隧桌面端已启动: http://%s\n", listener.Addr().String())
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "关闭桌面端失败: %v\n", err)
			return 1
		}
		return 0
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "运行桌面端失败: %v\n", err)
			return 1
		}
		return 0
	}
}
