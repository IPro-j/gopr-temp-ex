package service

/*
import (
	"context"
	"time"

	"advanced-blog-management-system/internal/model"
)

type MockPostRepo struct {
	createdPost   *model.Post
	getByIDPost   *model.Post
	getByIDErr    error
	getAllPosts   []*model.Post
	getAllErr     error
	deletedCalled bool
	createErr     error
	totalCount    int
	countErr      error
	existsVal     bool  // ← добавить
	existsErr     error // ← добавить
}

func NewMockPostRepo() *MockPostRepo {
	return &MockPostRepo{
		existsVal: true, // ← по умолчанию считаем, что пост существует
	}
}

// --- Setters ---

func (m *MockPostRepo) SetCreateError(err error) {
	m.createErr = err
}

func (m *MockPostRepo) SetGetByIDPost(post *model.Post, err error) {
	m.getByIDPost = post
	m.getByIDErr = err
}

func (m *MockPostRepo) SetGetAllPosts(posts []*model.Post) {
	m.getAllPosts = posts
}

func (m *MockPostRepo) SetGetAllError(err error) {
	m.getAllErr = err
}

func (m *MockPostRepo) SetTotalCount(count int, err error) {
	m.totalCount = count
	m.countErr = err
}

// --- Helpers ---

func (m *MockPostRepo) GetCreatedPost() *model.Post {
	return m.createdPost
}

func (m *MockPostRepo) IsDeleteCalled() bool {
	return m.deletedCalled
}

// --- Repository Methods ---

func (m *MockPostRepo) Create(ctx context.Context, post *model.Post) error {
	if m.createErr != nil {
		return m.createErr
	}
	saved := &model.Post{
		ID:        1,
		Title:     post.Title,
		Content:   post.Content,
		Status:    post.Status,
		PublishAt: post.PublishAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.createdPost = saved
	return nil
}

func (m *MockPostRepo) GetByID(ctx context.Context, id int) (*model.Post, error) {
	return m.getByIDPost, m.getByIDErr
}

func (m *MockPostRepo) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	return m.getAllPosts, nil
}

func (m *MockPostRepo) Update(ctx context.Context, post *model.Post) error {
	return nil
}

func (m *MockPostRepo) Delete(ctx context.Context, id int) error {
	m.deletedCalled = true
	return nil
}

func (m *MockPostRepo) GetTotalCount(ctx context.Context) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.totalCount, nil
}

func (m *MockPostRepo) SetExists(val bool, err error) {
	m.existsVal = val
	m.existsErr = err
}

func (m *MockPostRepo) Exists(ctx context.Context, id int) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	return m.existsVal, nil
}

func (m *MockPostRepo) GetByIDPost() *model.Post {
	return m.getByIDPost
}
*/
