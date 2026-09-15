package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logger
}

// TestScheduler_PublishesPostsAfterPublishAt — планировщик находит
// посты с publish_at <= now и публикует их
func TestScheduler_PublishesPostsAfterPublishAt(t *testing.T) {
	repo := NewMockRepo()

	posts := []ScheduledPost{
		{ID: 1, Title: "Post 1"},
		{ID: 2, Title: "Post 2"},
		{ID: 3, Title: "Post 3"},
	}
	repo.SetReadyPosts(posts)

	sch := NewScheduler(repo, newTestLogger(), 1, 3) // repo, logger, 1 сек, 3 воркера

	ctx, cancel := context.WithCancel(context.Background())
	sch.Start(ctx)

	time.Sleep(2200 * time.Millisecond) // ждём 2+ тика

	cancel()
	sch.Stop()

	published := repo.GetPublishedIDs()
	require.Len(t, published, 3, "должно быть опубликовано ровно 3 поста")
}

// TestScheduler_NoPostsToPublish — нет постов для публикации
func TestScheduler_NoPostsToPublish(t *testing.T) {
	repo := NewMockRepo()

	sched := NewScheduler(repo, newTestLogger(), 1, 2)

	ctx, cancel := context.WithCancel(context.Background())
	sched.Start(ctx)

	time.Sleep(1200 * time.Millisecond)

	cancel()
	sched.Stop()

	published := repo.GetPublishedIDs()
	if len(published) != 0 {
		t.Errorf("expected 0 published posts, got %d", len(published))
	}
}

// TestScheduler_PublishError — ошибка при публикации не останавливает планировщик
func TestScheduler_PublishError(t *testing.T) {
	repo := NewMockRepo()
	repo.SetReadyPosts([]ScheduledPost{
		{ID: 10, Title: "Failing Post"},
	})
	repo.SetPublishError(errors.New("db connection lost"))

	sched := NewScheduler(repo, newTestLogger(), 1, 2)

	ctx, cancel := context.WithCancel(context.Background())
	sched.Start(ctx)

	time.Sleep(1500 * time.Millisecond)

	cancel()
	sched.Stop()
}

// TestScheduler_GetReadyError — ошибка при получении постов логируется
func TestScheduler_GetReadyError(t *testing.T) {
	repo := NewMockRepo()
	repo.SetGetReadyError(errors.New("query failed"))

	sched := NewScheduler(repo, newTestLogger(), 1, 2)

	ctx, cancel := context.WithCancel(context.Background())
	sched.Start(ctx)

	time.Sleep(1200 * time.Millisecond)

	cancel()
	sched.Stop()

	published := repo.GetPublishedIDs()
	if len(published) != 0 {
		t.Errorf("expected 0 published posts on query error, got %d", len(published))
	}
}

// TestScheduler_GracefulShutdown — воркеры дорабатывают остаток очереди
func TestScheduler_GracefulShutdown(t *testing.T) {
	posts := make([]ScheduledPost, 100)
	for i := range posts {
		posts[i] = ScheduledPost{ID: int64(i + 1), Title: "Post"}
	}

	repo := NewMockRepo()
	repo.SetReadyPosts(posts)

	sched := NewScheduler(repo, newTestLogger(), 1, 2)

	ctx, cancel := context.WithCancel(context.Background())
	sched.Start(ctx)

	time.Sleep(600 * time.Millisecond)

	cancel()
	sched.Stop()

	published := repo.GetPublishedIDs()
	if len(published) != 100 {
		t.Errorf("expected all 100 posts published, got %d", len(published))
	}
}

// TestScheduler_ConcurrentWorkers — несколько воркеров не дублируют публикации
func TestScheduler_ConcurrentWorkers(t *testing.T) {
	const numPosts = 50

	posts := make([]ScheduledPost, numPosts)
	for i := range posts {
		posts[i] = ScheduledPost{ID: int64(i + 1), Title: "Post"}
	}

	repo := &countingRepo{
		posts:     posts, // ← ИСПРАВЛЕНО: передаём посты
		published: make(map[int64]int),
	}

	sched := NewScheduler(repo, newTestLogger(), 1, 5)

	ctx, cancel := context.WithCancel(context.Background())
	sched.Start(ctx)

	time.Sleep(2200 * time.Millisecond) // ждём минимум 2 тика

	cancel()
	sched.Stop()

	if len(repo.published) != numPosts {
		t.Errorf("expected %d unique published posts, got %d", numPosts, len(repo.published))
	}

	for id, count := range repo.published {
		if count > 1 {
			t.Errorf("post %d was published %d times (expected 1)", id, count)
		}
	}
}

// countingRepo — мок, который считает, сколько раз был опубликован каждый пост
type countingRepo struct {
	mu        sync.Mutex
	posts     []ScheduledPost
	published map[int64]int
}

func (c *countingRepo) GetPostsReadyToPublish(ctx context.Context) ([]ScheduledPost, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// ИСПРАВЛЕНО: фильтруем уже опубликованные посты
	var ready []ScheduledPost
	for _, p := range c.posts {
		if _, ok := c.published[p.ID]; !ok {
			ready = append(ready, p)
		}
	}
	return ready, nil
}

func (c *countingRepo) MarkAsPublished(ctx context.Context, postID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.published[postID]++
	return nil
}

func (c *countingRepo) SetPosts(posts []ScheduledPost) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.posts = posts
}
