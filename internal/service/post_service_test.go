package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"advanced-blog-management-system/internal/model"

	"github.com/stretchr/testify/assert"
)

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
