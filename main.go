package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
)

//go:embed tools.json
var toolsjson []byte

func main() {
	transport := flag.String("transport", "stdio", "transport type [stdio or sse]")
	logpath := flag.String("logpath", "/tmp/mcp-go-example.log", "log path")
	dbpath := flag.String("dbpath", "/tmp/test.db", "sqlite db path")
	flag.Parse()

	var logger io.WriteCloser
	if *transport == "stdio" {
		logger = newFileLogger(*logpath)
	} else {
		logger = os.Stderr
	}
	defer logger.Close()
	setLogger(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var server Server
	if *transport == "stdio" {
		slog.Info("using stdio transport")
		server = NewStdioServer(os.Stdin, os.Stdout)
	} else if *transport == "sse" {
		slog.Info("using sse transport")
		server = NewSSEServer()
	}
	server.Start(ctx)
	db, err := NewSQLite(*dbpath)
	if err != nil {
		slog.Error("failed to create sqlite db", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("server started")
	var insights = []string{}
	go func() {
		for msg := range server.ReadChannel() {
			slog.Info("received message", "method", msg.Method, "id", msg.ID)
			if strings.HasPrefix(msg.Method, "notifications/") {
				slog.Debug("notification", "msg", msg)
				continue
			}

			switch msg.Method {
			case "ping":
				resp := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      msg.ID,
					Result:  map[string]any{},
				}
				slog.Debug("sending ping response", "msg", msg, "resp", resp)
				server.WriteChannel() <- resp

			case "initialize":
				resp := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      msg.ID,
					Result:  map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{"experimental": map[string]any{}, "prompts": map[string]any{"listChanged": false}, "resources": map[string]any{"subscribe": false, "listChanged": false}, "tools": map[string]any{"listChanged": false}}, "serverInfo": map[string]any{"name": "sqlite", "version": "0.1.0"}},
				}
				slog.Debug("sending initialize response", "msg", resp)
				server.WriteChannel() <- resp

			case "tools/list":
				var resp JSONRPCResponse
				json.Unmarshal(toolsjson, &resp)
				resp.ID = msg.ID
				slog.Debug("sending tools response", "msg", msg)
				server.WriteChannel() <- resp

			case "prompts/list":
				resp := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      msg.ID,
					Result:  map[string]any{"prompts": []map[string]any{{"name": "mcp-demo", "description": "A prompt to seed the database with initial data and demonstrate what you can do with an SQLite MCP Server + Claude", "arguments": []map[string]any{{"name": "topic", "description": "Topic to seed the database with initial data", "required": true}}}}},
				}
				slog.Debug("sending prompts response", "msg", msg, "resp", resp)
				server.WriteChannel() <- resp

			case "prompts/get":
				topic, _ := msg.Params.Args["topic"].(string)
				resp := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      msg.ID,
					Result:  map[string]any{"description": "This is a test prompt", "messages": []map[string]any{{"role": "user", "content": map[string]any{"type": "text", "text": "The assistants goal is to walkthrough an informative demo of MCP. topic: " + topic}}}},
				}
				slog.Debug("sending prompts response", "msg", msg, "resp", resp)
				server.WriteChannel() <- resp

			case "resources/list":
				resp := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      msg.ID,
					Result:  map[string]any{"resources": []map[string]any{{"uri": "memo://insights", "name": "Business Insights Memo", "description": "A living document of discovered business insights", "mimeType": "text/plain"}}},
				}
				slog.Debug("sending resources response", "msg", msg, "resp", resp)
				server.WriteChannel() <- resp

			case "resources/read":
				resp := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      msg.ID,
					Result:  map[string]any{"contents": []map[string]any{{"uri": "memo://insights", "mimeType": "text/plain", "text": strings.Join(insights, "\n")}}},
				}
				slog.Debug("sending resources response", "msg", msg, "resp", resp)
				server.WriteChannel() <- resp

			case "resources/write":
				slog.Debug("sending resources/write response", "msg", msg)

			case "tools/call":
				if msg.Params.Name == "append-insight" {
					insight, _ := msg.Params.Args["insight"].(string)
					insights = append(insights, insight)

					resp := JSONRPCResponse{
						JSONRPC: "2.0",
						ID:      msg.ID,
						Result:  map[string]any{"content": []map[string]any{{"type": "text", "text": insight}}},
					}
					slog.Debug("sending tools/call append-insight response", "msg", msg, "resp", resp)
					server.WriteChannel() <- resp
					continue
				}

				query, _ := msg.Params.Args["query"].(string)
				results, err := db.Call(msg.Params.Name, query)
				if err != nil {
					slog.Error("failed to call sqlite", "error", err)
					continue
				}
				resp := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      msg.ID,
					Result:  results,
				}

				slog.Debug("sending tools/call response", "msg", msg, "resp", resp)
				server.WriteChannel() <- resp
			default:
				slog.Info("unknown method called", "method", msg.Method)
			}
		}
	}()

	server.Wait()
	slog.Info("server finished")
}

func setLogger(w io.Writer) {
	logger := slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
}

func newFileLogger(logpath string) io.WriteCloser {
	logfile, err := os.OpenFile(logpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	return logfile
}
