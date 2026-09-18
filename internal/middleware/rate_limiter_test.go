package middleware

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_GetBucket_Concurrency(t *testing.T) {
	rl := NewRateLimiter(10, 5) // 10 токенов/сек, ёмкость 5
	var wg sync.WaitGroup
	const clients = 10
	const callsPerClient = 5

	for i := 0; i < clients; i++ {
		wg.Add(1)
		clientID := "client-" + string(rune('a'+i))
		go func(id string) {
			defer wg.Done()
			for j := 0; j < callsPerClient; j++ {
				_ = rl.GetBucket(id)
			}
		}(clientID)
	}
	wg.Wait()

	// Проверяем, что бакетов создано ровно столько, сколько клиентов
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	if len(rl.buckets) != clients {
		t.Errorf("expected %d buckets, got %d", clients, len(rl.buckets))
	}
}

func TestRateLimiter_BucketLimitsRequests(t *testing.T) {
	rate := 10.0
	capacity := int64(2)
	rl := NewRateLimiter(rate, capacity)

	clientID := "test-client"
	bucket := rl.GetBucket(clientID)

	// TakeAvailable возвращает int64 — сколько токенов реально забрали
	// 0 означает, что токенов нет
	ok1 := bucket.TakeAvailable(1)
	ok2 := bucket.TakeAvailable(1)
	ok3 := bucket.TakeAvailable(1) // должен быть 0 — токены кончились

	if ok1 != 1 || ok2 != 1 || ok3 != 0 {
		t.Errorf("expected 1,1,0 got %d,%d,%d", ok1, ok2, ok3)
	}

	// Ждём пополнения (1 токен каждые 0.1 сек при rate=10)
	time.Sleep(110 * time.Millisecond)

	ok4 := bucket.TakeAvailable(1) // должен быть 1 — токен пополнился
	if ok4 != 1 {
		t.Error("bucket did not replenish tokens in time")
	}
}
