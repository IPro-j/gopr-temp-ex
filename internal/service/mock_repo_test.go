package service

import (
	"context"
	"sync"
	"time"

	"advanced-blog-management-system/internal/model"
)

type MockPostRepo struct {
	mu sync.Mutex

	// Create
	createdPost *model.Post
	createErr   error

	// GetByID
	getByIDPost *model.Post
	getByIDErr  error

	// GetAll
	getAllPosts []*model.Post
	getAllErr   error

	// Exists
	existsVal bool
	existsErr error

	totalCount int
	countErr   error
}

func NewMockPostRepo() *MockPostRepo {
	return &MockPostRepo{
		existsVal: true, // по умолчанию считаем, что пост существует
	}
}

// --- Setters ---

func (m *MockPostRepo) SetCreateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createErr = err
}

func (m *MockPostRepo) SetGetByIDPost(post *model.Post, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getByIDPost = post
	m.getByIDErr = err
}

func (m *MockPostRepo) SetGetAllPosts(posts []*model.Post) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Копируем только срез, сами указатели остаются теми же
	m.getAllPosts = make([]*model.Post, len(posts))
	copy(m.getAllPosts, posts)
}

func (m *MockPostRepo) SetGetAllError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getAllErr = err
}

func (m *MockPostRepo) SetExists(val bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.existsVal = val
	m.existsErr = err
}

// --- Getters / Helpers ---

func (m *MockPostRepo) GetCreatedPost() *model.Post {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.createdPost
}

// --- Repository Methods ---

func (m *MockPostRepo) Create(ctx context.Context, post *model.Post) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createErr != nil {
		return m.createErr
	}

	// Не меняем оригинальный объект: создаём копию или заполняем отдельное поле.
	// Здесь мы просто запоминаем «сохранённый» пост, не трогая оригинал.
	saved := &model.Post{
		Title:     post.Title,
		Content:   post.Content,
		Status:    post.Status,
		PublishAt: post.PublishAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		// ID можно присвоить здесь, если это часть логики репозитория
		ID: 1,
	}
	m.createdPost = saved
	return nil
}

func (m *MockPostRepo) GetByID(ctx context.Context, id int) (*model.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getByIDPost, m.getByIDErr
}

func (m *MockPostRepo) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getAllErr != nil {
		return nil, m.getAllErr
	}

	// Возвращаем копию среза, чтобы внешний код не мог сделать getAllPosts = []...
	res := make([]*model.Post, len(m.getAllPosts))
	copy(res, m.getAllPosts)
	return res, nil
}

func (m *MockPostRepo) Update(ctx context.Context, post *model.Post) error {
	// Заглушка: ничего не делаем, считаем успешным
	return nil
}

func (m *MockPostRepo) Delete(ctx context.Context, id int) error {
	// Заглушка: ничего не делаем, считаем успешным
	return nil
}

func (m *MockPostRepo) Exists(ctx context.Context, id int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.existsErr != nil {
		return false, m.existsErr
	}
	return m.existsVal, nil
}

// сеттер:
func (m *MockPostRepo) SetTotalCount(count int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalCount = count
	m.countErr = err
}

// метод интерфейса:
func (m *MockPostRepo) GetTotalCount(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.totalCount, nil
}
