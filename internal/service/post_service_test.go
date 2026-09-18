package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/pkg/apperr"

	"github.com/stretchr/testify/assert"
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

func TestPostService_Create_ImmediatePublish(t *testing.T) {
	repo := NewMockPostRepo()
	svc := NewPostService(repo, nil)

	req := &model.PostCreateRequest{
		Title:   "Test Post",
		Content: "Content",
	}

	post, err := svc.Create(context.Background(), 1, req)
	assert.NoError(t, err)
	assert.Equal(t, "published", post.Status)
	assert.Nil(t, post.PublishAt)

	saved := repo.GetCreatedPost()
	assert.NotNil(t, saved)
	assert.Equal(t, "published", saved.Status)
	assert.NotNil(t, saved.CreatedAt)
	assert.True(t, !saved.CreatedAt.IsZero())
}

func TestPostService_Create_ScheduledPost(t *testing.T) {
	repo := NewMockPostRepo()
	svc := NewPostService(repo, nil)

	futureTime := time.Now().Add(24 * time.Hour)
	req := &model.PostCreateRequest{
		Title:     "Scheduled Post",
		Content:   "Content",
		PublishAt: &futureTime,
	}

	post, err := svc.Create(context.Background(), 1, req)
	assert.NoError(t, err)
	assert.Equal(t, "draft", post.Status)

	assert.NotNil(t, post.PublishAt)
	// Используем WithinDuration, чтобы не зависеть от миллисекунд
	assert.WithinDuration(t, futureTime, *post.PublishAt, 5*time.Second)

	saved := repo.GetCreatedPost()
	assert.NotNil(t, saved)
	assert.Equal(t, "draft", saved.Status)
}

func TestPostService_Create_PastPublishAt(t *testing.T) {
	repo := NewMockPostRepo()
	svc := NewPostService(repo, nil)

	pastTime := time.Now().Add(-1 * time.Hour)
	req := &model.PostCreateRequest{
		Title:     "Past Post",
		Content:   "Content",
		PublishAt: &pastTime,
	}

	post, err := svc.Create(context.Background(), 1, req)
	assert.NoError(t, err)
	assert.Equal(t, "published", post.Status)
	assert.Nil(t, post.PublishAt) // publish_at сброшен, т.к. время в прошлом

	saved := repo.GetCreatedPost()
	assert.NotNil(t, saved)
	assert.Equal(t, "published", saved.Status)
	assert.Nil(t, saved.PublishAt)
}

func TestPostService_Create_EmptyTitle(t *testing.T) {
	repo := NewMockPostRepo()
	svc := NewPostService(repo, nil)

	req := &model.PostCreateRequest{
		Title:   "   ",
		Content: "Content",
	}

	_, err := svc.Create(context.Background(), 1, req)
	assert.Error(t, err)
	// Если у тебя есть доменная ошибка валидации, лучше проверять конкретно её:
	// assert.ErrorIs(t, err, ErrValidation)
}

func TestPostService_Create_EmptyContent(t *testing.T) {
	repo := NewMockPostRepo()
	svc := NewPostService(repo, nil)

	req := &model.PostCreateRequest{
		Title:   "Title",
		Content: "",
	}

	_, err := svc.Create(context.Background(), 1, req)
	assert.Error(t, err)
}

func TestPostService_Create_RepoError(t *testing.T) {
	repo := NewMockPostRepo()
	repo.SetCreateError(errors.New("db down"))
	svc := NewPostService(repo, nil)

	req := &model.PostCreateRequest{
		Title:   "Title",
		Content: "Content",
	}

	_, err := svc.Create(context.Background(), 1, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

func TestPostService_Update_Success(t *testing.T) {
	repo := NewMockPostRepo()
	svc := NewPostService(repo, nil)

	// Сначала создадим пост, чтобы он существовал
	existingPost := &model.Post{
		ID:       1,
		Title:    "Old Title",
		Content:  "Old Content",
		AuthorID: 1, // Тот же автор, что и в запросе ниже
		Status:   model.PostStatusDraft,
	}
	repo.SetGetByIDPost(existingPost, nil) // Настраиваем мок, чтобы GetByID возвращал этот пост

	req := &model.PostUpdateRequest{
		Title:   "New Title",
		Content: "New Content",
	}

	updated, err := svc.Update(context.Background(), 1, 1, req) // userID=1, postID=1
	assert.NoError(t, err)
	assert.Equal(t, "New Title", updated.Title)
	assert.Equal(t, 1, updated.AuthorID)
}

func TestPostService_Update_Forbidden_OtherUser(t *testing.T) {
	repo := NewMockPostRepo()
	svc := NewPostService(repo, nil)

	otherAuthorPost := &model.Post{
		ID:       99,
		Title:    "Other User's Post",
		Content:  "Content",
		AuthorID: 99, // Автор поста — 99
		Status:   model.PostStatusDraft,
	}
	repo.SetGetByIDPost(otherAuthorPost, nil)

	t.Logf("DEBUG: repo.getByIDPost.AuthorID = %d", repo.GetByIDPost().AuthorID)

	req := &model.PostUpdateRequest{
		Title:   "Trying to hack",
		Content: "Content",
	}

	_, err := svc.Update(context.Background(), 99, 42, req) // Пользователь 42 пытается править пост 99
	assert.Error(t, err)

	// ВАЖНО: Проверяй конкретную доменную ошибку, а не просто "есть ошибка"
	// Если у тебя есть ErrForbidden:
	// assert.ErrorIs(t, err, ErrForbidden)
	// Или хотя бы проверяй текст, если доменных ошибок пока нет:
	assert.Contains(t, err.Error(), "forbidden")
}

func TestPostService_Update_NotFound(t *testing.T) {
	repo := NewMockPostRepo()
	repo.SetGetByIDPost(nil, errors.New("not found")) // Настраиваем ошибку для GetByID
	svc := NewPostService(repo, nil)

	req := &model.PostUpdateRequest{
		Title:   "Title",
		Content: "Content",
	}

	_, err := svc.Update(context.Background(), 1, 999, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPostService_Delete_Success(t *testing.T) {
	repo := NewMockPostRepo()
	post := &model.Post{ID: 1, AuthorID: 1, Status: model.PostStatusPublished}
	repo.SetGetByIDPost(post, nil)
	svc := NewPostService(repo, nil)

	err := svc.Delete(context.Background(), 1, 1)
	assert.NoError(t, err)
	assert.True(t, repo.IsDeleteCalled())
}

func TestPostService_Delete_Forbidden(t *testing.T) {
	repo := NewMockPostRepo()
	post := &model.Post{ID: 5, AuthorID: 99, Status: model.PostStatusPublished}
	repo.SetGetByIDPost(post, nil)
	svc := NewPostService(repo, nil)

	err := svc.Delete(context.Background(), 5, 1) // postID=5, userID=1
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrForbidden)
	assert.False(t, repo.IsDeleteCalled()) // Delete в репо не должен вызваться
}

func TestPostService_Delete_NotFound(t *testing.T) {
	repo := NewMockPostRepo()
	repo.SetGetByIDPost(nil, errors.New("not found"))
	svc := NewPostService(repo, nil)

	err := svc.Delete(context.Background(), 999, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPostService_GetByID_Success(t *testing.T) {
	repo := NewMockPostRepo()
	post := &model.Post{ID: 1, Title: "Test", AuthorID: 1, Status: model.PostStatusPublished}
	repo.SetGetByIDPost(post, nil)
	svc := NewPostService(repo, nil)

	got, err := svc.GetByID(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, "Test", got.Title)
}

func TestPostService_GetByID_NotFound(t *testing.T) {
	repo := NewMockPostRepo()
	repo.SetGetByIDPost(nil, errors.New("not found"))
	svc := NewPostService(repo, nil)

	_, err := svc.GetByID(context.Background(), 999)
	assert.Error(t, err)
}

func TestPostService_GetAll_Success(t *testing.T) {
	repo := NewMockPostRepo()
	posts := []*model.Post{
		{ID: 1, Title: "Post 1"},
		{ID: 2, Title: "Post 2"},
	}
	repo.SetGetAllPosts(posts)
	repo.SetTotalCount(2, nil)
	svc := NewPostService(repo, nil)

	got, total, err := svc.GetAll(context.Background(), 10, 0)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, 2, total)
}

func TestPostService_GetAll_Empty(t *testing.T) {
	repo := NewMockPostRepo()
	repo.SetGetAllPosts([]*model.Post{})
	repo.SetTotalCount(0, nil)
	svc := NewPostService(repo, nil)

	got, total, err := svc.GetAll(context.Background(), 10, 0)
	assert.NoError(t, err)
	assert.Empty(t, got)
	assert.Zero(t, total)
}

func TestPostService_GetAll_RepoError(t *testing.T) {
	repo := NewMockPostRepo()
	repo.SetGetAllError(errors.New("db timeout"))
	svc := NewPostService(repo, nil)

	_, _, err := svc.GetAll(context.Background(), 10, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db timeout")
}

func TestPostService_GetAll_Pagination(t *testing.T) {
	repo := NewMockPostRepo()
	allPosts := make([]*model.Post, 5)
	for i := 0; i < 5; i++ {
		allPosts[i] = &model.Post{ID: i + 1}
	}
	// Репозиторий возвращает только 2 поста для страницы (offset=2, limit=2)
	repo.SetGetAllPosts(allPosts[2:4])
	repo.SetTotalCount(5, nil)
	svc := NewPostService(repo, nil)

	got, total, err := svc.GetAll(context.Background(), 2, 2) // limit=2, offset=2
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, 5, total) // общее количество всё равно 5
}
