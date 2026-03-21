package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/config"
)

type Server struct {
	cfg    config.Config
	logger *slog.Logger
	engine *gin.Engine
}

func New(cfg config.Config, logger *slog.Logger) *Server {
	if cfg.Environment == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(corsMiddleware(cfg.CORS.AllowedOrigins))
	engine.Use(requestIDMiddleware())
	engine.Use(accessLogMiddleware(logger))
	engine.Use(recoveryMiddleware(logger))

	registerRoutes(engine)

	return &Server{
		cfg:    cfg,
		logger: logger,
		engine: engine,
	}
}

func (s *Server) Run(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", s.cfg.HTTP.Host, s.cfg.HTTP.Port),
		Handler:      s.engine,
		ReadTimeout:  s.cfg.HTTP.ReadTimeout,
		WriteTimeout: s.cfg.HTTP.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("api server starting", "addr", httpServer.Addr)
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done():
		s.logger.Info("api server stopping", "reason", "context canceled")
	case <-sigCh:
		s.logger.Info("api server stopping", "reason", "signal received")
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.HTTP.ShutdownTimeout)
	defer cancel()

	return httpServer.Shutdown(shutdownCtx)
}
