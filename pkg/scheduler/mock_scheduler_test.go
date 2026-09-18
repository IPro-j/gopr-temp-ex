package scheduler

import (
	"context"
	"sync"
)

/*type ScheduledPost struct {
	ID        int64
	Title     string
	Status    string     // например, "draft" или "published"
	PublishAt *time.Time // нужен для проверки времени; если у тебя в структуре нет, можно убрать
}
*/
// MockRepo — тестовая реализация ScheduledPostRepository
type MockRepo struct {
	mu            sync.Mutex
	readyPosts    []ScheduledPost
	publishedIDs  map[int64]struct{} // используем map для быстрого поиска
	publishError  error
	getReadyError error
}

func NewMockRepo() *MockRepo {
	return &MockRepo{
		publishedIDs: make(map[int64]struct{}),
	}
}

func (m *MockRepo) SetReadyPosts(posts []ScheduledPost) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readyPosts = posts
}

func (m *MockRepo) SetPublishError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishError = err
}

func (m *MockRepo) SetGetReadyError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getReadyError = err
}

func (m *MockRepo) GetPublishedIDs() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]int64, 0, len(m.publishedIDs))
	for id := range m.publishedIDs {
		ids = append(ids, id)
	}
	return ids
}

func (m *MockRepo) GetPostsReadyToPublish(ctx context.Context) ([]ScheduledPost, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getReadyError != nil {
		return nil, m.getReadyError
	}

	var ready []ScheduledPost
	for _, p := range m.readyPosts {
		// Пост готов, только если он ещё не опубликован
		if _, ok := m.publishedIDs[p.ID]; !ok {
			ready = append(ready, p)
		}
	}
	return ready, nil
}

func (m *MockRepo) MarkAsPublished(ctx context.Context, postID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.publishError != nil {
		return m.publishError
	}
	m.publishedIDs[postID] = struct{}{}
	return nil
}
