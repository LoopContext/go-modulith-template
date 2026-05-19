//go:build asynq

package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

// Client is the Asynq-backed task queue client.
type Client struct {
	client *asynq.Client
}

// NewClient creates a new Asynq Client.
func NewClient(addr string, password string, db int) *Client {
	return &Client{
		client: asynq.NewClient(asynq.RedisClientOpt{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

// Enqueue sends a task to the Valkey-backed Asynq queue.
func (c *Client) Enqueue(task *Task) error {
	asynqTask := asynq.NewTask(task.Type(), task.Payload())
	_, err := c.client.Enqueue(asynqTask)
	if err != nil {
		return fmt.Errorf("asynq enqueue failed: %w", err)
	}
	slog.Debug("Enqueued task in Asynq", "type", task.Type())
	return nil
}

// Close closes the Asynq client Valkey connection.
func (c *Client) Close() error {
	return c.client.Close()
}

// Server is the Asynq-backed task queue server.
type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

// NewServer creates a new Asynq Server.
func NewServer(addr string, password string, db int, concurrency int) *Server {
	return &Server{
		server: asynq.NewServer(
			asynq.RedisClientOpt{Addr: addr, Password: password, DB: db},
			asynq.Config{Concurrency: concurrency},
		),
		mux: asynq.NewServeMux(),
	}
}

// HandleFunc registers a handler function for the given task type.
func (s *Server) HandleFunc(pattern string, handler func(context.Context, *Task) error) {
	s.mux.HandleFunc(pattern, func(ctx context.Context, t *asynq.Task) error {
		task := &Task{typename: t.Type(), payload: t.Payload()}
		return handler(ctx, task)
	})
}

// Start runs the Asynq worker server loop.
func (s *Server) Start() error {
	slog.Info("Starting Asynq worker server...")
	return s.server.Start(s.mux)
}

// Shutdown gracefully stops the Asynq worker server.
func (s *Server) Shutdown() {
	slog.Info("Shutting down Asynq worker server...")
	s.server.Shutdown()
	slog.Info("Asynq worker server shut down successfully.")
}
