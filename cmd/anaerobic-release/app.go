package main

import (
	"anaerobic-release/internal/api"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/workflow"
	"fmt"
	"net/http"
	"time"
)

func buildServer(cfg config) (*http.Server, error) {
	store, err := storage.Open(cfg.dataFile)
	if err != nil {
		return nil, fmt.Errorf("打开业务存储: %w", err)
	}
	service := workflow.New(store)
	handler := api.New(service).Handler()
	return &http.Server{Addr: cfg.address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 1 << 20}, nil
}
