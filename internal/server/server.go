package server

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/aiujdha/ai-gateway/internal/relay"
	"github.com/aiujdha/ai-gateway/internal/trace"
)

//go:embed index.html
var page string

const LocalKey = "local-gateway-key"

type Server struct {
	mu      sync.Mutex
	balance int
	events  []trace.Event
}

func New() *Server { return &Server{balance: 10000, events: []trace.Event{}} }
func id() string {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value)
}
func jsonResponse(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func failure(w http.ResponseWriter, code int, message string) {
	jsonResponse(w, code, map[string]any{"error": map[string]string{"message": message}})
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, page)
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { jsonResponse(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /v1/models", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{"object": "list", "data": []any{map[string]string{"id": "local-chat", "object": "model", "owned_by": "local"}}})
	}))
	mux.HandleFunc("POST /v1/chat/completions", s.authorized(s.chat))
	mux.HandleFunc("GET /api/status", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		jsonResponse(w, 200, map[string]any{"remaining_units": s.balance, "events": s.events})
	}))
	return mux
}
func (s *Server) authorized(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+LocalKey {
			failure(w, 401, "Invalid local API key")
			return
		}
		next(w, r)
	}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
}

func (s *Server) record(event trace.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	if len(s.events) > 100 {
		s.events = s.events[len(s.events)-100:]
	}
}
func (s *Server) chat(w http.ResponseWriter, r *http.Request) {
	var request chatRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	if err := decoder.Decode(&request); err != nil {
		failure(w, 400, "Invalid JSON request")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		failure(w, 400, "Only one JSON object is accepted")
		return
	}
	if request.Model != "local-chat" {
		failure(w, 404, "Unknown model")
		return
	}
	if len(request.Messages) == 0 {
		failure(w, 400, "messages is required")
		return
	}
	prompt := strings.TrimSpace(request.Messages[len(request.Messages)-1].Content)
	if prompt == "" || utf8.RuneCountInString(prompt) > 2000 {
		failure(w, 400, "Last message must contain 1 to 2000 characters")
		return
	}
	requestID, traceID := id(), id()
	w.Header().Set("X-Request-ID", requestID)
	w.Header().Set("X-Trace-ID", traceID)
	var result relay.Response
	selected := ""
	for attempt, channel := range []string{"primary", "backup"} {
		started := time.Now()
		code, errCode := 200, ""
		if r.Header.Get("X-Local-Fail-All") == "true" || (attempt == 0 && r.Header.Get("X-Local-Fail-Primary") == "true") {
			code, errCode = 503, "simulated_upstream_failure"
		} else {
			var err error
			result, err = (relay.LocalProvider{Channel: channel}).Generate(r.Context(), relay.Request{Model: request.Model, Prompt: prompt})
			if err != nil {
				code, errCode = 503, "request_canceled"
			}
		}
		s.record(trace.Event{RequestID: requestID, TraceID: traceID, ChannelID: channel, Model: request.Model, Attempt: attempt + 1, StatusCode: code, DurationMS: time.Since(started).Milliseconds(), ErrorCode: errCode})
		if code == 200 {
			selected = channel
			break
		}
	}
	if selected == "" {
		failure(w, 502, "All local channels failed; quota unchanged")
		return
	}
	units := result.InputTokens + result.OutputTokens
	s.mu.Lock()
	if s.balance < units {
		s.mu.Unlock()
		failure(w, 429, "Local quota exhausted; restart server to reset")
		return
	}
	s.balance -= units
	s.mu.Unlock()
	w.Header().Set("X-Channel-ID", selected)
	usage := map[string]int{"prompt_tokens": result.InputTokens, "completion_tokens": result.OutputTokens, "total_tokens": units}
	if !request.Stream {
		jsonResponse(w, 200, map[string]any{"id": requestID, "object": "chat.completion", "model": request.Model, "choices": []any{map[string]any{"index": 0, "message": message{Role: "assistant", Content: result.Text}, "finish_reason": "stop"}}, "usage": usage})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}
	for _, character := range result.Text {
		if r.Context().Err() != nil {
			return
		}
		data, _ := json.Marshal(map[string]any{"id": requestID, "object": "chat.completion.chunk", "model": request.Model, "choices": []any{map[string]any{"index": 0, "delta": map[string]string{"content": string(character)}, "finish_reason": nil}}})
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return
		}
		flusher.Flush()
	}
	final, _ := json.Marshal(map[string]any{"id": requestID, "choices": []any{map[string]any{"index": 0, "delta": map[string]string{}, "finish_reason": "stop"}}, "usage": usage})
	_, _ = fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", final)
	flusher.Flush()
}
