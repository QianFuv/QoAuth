package main

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

type drainingHandler struct {
	handler http.Handler
	mutex   sync.Mutex
	closed  bool
	active  sync.WaitGroup
}

func (handler *drainingHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	handler.mutex.Lock()
	if handler.closed {
		handler.mutex.Unlock()
		http.Error(response, "server is stopping", http.StatusServiceUnavailable)
		return
	}
	handler.active.Add(1)
	handler.mutex.Unlock()
	defer handler.active.Done()
	handler.handler.ServeHTTP(response, request)
}

// serveHTTP drains handlers before its caller closes SQLite and releases the lock.
func serveHTTP(ctx context.Context, server *http.Server) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	handler := &drainingHandler{handler: server.Handler}
	server.Handler = handler
	drained := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		err := server.Shutdown(shutdown)
		if err != nil {
			server.Close()
		}
		handler.mutex.Lock()
		handler.closed = true
		handler.mutex.Unlock()
		handler.active.Wait()
		drained <- err
	}()
	serveErr := server.ListenAndServe()
	cancel()
	shutdownErr := <-drained
	if serveErr != nil && serveErr != http.ErrServerClosed {
		return errors.New("HTTP server stopped unexpectedly")
	}
	if shutdownErr != nil {
		return errors.New("HTTP shutdown timed out; connections closed and handlers drained")
	}
	return nil
}
