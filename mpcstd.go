package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

type StdioServer struct {
	inReader  *bufio.Scanner
	outWriter io.Writer
	readCh    chan JSONRPCMessage
	writeCh   chan JSONRPCResponse
	wg        sync.WaitGroup
}

func NewStdioServer(stdin io.Reader, stdout io.Writer) *StdioServer {
	return &StdioServer{
		inReader:  bufio.NewScanner(stdin),
		outWriter: stdout,
		readCh:    make(chan JSONRPCMessage, 100),
		writeCh:   make(chan JSONRPCResponse, 100),
	}
}

func (s *StdioServer) Start(ctx context.Context) {
	s.wg.Add(2)

	go s.readLoop(ctx)
	go s.writeLoop(ctx)
}

func (s *StdioServer) Wait() {
	s.wg.Wait()
}

func (s *StdioServer) ReadChannel() <-chan JSONRPCMessage {
	return s.readCh
}

func (s *StdioServer) WriteChannel() chan<- JSONRPCResponse {
	return s.writeCh
}

func (s *StdioServer) readLoop(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !s.inReader.Scan() {
				if err := s.inReader.Err(); err != nil {
					slog.Error("Error reading input", "error", err)
				}
				return
			}

			var msg JSONRPCMessage
			if err := json.Unmarshal([]byte(s.inReader.Text()), &msg); err != nil {
				slog.Error("Failed to parse JSON", "error", err)
				continue
			}

			s.readCh <- msg
		}
	}
}

func (s *StdioServer) writeLoop(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-s.writeCh:
			if !ok {
				return
			}

			jsonData, err := json.Marshal(msg)
			if err != nil {
				slog.Error("Failed to encode JSON", "error", err)
				continue
			}

			if _, err := fmt.Fprintln(s.outWriter, string(jsonData)); err != nil {
				slog.Error("Failed to write to stdout", "error", err)
				return
			}
		}
	}
}
