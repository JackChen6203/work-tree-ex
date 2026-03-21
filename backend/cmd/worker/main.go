package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/config"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/logging"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.Environment)

	logger.Info("worker started")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	<-sigCh

	logger.Info("worker stopped")
}
