package middleware

import (
	//"advanced-blog-management-system/internal/middleware"

	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ---------------------------------------------------------------------------
// Хелперы
// ---------------------------------------------------------------------------

func newTestLogger(buf *bytes.Buffer) *logrus.Logger {
	logger := logrus.New()
	logger.Out = buf
	logger.Formatter = &logrus.JSONFormatter{}
	logger.Level = logrus.DebugLevel
	return logger
}

func newTestMiddleware(buf *bytes.Buffer) *LoggingMiddleware {
	return NewLoggingMiddleware(newTestLogger(buf))
}

// ---------------------------------------------------------------------------
// RequestID
// ---------------------------------------------------------------------------

func TestRequestID_GeneratesAndSets(t *testing.T) {

	logger := logrus.New()
	logger.Level = logrus.ErrorLevel

	mw := NewLoggingMiddleware(logger)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		reqID, ok := GetRequestIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected request_id in context")
		}
		if reqID == "" {
			t.Fatal("expected non-empty request_id")
		}
		if _, err := uuid.Parse(reqID); err != nil {
			t.Fatalf("expected valid UUID, got %q: %v", reqID, err)
		}

		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	h := mw.RequestID(next)
	h.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal("next handler should be called")
	}

	headerID := rr.Header().Get("X-Request-ID")
	if headerID == "" {
		t.Fatal("expected X-Request-ID header in response")
	}
	if _, err := uuid.Parse(headerID); err != nil {
		t.Fatalf("X-Request-ID should be valid UUID, got %q: %v", headerID, err)
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {

	logger := logrus.New()
	logger.Level = logrus.ErrorLevel

	mw := NewLoggingMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := mw.RequestID(next)

	rr1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	h.ServeHTTP(rr1, req1)

	rr2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	h.ServeHTTP(rr2, req2)

	id1 := rr1.Header().Get("X-Request-ID")
	id2 := rr2.Header().Get("X-Request-ID")

	if id1 == id2 {
		t.Fatalf("expected unique IDs, got %q == %q", id1, id2)
	}
}

// ---------------------------------------------------------------------------
// Logger
// ---------------------------------------------------------------------------

func TestLogger_WithRequestID(t *testing.T) {
	buf := &bytes.Buffer{}
	mw := newTestMiddleware(buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что request_id дошёл до хендлера
		reqID, ok := GetRequestIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected request_id in context")
		}
		if reqID != "test-uuid-123" {
			t.Fatalf("expected test-uuid-123, got %q", reqID)
		}
		w.WriteHeader(http.StatusCreated)
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/posts/42", nil)
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-uuid-123")
	req = req.WithContext(ctx)

	h := mw.Logger(next)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, `"request_id":"test-uuid-123"`) {
		t.Errorf("log should contain request_id, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"status":201`) {
		t.Errorf("log should contain status 201, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"method":"PUT"`) {
		t.Errorf("log should contain method PUT, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"path":"/api/posts/42"`) {
		t.Errorf("log should contain path, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"duration_ms"`) {
		t.Errorf("log should contain duration_ms, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"msg":"request_completed"`) {
		t.Errorf("log should contain request_completed, got: %s", logOutput)
	}
}

func TestLogger_MissingRequestID(t *testing.T) {
	buf := &bytes.Buffer{}
	mw := newTestMiddleware(buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	// Контекст без request_id

	h := mw.Logger(next)
	h.ServeHTTP(rr, req)

	logOutput := buf.String()

	// Должен быть warning про отсутствующий request_id
	if !strings.Contains(logOutput, `"msg":"missing request_id"`) {
		t.Errorf("log should contain missing request_id warning, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"level":"warning"`) {
		t.Errorf("log should contain warning level, got: %s", logOutput)
	}

	// И при этом обычный info-лог тоже должен быть
	if !strings.Contains(logOutput, `"msg":"request_completed"`) {
		t.Errorf("log should still contain request_completed, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"request_id":"no-request-id"`) {
		t.Errorf("log should contain fallback request_id, got: %s", logOutput)
	}
}

// ---------------------------------------------------------------------------
// Recovery
// ---------------------------------------------------------------------------

func TestRecovery_CatchesPanic(t *testing.T) {
	buf := &bytes.Buffer{}
	mw := newTestMiddleware(buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("database connection lost")
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/crash", nil)

	h := mw.Recovery(next)
	h.ServeHTTP(rr, req)

	// Ответ: 500 + JSON
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "internal_server_error" {
		t.Errorf("expected error=internal_server_error, got %v", resp["error"])
	}
	if msg, ok := resp["message"].(string); !ok || !strings.Contains(msg, "unexpected error") {
		t.Errorf("expected message about unexpected error, got %v", resp["message"])
	}

	// Лог: panic_caught + стек
	logOutput := buf.String()
	if !strings.Contains(logOutput, `"msg":"panic_caught"`) {
		t.Errorf("log should contain panic_caught, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"stack"`) {
		t.Errorf("log should contain stack trace, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"path":"/api/crash"`) {
		t.Errorf("log should contain path, got: %s", logOutput)
	}
}

func TestRecovery_NoPanic_NormalFlow(t *testing.T) {
	buf := &bytes.Buffer{}
	mw := newTestMiddleware(buf)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/safe", nil)

	h := mw.Recovery(next)
	h.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal("handler should be called when no panic")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// Логов panic_caught быть не должно
	logOutput := buf.String()
	if strings.Contains(logOutput, "panic_caught") {
		t.Errorf("log should not contain panic_caught, got: %s", logOutput)
	}
}

func TestRecovery_PanicWithRequestID(t *testing.T) {
	buf := &bytes.Buffer{}
	mw := newTestMiddleware(buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/crash", nil)
	ctx := context.WithValue(context.Background(), RequestIDKey, "req-abc-999")
	req = req.WithContext(ctx)

	h := mw.Recovery(next)
	h.ServeHTTP(rr, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, `"request_id":"req-abc-999"`) {
		t.Errorf("log should contain request_id from context, got: %s", logOutput)
	}
}

// ---------------------------------------------------------------------------
// CORS
// ---------------------------------------------------------------------------

func TestCORS_AllowedOrigin(t *testing.T) {
	mw := NewLoggingMiddleware(logrus.New())

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	h := mw.CORS(next)
	h.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal("next handler should be called for allowed origin")
	}
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:3000" {
		t.Fatalf("expected Access-Control-Allow-Origin, got %q", origin)
	}
	if methods := rr.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
	if headers := rr.Header().Get("Access-Control-Allow-Headers"); headers == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}
	if vary := rr.Header().Get("Vary"); !strings.Contains(vary, "Origin") {
		t.Errorf("expected Vary to contain Origin, got %q", vary)
	}
}

func TestCORS_BlockedOrigin(t *testing.T) {
	mw := NewLoggingMiddleware(logrus.New())

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Origin", "http://evil.com")

	h := mw.CORS(next)
	h.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal("next handler should still be called even for blocked origin")
	}
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Errorf("expected no CORS headers for blocked origin, got %q", origin)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	mw := NewLoggingMiddleware(logrus.New())

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/data", nil)
	// Origin не установлен

	h := mw.CORS(next)
	h.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal("next handler should be called when no Origin header")
	}
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Errorf("expected no CORS headers without Origin, got %q", origin)
	}
}

func TestCORS_OptionsPreflight(t *testing.T) {
	mw := NewLoggingMiddleware(logrus.New())

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/api/data", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	h := mw.CORS(next)
	h.ServeHTTP(rr, req)

	// OPTIONS должен вернуть 204 и не вызывать next
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS preflight, got %d", rr.Code)
	}
	if handlerCalled {
		t.Fatal("next handler should NOT be called for OPTIONS preflight")
	}
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:3000" {
		t.Fatalf("expected CORS header for OPTIONS, got %q", origin)
	}
}

// ---------------------------------------------------------------------------
// Chain
// ---------------------------------------------------------------------------

func TestChain_PanicCaught_RequestIDPresent(t *testing.T) {
	buf := &bytes.Buffer{}
	mw := newTestMiddleware(buf)

	requestIDSeen := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetRequestIDFromContext(r.Context())
		if ok {
			requestIDSeen = true
		}
		panic("chain test panic")
	})

	chain := mw.Chain()
	h := chain(next)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/chain", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	h.ServeHTTP(rr, req)

	// Recovery поймал панику → 500
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from Recovery, got %d", rr.Code)
	}

	// RequestID был в контексте до паники
	if !requestIDSeen {
		t.Fatal("expected request_id in context before panic")
	}

	// X-Request-ID в заголовке ответа
	if rr.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header in response")
	}

	// CORS отработал (т.к. origin разрешён)
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Error("expected CORS header from chain")
	}

	// В логе есть panic_caught
	logOutput := buf.String()
	if !strings.Contains(logOutput, "panic_caught") {
		t.Errorf("expected panic_caught in log, got: %s", logOutput)
	}
}

func TestChain_NormalRequest(t *testing.T) {
	buf := &bytes.Buffer{}
	mw := newTestMiddleware(buf)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		_, ok := GetRequestIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected request_id in context")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	chain := mw.Chain()
	h := chain(next)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/normal", nil)

	h.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal("handler should be called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// X-Request-ID установлен
	if rr.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header")
	}

	// Лог содержит request_completed
	logOutput := buf.String()
	if !strings.Contains(logOutput, "request_completed") {
		t.Errorf("expected request_completed in log, got: %s", logOutput)
	}
}

func TestChain_NoRateLimiter_NotBlocked(t *testing.T) {
	buf := &bytes.Buffer{}
	mw := newTestMiddleware(buf)
	// SetRateLimiter НЕ вызывается → rateLimitEnabled = false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	chain := mw.Chain()
	h := chain(next)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/no-limit", nil)

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 without rate limiter, got %d", rr.Code)
	}
}

// -------------------- getClientIT
func TestGetClientIP_AllBranches(t *testing.T) {
	tests := []struct {
		name       string
		xForwarded string
		xRealIP    string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For with multiple IPs",
			xForwarded: "10.0.0.1, 192.168.1.5, 172.16.0.2",
			expected:   "10.0.0.1",
		},
		{
			name:     "X-Real-IP only",
			xRealIP:  "192.168.1.10",
			expected: "192.168.1.10",
		},
		{
			name:       "both headers — X-Forwarded-For wins",
			xForwarded: "10.0.0.1, 192.168.1.5",
			xRealIP:    "172.16.0.20",
			expected:   "10.0.0.1",
		},
		{
			name:       "RemoteAddr with port — strip port",
			remoteAddr: "203.0.113.5:54321",
			expected:   "203.0.113.5",
		},
		{
			name:     "no headers, no RemoteAddr",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.xForwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwarded)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			req.RemoteAddr = tt.remoteAddr

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, ip)
			}
		})
	}
}

// ----------

// ---------------------------------------------------------------------------
// Тест для SetRateLimiter
// ---------------------------------------------------------------------------
func TestSetRateLimiter_EnablesRateLimiting(t *testing.T) {
	mw := NewLoggingMiddleware(logrus.New())

	if mw.RateLimitEnabled() {
		t.Fatal("rate limiter should be disabled by default")
	}
	if mw.GetRateLimiter() != nil {
		t.Fatal("rateLimiter should be nil by default")
	}

	rl := NewRateLimiter(100.0, 10)
	mw.SetRateLimiter(rl)

	if !mw.RateLimitEnabled() {
		t.Fatal("rateLimitEnabled should be true after SetRateLimiter")
	}
	if mw.GetRateLimiter() == nil {
		t.Fatal("rateLimiter should not be nil after SetRateLimiter")
	}
}

// ---------------------------------------------------------------------------
// Тесты для LimitRate
// ---------------------------------------------------------------------------

func TestLimitRate_Disabled_Passthrough(t *testing.T) {
	mw := NewLoggingMiddleware(logrus.New())
	// SetRateLimiter НЕ вызывается → rateLimitEnabled = false

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/posts", nil)

	h := mw.LimitRate(next)
	h.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal("handler should be called when rate limiter is disabled")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("X-RateLimit-Limit") != "" {
		t.Error("should not set X-RateLimit-Limit when disabled")
	}
}

func TestLimitRate_HealthPath_SkipsLimit(t *testing.T) {
	mw := NewLoggingMiddleware(logrus.New())
	rl := NewRateLimiter(100.0, 10)
	mw.SetRateLimiter(rl)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/health", nil)

	h := mw.LimitRate(next)
	h.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal("handler should be called for /api/health")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("X-RateLimit-Limit") != "" {
		t.Error("should not set rate limit headers for health path")
	}
}

func TestLimitRate_TokenAvailable_AllowsRequest(t *testing.T) {
	mw := NewLoggingMiddleware(logrus.New())
	rl := NewRateLimiter(100.0, 10)
	mw.SetRateLimiter(rl)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/posts", nil)
	req.Header.Set("X-Real-IP", "192.168.1.50")

	h := mw.LimitRate(next)
	h.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal("handler should be called when token is available")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("should set X-RateLimit-Limit")
	}
	if rr.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("should set X-RateLimit-Remaining")
	}
}

func TestLimitRate_TokenExhausted_Returns429(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logrus.New()
	logger.SetOutput(buf)
	logger.SetLevel(logrus.WarnLevel) // чтобы видеть warning про blocked

	mw := NewLoggingMiddleware(logger)

	// capacity=1, очень медленный refill — после первого запроса бакет пуст
	rl := NewRateLimiter(0.0001, 1)
	mw.SetRateLimiter(rl)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := mw.LimitRate(next)

	// Первый запрос — забирает единственный токен
	rr1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/api/posts", nil)
	req1.Header.Set("X-Real-IP", "10.0.0.99")
	h.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("first request expected 200, got %d", rr1.Code)
	}

	// Второй запрос — токенов нет → 429
	rr2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/posts", nil)
	req2.Header.Set("X-Real-IP", "10.0.0.99")
	h.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr2.Code)
	}

	// Тело ответа — JSON с ошибкой
	var resp map[string]string
	if err := json.Unmarshal(rr2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "rate_limit_exceeded" {
		t.Errorf("expected error=rate_limit_exceeded, got %q", resp["error"])
	}

	// Заголовки
	if rr2.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
	if rr2.Header().Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type application/json")
	}
	if rr2.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header even on 429")
	}

	// Лог содержит warning про rate limit
	logOutput := buf.String()
	if !strings.Contains(logOutput, "blocked") {
		t.Errorf("expected 'blocked' in log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "10.0.0.99") {
		t.Errorf("expected IP in log, got: %s", logOutput)
	}
}
