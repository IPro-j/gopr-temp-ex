package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// contextKey — кастомный тип для ключей контекста, чтобы избежать коллизий
type contextKey string

const (
	RequestIDKey contextKey = "requestID"
)

// LoggingMiddleware предоставляет утилиты: логирование, CORS, recovery, request ID и т.д.
//type LoggingMiddleware struct {
//	logger *log.Logger
//}

type LoggingMiddleware struct {
	logger           *logrus.Entry
	rateLimiter      *RateLimiter
	rateLimitEnabled bool
}

func NewLoggingMiddleware(logger *logrus.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger.WithField("component", "middleware"),
	}
}

// SetRateLimiter подключает rate limiter к middleware
func (m *LoggingMiddleware) SetRateLimiter(rl *RateLimiter) {
	m.rateLimiter = rl
	m.rateLimitEnabled = true
}

func (m *LoggingMiddleware) LimitRate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.rateLimitEnabled || m.rateLimiter == nil {
			next(w, r)
			return
		}

		// health-check не лимитируется
		if r.URL.Path == "/api/health" {
			next(w, r)
			return
		}

		ip := getClientIP(r)
		bucket := m.rateLimiter.GetBucket(ip)

		if bucket.TakeAvailable(1) == 0 {
			m.logger.
				WithField("ip", ip).
				WithField("path", r.URL.Path).
				Warn("rate limit exceeded")

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", m.rateLimiter.capacity))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("Retry-After", "1")

			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Too many requests. Please try again later."}`))
			return
		}

		remaining := bucket.Available()
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", m.rateLimiter.capacity))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		next(w, r)
	}
}

// Logger логирует все HTTP-запросы с временем выполнения и статусом
func (m *LoggingMiddleware) Logger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		reqID, ok := GetRequestIDFromContext(r.Context())
		if !ok {
			// Это ненормально: RequestID должен быть всегда.
			m.logger.WithField("level", "warning").
				WithField("method", r.Method).
				WithField("path", r.URL.Path).
				Warn("missing request_id")
			reqID = "no-request-id"
		}

		rw := newResponseWriter(w)
		next(rw, r) // если тут паника — управление уйдёт в Recovery, а не сюда

		duration := time.Since(start)
		ip := getClientIP(r)

		// Эта строка выполнится только если паники не было
		m.logger.
			WithField("request_id", reqID).
			WithField("ip", ip).
			WithField("method", r.Method).
			WithField("path", r.URL.Path).
			WithField("status", rw.statusCode).
			WithField("duration_ms", duration.Milliseconds()).
			Info("request_completed")
	}
}

func (m *LoggingMiddleware) Chain() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Порядок важен: CORS → LimitRate → RequestID → Logger → Recovery
		h := http.HandlerFunc(m.Recovery(next.ServeHTTP)) // ловит панику, когда request_id уже есть
		h = http.HandlerFunc(m.Logger(h.ServeHTTP))       // логирует уже с request_id
		h = http.HandlerFunc(m.RequestID(h.ServeHTTP))    // создаёт request_id до Recovery
		if m.rateLimitEnabled && m.rateLimiter != nil {
			h = http.HandlerFunc(m.LimitRate(h.ServeHTTP)) // после CORS, до RequestID
		}

		h = http.HandlerFunc(m.CORS(h.ServeHTTP)) // самый внешний

		return h
	}
}

func (m *LoggingMiddleware) Recovery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				reqID, _ := GetRequestIDFromContext(r.Context())
				ip := getClientIP(r)
				stack := string(debug.Stack())

				// Лог: всё важное для отладки (включая стек)
				m.logger.
					WithField("request_id", reqID).
					WithField("ip", ip).
					WithField("method", r.Method).
					WithField("path", r.URL.Path).
					WithError(fmt.Errorf("%v", v)). // или просто Error(v)
					WithField("stack", stack).
					Error("panic_caught")

				// Ответ клиенту: единый JSON-формат, без стека
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				resp := map[string]any{
					"error":   "internal_server_error",
					"message": "An unexpected error occurred. Please try again later.",
				}

				// Используем простой json.Marshal
				data, err := json.Marshal(resp)
				if err != nil {
					// Если даже маршалинг упал — пишем хотя бы текст
					w.Write([]byte(`{"error":"internal_server_error","message":"An unexpected error occurred."}`))
					return
				}
				w.Write(data)
			}
		}()

		next(w, r)
	}
}

// CORS добавляет CORS-заголовки
func (m *LoggingMiddleware) CORS(next http.HandlerFunc) http.HandlerFunc {
	// В реальном проекте сделать чтение из env/config
	allowedOrigins := map[string]struct{}{
		"http://localhost:3000":     {},
		"https://myapp.com":         {},
		"https://app.mycompany.com": {},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Если Origin нет — это не браузерный CORS-запрос, можно просто идти дальше
		if origin == "" {
			next(w, r)
			return
		}

		// Проверяем, есть ли origin в whitelist
		if _, ok := allowedOrigins[origin]; !ok {
			// Не отвечаем CORS-заголовками и не разрешаем запрос.
			// Клиент получит ошибку , что правильно.
			next(w, r)
			return
		}

		// Разрешённый origin найден — ставим заголовки
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Vary: Origin нужен, чтобы кэши (CDN, прокси) не отдавали один ответ для разных origin
		w.Header().Add("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// RequestID генерирует уникальный ID запроса, кладёт в контекст и добавляет в заголовок ответа
func (m *LoggingMiddleware) RequestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String()

		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)
		r = r.WithContext(ctx)

		w.Header().Set("X-Request-ID", reqID)

		next(w, r)
	}
}

// ContentTypeJSON принудительно ставит Content-Type: application/json для всех ответов
func (m *LoggingMiddleware) ContentTypeJSON(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next(w, r)
	}
}

// getClientIP извлекает реальный IP клиента с учётом прокси
func getClientIP(r *http.Request) string {
	// X-Forwarded-For: client, proxy1, proxy2
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// Fallback: RemoteAddr (может быть в формате ip:port)
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx]
	}
	return addr
}

// responseWriter — обёртка для перехвата статус-кода
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		written:        false,
	}
}

// GetRequestIDFromContext извлекает RequestID из контекста
func GetRequestIDFromContext(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(RequestIDKey).(string)
	return val, ok
}
