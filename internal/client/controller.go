package client

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
)

type RuntimeStatus struct {
	Running   bool      `json:"running"`
	LocalAddr string    `json:"local_addr"`
	HTTPAddr  string    `json:"http_addr,omitempty"`
	RelayAddr string    `json:"relay_addr"`
	StartedAt time.Time `json:"started_at,omitempty"`
	LastError string    `json:"last_error,omitempty"`
}

type Controller struct {
	mu        sync.Mutex
	cfg       config.ClientConfig
	serve     func(context.Context) error
	cancel    context.CancelFunc
	done      chan error
	startedAt time.Time
	lastError string
}

func NewController(cfg config.ClientConfig, logger *log.Logger) (*Controller, error) {
	service, err := NewService(cfg, logger)
	if err != nil {
		return nil, err
	}
	return &Controller{
		cfg:   cfg,
		serve: service.Serve,
	}, nil
}

func (c *Controller) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		return errors.New("客户端已经在运行")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	c.cancel = cancel
	c.done = done
	c.startedAt = time.Now()
	c.lastError = ""
	serve := c.serve

	go func() {
		err := serve(runCtx)
		c.mu.Lock()
		if err != nil && runCtx.Err() == nil {
			c.lastError = err.Error()
		}
		c.cancel = nil
		c.done = nil
		c.mu.Unlock()
		done <- err
	}()

	return nil
}

func (c *Controller) Stop(ctx context.Context) error {
	c.mu.Lock()
	cancel := c.cancel
	done := c.done
	c.mu.Unlock()

	if cancel == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cancel()

	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Controller) Status() RuntimeStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	return RuntimeStatus{
		Running:   c.cancel != nil,
		LocalAddr: c.cfg.LocalAddr,
		HTTPAddr:  c.cfg.HTTPAddr,
		RelayAddr: c.cfg.RelayAddr,
		StartedAt: c.startedAt,
		LastError: c.lastError,
	}
}
