package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/coyoteXujie/mingsui/internal/buildinfo"
	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/desktop"
)

var errNoDesktopWindowHost = errors.New("未找到可用的桌面窗口宿主；请安装 Google Chrome/Chromium/Edge，或使用 -web 打开浏览器调试界面")

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("mingsui-desktop", flag.ContinueOnError)
	cfgPath := fs.String("config", defaultDesktopConfigPath(), "客户端配置文件路径")
	listenAddr := fs.String("listen", "127.0.0.1:18200", "本机桌面服务监听地址")
	openWindow := fs.Bool("open", true, "启动后打开桌面窗口")
	webMode := fs.Bool("web", false, "用默认浏览器打开调试界面")
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
	windowDone := make(chan error, 1)

	appURL := "http://" + listener.Addr().String()
	fmt.Fprintf(os.Stdout, "明隧桌面端已启动: %s\n", appURL)
	if *openWindow {
		go func() {
			time.Sleep(200 * time.Millisecond)
			if *webMode {
				if err := openURL(appURL); err != nil {
					logger.Printf("打开浏览器调试界面失败: %v", err)
				}
				return
			}
			window, err := openDesktopWindow(appURL)
			if err != nil {
				windowDone <- err
				return
			}
			if window.Done != nil {
				windowDone <- <-window.Done
			}
		}()
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		if err := shutdownServer(server); err != nil {
			fmt.Fprintf(os.Stderr, "关闭桌面端失败: %v\n", err)
			return 1
		}
		return 0
	case err := <-windowDone:
		if err != nil {
			logger.Printf("桌面窗口已退出: %v", err)
		}
		if err := shutdownServer(server); err != nil {
			fmt.Fprintf(os.Stderr, "关闭桌面端失败: %v\n", err)
			return 1
		}
		if err != nil {
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

func shutdownServer(server *http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func defaultDesktopConfigPath() string {
	return config.DefaultClientPath()
}

func openURL(url string) error {
	name, args := browserCommand(url)
	return exec.Command(name, args...).Start()
}

type desktopWindow struct {
	Done <-chan error
}

type desktopWindowSpec struct {
	Name    string
	Args    []string
	Cleanup func()
	Monitor bool
}

func openDesktopWindow(url string) (*desktopWindow, error) {
	spec, ok := desktopWindowCommand(url)
	if !ok {
		return nil, errNoDesktopWindowHost
	}
	cmd := exec.Command(spec.Name, spec.Args...)
	if err := cmd.Start(); err != nil {
		if spec.Cleanup != nil {
			spec.Cleanup()
		}
		return nil, err
	}
	if !spec.Monitor {
		return &desktopWindow{}, nil
	}
	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		if spec.Cleanup != nil {
			spec.Cleanup()
		}
		done <- err
	}()
	return &desktopWindow{Done: done}, nil
}

func newLinuxBrowserSpec(name, url string) (desktopWindowSpec, error) {
	profileDir, err := os.MkdirTemp("", "mingsui-desktop-profile-*")
	if err != nil {
		return desktopWindowSpec{}, err
	}
	return desktopWindowSpec{
		Name: name,
		Args: []string{
			"--app=" + url,
			"--class=MingSui",
			"--name=MingSui",
			"--user-data-dir=" + profileDir,
			"--no-first-run",
			"--no-default-browser-check",
			"--disable-default-apps",
		},
		Cleanup: func() {
			_ = os.RemoveAll(profileDir)
		},
		Monitor: true,
	}, nil
}

func newSimpleBrowserSpec(name, url string) (desktopWindowSpec, error) {
	return desktopWindowSpec{
		Name:    name,
		Args:    []string{"--app=" + url},
		Monitor: false,
	}, nil
}

var lookPath = exec.LookPath

func desktopWindowCommand(url string) (desktopWindowSpec, bool) {
	switch runtime.GOOS {
	case "windows":
		return firstAppBrowserCommand([]string{"msedge.exe", "chrome.exe"}, url, newSimpleBrowserSpec)
	case "darwin":
		if _, err := lookPath("open"); err == nil {
			return desktopWindowSpec{
				Name:    "open",
				Args:    []string{"-b", "com.google.Chrome", "--args", "--app=" + url},
				Monitor: false,
			}, true
		}
	case "linux":
		return firstAppBrowserCommand([]string{"google-chrome", "chromium", "chromium-browser", "microsoft-edge"}, url, newLinuxBrowserSpec)
	}
	return desktopWindowSpec{}, false
}

func firstAppBrowserCommand(candidates []string, url string, build func(string, string) (desktopWindowSpec, error)) (desktopWindowSpec, bool) {
	for _, name := range candidates {
		if _, err := lookPath(name); err == nil {
			spec, err := build(name, url)
			if err != nil {
				continue
			}
			return spec, true
		}
	}
	return desktopWindowSpec{}, false
}

func browserCommand(url string) (string, []string) {
	switch runtime.GOOS {
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		return "open", []string{url}
	default:
		return "xdg-open", []string{url}
	}
}
