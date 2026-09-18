// post_handler_test.go
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"advanced-blog-management-system/internal/middleware"
	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/internal/repository"
	"advanced-blog-management-system/internal/service"
	"advanced-blog-management-system/pkg/apperr"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	userID = 42
	postID = 123
)

// mockPostRepo реализует интерфейс repository.PostRepository
type mockPostRepo struct {
	getAllFunc   func(ctx context.Context, limit, offset int) ([]*model.Post, error)
	getTotalFunc func(ctx context.Context) (int, error)
	createFunc   func(ctx context.Context, post *model.Post) error
	getByIDFunc  func(ctx context.Context, id int) (*model.Post, error)
	updateFunc   func(ctx context.Context, post *model.Post) error
	deleteFunc   func(ctx context.Context, id int) error
	existsFunc   func(ctx context.Context, id int) (bool, error)
}

func (m *mockPostRepo) Create(ctx context.Context, post *model.Post) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, post)
	}
	return nil
}

func (m *mockPostRepo) GetByID(ctx context.Context, id int) (*model.Post, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockPostRepo) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockPostRepo) GetTotalCount(ctx context.Context) (int, error) {
	if m.getTotalFunc != nil {
		return m.getTotalFunc(ctx)
	}
	return 0, nil
}

func (m *mockPostRepo) Update(ctx context.Context, post *model.Post) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, post)
	}
	return nil
}

func (m *mockPostRepo) Delete(ctx context.Context, id int) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockPostRepo) Exists(ctx context.Context, id int) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	// По умолчанию считаем, что пост существует, если мок не настроен иначе
	return true, nil
}

// Гарантия: мок всегда реализует интерфейс
var _ repository.PostRepository = (*mockPostRepo)(nil)

// === ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ===

func ptrTime(t time.Time) *time.Time {
	return &t
}

// ВАЖНО: замени testUserIDKey на тот ключ, который использует твоя функция getUserIDFromContext.
// Если у тебя ключ объявлен в middleware или в handler, используй его.
// Например: если в коде ключ называется "userID" с типом string — поменяй здесь.
//type testCtxKey string

//const testUserIDKey testCtxKey = "userID"

func reqWithUserID(r *http.Request, userID int) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func TestPostHandler_GetAll_Success(t *testing.T) {
	// 1. Настраиваем мок репозитория
	mockRepo := &mockPostRepo{
		getAllFunc: func(ctx context.Context, limit, offset int) ([]*model.Post, error) {
			// Возвращаем данные, которые ожидает тест
			return []*model.Post{
				{ID: 1, Title: "Post 1", Content: "Content 1", AuthorID: 1},
				{ID: 2, Title: "Post 2", Content: "Content 2", AuthorID: 1},
				{ID: 3, Title: "Post 3", Content: "Content 3", AuthorID: 1},
			}, nil
		},
		getTotalFunc: func(ctx context.Context) (int, error) {
			// Сервис вызовет этот метод для формирования ответа Total
			return 10, nil
		},
	}

	// 2. Создаем РЕАЛЬНЫЙ сервис, но с МОК репозиторием.
	// Это ключевое изменение: мы используем настоящий конструктор NewPostService.
	// Второй аргумент (userRepo) передаем nil, так как в методе GetAll он не используется.
	realService := service.NewPostService(mockRepo, nil)

	// 3. Передаем реальный сервис в хендлер.
	// Теперь типы совпадают на 100%: realService имеет тип *service.PostService
	h := &PostHandler{postService: realService}

	req, _ := http.NewRequest(http.MethodGet, "/posts?limit=3&offset=5", nil)

	w := httptest.NewRecorder()

	h.GetAll(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp struct {
		Posts  []*model.Post `json:"posts"`
		Total  int           `json:"total"`
		Limit  int           `json:"limit"`
		Offset int           `json:"offset"`
	}
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)

	assert.Len(t, resp.Posts, 3)
	assert.Equal(t, 10, resp.Total)
	assert.Equal(t, 3, resp.Limit)
	assert.Equal(t, 5, resp.Offset)
}

func TestPostHandler_GetAll_InvalidLimit(t *testing.T) {
	mockRepo := &mockPostRepo{}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	req, _ := http.NewRequest(http.MethodGet, "/posts?limit=-5", nil)
	w := httptest.NewRecorder()

	h.GetAll(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostHandler_GetAll_MethodNotAllowed(t *testing.T) {
	mockRepo := &mockPostRepo{}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	req, _ := http.NewRequest(http.MethodPost, "/posts", nil)
	w := httptest.NewRecorder()

	h.GetAll(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestPostHandler_GetAll_InternalError(t *testing.T) {
	mockRepo := &mockPostRepo{
		getAllFunc: func(ctx context.Context, limit, offset int) ([]*model.Post, error) {
			return nil, fmt.Errorf("database down")
		},
	}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	req, _ := http.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()

	h.GetAll(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPostHandler_GetByID_Success(t *testing.T) {
	expectedPost := &model.Post{
		ID:       123,
		Title:    "My Great Post",
		Content:  "Content here...",
		AuthorID: userID,
		Status:   "draft",
	}

	mockRepo := &mockPostRepo{
		getByIDFunc: func(ctx context.Context, id int) (*model.Post, error) {
			return expectedPost, nil
		},
	}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	// Создаём chi-роутер и регистрируем маршрут
	r := chi.NewRouter()
	r.Get("/posts/{id}", h.GetByID)

	req, _ := http.NewRequest(http.MethodGet, "/posts/123", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req) // <-- запрос идёт через роутер, chi заполнит URLParam

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp model.Post
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, expectedPost.ID, resp.ID)
	assert.Equal(t, expectedPost.Title, resp.Title)
}

func TestPostHandler_GetByID_NotFound(t *testing.T) {
	mockRepo := &mockPostRepo{
		getByIDFunc: func(ctx context.Context, id int) (*model.Post, error) {
			return nil, apperr.ErrPostNotFound
		},
	}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	r := chi.NewRouter()
	r.Get("/posts/{id}", h.GetByID)

	req, _ := http.NewRequest(http.MethodGet, "/posts/999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPostHandler_GetByID_InternalError(t *testing.T) {
	mockRepo := &mockPostRepo{
		getByIDFunc: func(ctx context.Context, id int) (*model.Post, error) {
			return nil, fmt.Errorf("db connection lost")
		},
	}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	r := chi.NewRouter()
	r.Get("/posts/{id}", h.GetByID)

	req, _ := http.NewRequest(http.MethodGet, "/posts/123", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPostHandler_Create_Success(t *testing.T) {
	//userID := 42

	mockRepo := &mockPostRepo{
		createFunc: func(ctx context.Context, post *model.Post) error {
			post.ID = 123
			post.Status = "published"
			post.CreatedAt = time.Now()
			post.UpdatedAt = time.Now()
			return nil
		},
	}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	body := `{"title":"New Post","content":"Content here..."}`
	r := chi.NewRouter()
	r.Post("/posts", h.Create)

	req, _ := http.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUserID(req, 42)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp model.Post
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 123, resp.ID)
	assert.Equal(t, "New Post", resp.Title)
	assert.Equal(t, userID, resp.AuthorID)
}

func TestPostHandler_Create_InvalidJSON(t *testing.T) {
	mockRepo := &mockPostRepo{}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	r := chi.NewRouter()
	r.Post("/posts", h.Create)

	// Невалидный JSON
	req, _ := http.NewRequest(http.MethodPost, "/posts", strings.NewReader("{title: invalid}"))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUserID(req, 42)

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostHandler_Create_ValidationFail(t *testing.T) {
	mockRepo := &mockPostRepo{}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	// Пустой title — сервис должен отклонить
	body := `{"title":"","content":"some content"}`

	r := chi.NewRouter()
	r.Post("/posts", h.Create)

	req, _ := http.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUserID(req, 42)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Сервис возвращает ошибку валидации, хендлер отдаёт 500
	// (т.к. в хендлере нет отдельной обработки ошибок валидации)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPostHandler_Create_InternalError(t *testing.T) {
	mockRepo := &mockPostRepo{
		createFunc: func(ctx context.Context, post *model.Post) error {
			return fmt.Errorf("db error")
		},
	}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	body := `{"title":"New Post","content":"Content here..."}`
	r := chi.NewRouter()
	r.Post("/posts", h.Create)

	req, _ := http.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUserID(req, 42)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// === ТЕСТЫ Update ===

func TestPostHandler_Update_Success(t *testing.T) {
	//userID := 42
	//p/ostID := 123

	existingPost := &model.Post{
		ID:       postID,
		Title:    "Old Title",
		Content:  "Old content",
		AuthorID: userID, // пользователь — автор
		Status:   "published",
	}

	mockRepo := &mockPostRepo{
		getByIDFunc: func(ctx context.Context, id int) (*model.Post, error) {
			return existingPost, nil
		},
		updateFunc: func(ctx context.Context, post *model.Post) error {
			post.UpdatedAt = time.Now()
			return nil
		},
	}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	body := `{"title":"Updated Title","content":"Updated content"}`

	r := chi.NewRouter()
	r.Patch("/posts/{id}", h.Update)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/posts/%d", postID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUserID(req, userID)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp model.Post
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", resp.Title)
	assert.Equal(t, "Updated content", resp.Content)
}

func TestPostHandler_Update_NotFound(t *testing.T) {
	//userID := 42

	mockRepo := &mockPostRepo{
		getByIDFunc: func(ctx context.Context, id int) (*model.Post, error) {
			return nil, apperr.ErrPostNotFound
		},
	}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	body := `{"title":"Updated Title"}`

	r := chi.NewRouter()
	r.Patch("/posts/{id}", h.Update)

	req, _ := http.NewRequest(http.MethodPatch, "/posts/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUserID(req, userID)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPostHandler_Update_Forbidden(t *testing.T) {
	//userID := 42
	//postID := 123

	// Пост принадлежит другому автору
	existingPost := &model.Post{
		ID:       postID,
		Title:    "Old Title",
		Content:  "Old content",
		AuthorID: 99, // другой автор!
	}

	mockRepo := &mockPostRepo{
		getByIDFunc: func(ctx context.Context, id int) (*model.Post, error) {
			return existingPost, nil
		},
	}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	body := `{"title":"Updated Title"}`

	r := chi.NewRouter()
	r.Patch("/posts/{id}", h.Update)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/posts/%d", postID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUserID(req, userID)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPostHandler_Update_InvalidID(t *testing.T) {
	mockRepo := &mockPostRepo{}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	body := `{"title":"Updated Title"}`

	r := chi.NewRouter()
	r.Patch("/posts/{id}", h.Update)

	req, _ := http.NewRequest(http.MethodPatch, "/posts/abc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUserID(req, 42)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostHandler_Update_InvalidJSON(t *testing.T) {
	mockRepo := &mockPostRepo{}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	r := chi.NewRouter()
	r.Patch("/posts/{id}", h.Update)

	req, _ := http.NewRequest(http.MethodPatch, "/posts/123", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUserID(req, 42)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostHandler_Update_InternalError(t *testing.T) {
	//userID := 42
	//postID := 123

	mockRepo := &mockPostRepo{
		getByIDFunc: func(ctx context.Context, id int) (*model.Post, error) {
			return &model.Post{ID: postID, AuthorID: userID, Title: "Old", Content: "Old", Status: "draft"}, nil
		},
		updateFunc: func(ctx context.Context, post *model.Post) error {
			return fmt.Errorf("db error")
		},
	}
	realService := service.NewPostService(mockRepo, nil)
	h := &PostHandler{postService: realService}

	body := `{"title":"Updated Title"}`

	r := chi.NewRouter()
	r.Patch("/posts/{id}", h.Update)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/posts/%d", postID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = reqWithUserID(req, userID)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
