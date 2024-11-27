package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/google/uuid"
)

type SSEServer struct {
	streams map[uuid.UUID]chan JSONRPCMessage
	readCh  chan JSONRPCMessage
	writeCh chan JSONRPCResponse
	wg      sync.WaitGroup
}

func NewSSEServer() *SSEServer {
	return &SSEServer{
		streams: make(map[uuid.UUID]chan JSONRPCMessage),
		readCh:  make(chan JSONRPCMessage, 100),
		writeCh: make(chan JSONRPCResponse, 100),
	}
}

func (s *SSEServer) Start(ctx context.Context) {
	s.wg.Add(1)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	go func() {
		defer s.wg.Done()
		http.HandleFunc("/sse", s.HandleSSE)
		http.HandleFunc("/messages", s.HandleMessage)

		server := &http.Server{Addr: fmt.Sprintf(":%s", port)}

		slog.Info("SSE server started", "port", port)
		go func() {
			<-ctx.Done()
			server.Shutdown(context.Background())
		}()

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server ListenAndServe:", "error", err)
		}
	}()
}

func (s *SSEServer) ReadChannel() <-chan JSONRPCMessage {
	return s.readCh
}

func (s *SSEServer) WriteChannel() chan<- JSONRPCResponse {
	return s.writeCh
}

func (s *SSEServer) Wait() {
	s.wg.Wait()
}

func (s *SSEServer) HandleSSE(w http.ResponseWriter, r *http.Request) {
	slog.Info("ConnectSSE: Setting up SSE connection")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	sessionID := uuid.New()
	sessionURI := fmt.Sprintf("%s?session_id=%s", "/messages", sessionID.String())

	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", sessionURI)
	w.(http.Flusher).Flush()
	slog.Info("SSE: Sent endpoint", "data", sessionURI)

	for msg := range s.writeCh {
		data, _ := json.Marshal(msg)
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(data))
		slog.Debug("SSE: Sent message", "data", string(data))
		w.(http.Flusher).Flush()
	}
}

func (s *SSEServer) HandleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionIDStr := r.URL.Query().Get("session_id")
	if sessionIDStr == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}
	slog.Info("HandleMessage: sessionID", "sessionID", sessionID)

	var message JSONRPCMessage
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		slog.Error("HandleMessage: Could not parse message", "error", err)
		http.Error(w, "Could not parse message", http.StatusBadRequest)
		return
	}
	slog.Debug("HandleMessage: message", "message", message)

	s.readCh <- message
	w.WriteHeader(http.StatusAccepted)
}
