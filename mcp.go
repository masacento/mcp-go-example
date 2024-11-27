package main

import "context"

type Server interface {
	Start(ctx context.Context)
	WriteChannel() chan<- JSONRPCResponse
	ReadChannel() <-chan JSONRPCMessage
	Wait()
}

type JSONRPCMessage struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      uint           `json:"id,omitempty"`
	Method  string         `json:"method,omitempty"`
	Params  *JSONRPCParams `json:"params,omitempty"`
	Result  any            `json:"result,omitempty"`
	Error   *JSONRPCError  `json:"error,omitempty"`
}

type JSONRPCParams struct {
	Name string         `json:"name,omitempty"`
	Args map[string]any `json:"arguments,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      uint          `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}
