// toolkit/windowsservice/programwindows.go
//go:build windows

package windowsservice

import (
	"context"

	"github.com/dalemusser/waffle/app"
	"github.com/kardianos/service"
)

// Program wraps waffle/app.Run so it can be driven by the Windows
// Service Control Manager (SCM).
//
// C = app-specific config type
// D = app-specific DB/deps bundle type
type Program[C any, D any] struct {
	Hooks  app.Hooks[C, D]
	cancel func()
}

// Start is called by the SCM when the service is started.
func (p *Program[C, D]) Start(s service.Service) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	// Run WAFFLE app in a goroutine so Start can return quickly.
	go func() {
		_ = app.Run(ctx, p.Hooks)
	}()

	return nil
}

// Stop is called by the SCM when the service is stopped.
func (p *Program[C, D]) Stop(s service.Service) error {
	if p.cancel != nil {
		p.cancel() // triggers graceful shutdown inside waffle/app + waffle/server
	}
	return nil
}
