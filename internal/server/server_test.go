package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFailoverAndQuota(t *testing.T) {
	s := New()
	handler := s.Handler()
	request := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"local-chat","messages":[{"role":"user","content":"hello"}]}`))
	request.Header.Set("Authorization", "Bearer "+LocalKey)
	request.Header.Set("X-Local-Fail-Primary", "true")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || response.Header().Get("X-Channel-ID") != "backup" {
		t.Fatal(response.Code, response.Body.String())
	}
	if len(s.events) != 2 || s.events[0].StatusCode != 503 || s.events[1].StatusCode != 200 {
		t.Fatal(s.events)
	}
	if s.balance >= 10000 {
		t.Fatal("quota was not settled")
	}
	if s.events[0].RequestID != s.events[1].RequestID {
		t.Fatal("trace correlation lost")
	}
}
func TestStreamAndFailures(t *testing.T) {
	for _, test := range []struct {
		name, body, key, fail string
		code                  int
	}{
		{"stream", `{"model":"local-chat","messages":[{"content":"你好"}],"stream":true}`, LocalKey, "", 200},
		{"auth", `{}`, "bad", "", 401},
		{"model", `{"model":"unknown"}`, LocalKey, "", 404},
		{"json", `not json`, LocalKey, "", 400},
		{"all_failed", `{"model":"local-chat","messages":[{"content":"hi"}]}`, LocalKey, "true", 502},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := New()
			request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(test.body))
			request.Header.Set("Authorization", "Bearer "+test.key)
			request.Header.Set("X-Local-Fail-All", test.fail)
			response := httptest.NewRecorder()
			s.Handler().ServeHTTP(response, request)
			if response.Code != test.code {
				t.Fatal(response.Code, response.Body.String())
			}
			if test.name == "stream" && !strings.Contains(response.Body.String(), "data: [DONE]") {
				t.Fatal("stream not completed")
			}
			if test.code != 200 && s.balance != 10000 {
				t.Fatal("failed request consumed quota")
			}
		})
	}
}
