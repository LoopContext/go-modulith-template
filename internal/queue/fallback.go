//go:build !asynq

// Package queue provides an in-memory fallback task queue and a real Asynq queue implementation.
package queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Global in-memory channel for the fallback queue.
var (
	taskChan = make(chan *Task, 1000)
	chanMu   sync.Mutex
)

// Client is the in-memory fallback task queue client.
type Client struct{}

// NewClient creates a new fallback Client.
func NewClient(_ string, _ string, _ int) *Client {
	return &Client{}
}

// Enqueue sends a task to the in-memory queue.
func (c *Client) Enqueue(task *Task) error {
	chanMu.Lock()
	defer chanMu.Unlock()

	select {
	case taskChan <- task:
		slog.Debug("Enqueued task in memory fallback queue", "type", task.Type())
		return nil
	default:
		return fmt.Errorf("in-memory task queue is full")
	}
}

// Close is a no-op for the fallback Client.
func (c *Client) Close() error {
	return nil
}

// Server is the in-memory fallback task queue server.
type Server struct {
	handlers map[string]func(context.Context, *Task) error
	stop     chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
}

// NewServer creates a new fallback Server.
func NewServer(_ string, _ string, _ int, _ int) *Server {
	return &Server{
		handlers: make(map[string]func(context.Context, *Task) error),
		stop:     make(chan struct{}),
	}
}

// HandleFunc registers a handler function for the given task type.
func (s *Server) HandleFunc(pattern string, handler func(context.Context, *Task) error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers[pattern] = handler
}

// Start runs the server worker loop in a background goroutine.
func (s *Server) Start() error {
	slog.Info("Starting in-memory fallback queue server...")

	s.wg.Go(func() {
		for {
			select {
			case <-s.stop:
				return
			case task := <-taskChan:
				s.mu.RLock()
				handler, ok := s.handlers[task.Type()]
				s.mu.RUnlock()

				if !ok {
					slog.Warn("No handler registered for task type", "type", task.Type())
					continue
				}

				s.wg.Add(1)
				go func(t *Task, h func(context.Context, *Task) error) {
					defer s.wg.Done()

					ctx := context.Background()

					if err := h(ctx, t); err != nil {
						slog.Error("Task execution failed in fallback queue", "type", t.Type(), "error", err)
					}
				}(task, handler)
			}
		}
	})

	return nil
}

// Shutdown stops the worker loop and waits for all active tasks to complete.
func (s *Server) Shutdown() {
	slog.Info("Shutting down in-memory fallback queue server...")
	close(s.stop)
	s.wg.Wait()
	slog.Info("In-memory fallback queue server shut down successfully.")
}
