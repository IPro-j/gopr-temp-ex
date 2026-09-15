package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ScheduledPost — минимальная информация о посте для публикации
type ScheduledPost struct {
	ID    int64
	Title string
}

// ScheduledPostRepository — интерфейс, который реализует PostRepo
type ScheduledPostRepository interface {
	GetPostsReadyToPublish(ctx context.Context) ([]ScheduledPost, error)
	MarkAsPublished(ctx context.Context, postID int64) error
}

// Scheduler — планировщик отложенных публикаций
type Scheduler struct {
	repo     ScheduledPostRepository
	logger   *logrus.Entry
	interval time.Duration
	workers  int
	jobs     chan ScheduledPost

	tickerWg sync.WaitGroup // ожидание ticker-горутины
	workerWg sync.WaitGroup // ожидание worker-горутин
}

func NewScheduler(repo ScheduledPostRepository, logger *logrus.Logger, intervalSec int, workers int) *Scheduler {
	return &Scheduler{
		repo:     repo,
		logger:   logger.WithField("component", "scheduler"),
		interval: time.Duration(intervalSec) * time.Second,
		workers:  workers,
		jobs:     make(chan ScheduledPost, workers*4),
	}
}

// Start запускает worker pool и ticker-горутину
func (s *Scheduler) Start(ctx context.Context) {
	for i := 0; i < s.workers; i++ {
		s.workerWg.Add(1)
		go s.worker(i)
	}

	s.tickerWg.Add(1)
	go s.tickerLoop(ctx)

	s.logger.WithFields(logrus.Fields{
		"interval_seconds": s.interval.Seconds(),
		"workers":          s.workers,
	}).Info("scheduler started")
}

// tickerLoop периодически сканирует БД и отправляет посты в канал jobs
func (s *Scheduler) tickerLoop(ctx context.Context) {
	defer s.tickerWg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Сразу проверяем при старте — не ждём первый тик
	s.scan(ctx)

	for {
		select {
		case <-ticker.C:
			s.scan(ctx)
		case <-ctx.Done():
			s.logger.Info("ticker stopped, no new scans will be performed")
			return
		}
	}
}

// scan запрашивает готовые посты и отправляет их в канал
func (s *Scheduler) scan(ctx context.Context) {
	posts, err := s.repo.GetPostsReadyToPublish(ctx)
	if err != nil {
		s.logger.WithError(err).Error("failed to fetch posts ready to publish")
		return
	}

	if len(posts) == 0 {
		return
	}

	s.logger.WithField("count", len(posts)).Info("found posts ready to publish")

	for _, post := range posts {
		select {
		case s.jobs <- post:
			// отправлено воркерам
		case <-ctx.Done():
			s.logger.Info("scan interrupted by context cancellation")
			return
		}
	}
}

// worker читает из канала jobs и публикует посты
func (s *Scheduler) worker(id int) {
	defer s.workerWg.Done()

	s.logger.WithField("worker_id", id).Debug("worker started")

	for post := range s.jobs {
		s.publish(post, id)
	}

	s.logger.WithField("worker_id", id).Debug("worker stopped")
}

// publish публикует один пост с таймаутом и логированием
func (s *Scheduler) publish(post ScheduledPost, workerID int) {
	// Свой контекст с таймаутом — даже при shutdown даём время на завершение
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()

	err := s.repo.MarkAsPublished(ctx, post.ID)
	if err != nil {
		s.logger.WithError(err).
			WithField("post_id", post.ID).
			WithField("title", post.Title).
			WithField("worker_id", workerID).
			Error("failed to publish post")
		return
	}

	s.logger.WithFields(logrus.Fields{
		"post_id":     post.ID,
		"title":       post.Title,
		"worker_id":   workerID,
		"duration_ms": time.Since(start).Milliseconds(),
	}).Info("post published successfully")
}

// Stop корректно останавливает планировщик:
// 1) ждёт завершения ticker-горутины (context уже отменён вызывающим)
// 2) закрывает канал jobs — воркеры дочитывают остаток и выходят
// 3) ждёт завершения всех воркеров
func (s *Scheduler) Stop() {
	s.tickerWg.Wait()

	s.logger.Info("closing jobs channel, waiting for workers to finish remaining jobs")
	close(s.jobs)

	s.workerWg.Wait()
	s.logger.Info("scheduler stopped gracefully")
}
