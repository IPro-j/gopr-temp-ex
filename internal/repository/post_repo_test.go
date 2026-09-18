package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/pkg/apperr"

	_ "github.com/lib/pq" // или твой драйвер
)

func getTestDB(t *testing.T) *sql.DB {
	dsn := "host=localhost user=bloguser password=blogpassword dbname=blogdb sslmode=disable"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("DB ping failed: %v, check connection string", err)
	}

	// Очищаем таблицы: сначала posts (из‑за FK на users), потом users.
	// CASCADE автоматически удалит зависимые строки, но здесь мы явно перечисляем обе.
	_, err = db.Exec("TRUNCATE public.posts, public.users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	// Создаём тестовых пользователей — покрываем все author_id, которые используются в тестах (1–7)
	for i := 1; i <= 7; i++ {
		username := fmt.Sprintf("user_%d", i)
		email := fmt.Sprintf("user_%d@test.com", i)

		// Пароль можно любой, главное — не пустой, т.к. колонка NOT NULL
		_, err = db.Exec(
			`INSERT INTO public.users (id, username, email, "password") VALUES ($1, $2, $3, $4)`,
			i, username, email, "testpassword123",
		)
		if err != nil {
			t.Fatalf("failed to create test user %d: %v", i, err)
		}
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func TestPostRepo_Create_And_GetByID(t *testing.T) {
	db := getTestDB(t)
	repo := NewPostRepo(db)
	ctx := context.Background()

	post := &model.Post{
		Title:    "My first post",
		Content:  "Hello, world!",
		AuthorID: 1,
		Status:   "draft",
	}

	err := repo.Create(ctx, post)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if post.ID == 0 {
		t.Error("Post ID should be set after Create")
	}

	found, err := repo.GetByID(ctx, int(post.ID))
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if found.Title != post.Title || found.Content != post.Content {
		t.Errorf("Post mismatch: got %+v, want %+v", found, post)
	}
}

func TestPostRepo_GetByID_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewPostRepo(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if !errors.Is(err, apperr.ErrPostNotFound) {
		t.Errorf("Expected ErrPostNotFound, got: %v", err)
	}
}

func TestPostRepo_Exists(t *testing.T) {
	db := getTestDB(t)
	repo := NewPostRepo(db)
	ctx := context.Background()

	ok, err := repo.Exists(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("Should not exist yet")
	}

	post := &model.Post{Title: "Exists test", Content: "x", AuthorID: 2, Status: "draft"}
	if err := repo.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

	ok, err = repo.Exists(ctx, int(post.ID))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("Expected Exists=true after Create")
	}
}

func TestPostRepo_Update(t *testing.T) {
	db := getTestDB(t)
	repo := NewPostRepo(db)
	ctx := context.Background()

	post := &model.Post{Title: "Before", Content: "before", AuthorID: 3, Status: "draft"}
	if err := repo.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

	post.Title = "After"
	post.Content = "after"
	if err := repo.Update(ctx, post); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	updated, err := repo.GetByID(ctx, int(post.ID))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "After" || updated.Content != "after" {
		t.Errorf("Update not persisted: %+v", updated)
	}
}

func TestPostRepo_Delete(t *testing.T) {
	db := getTestDB(t)
	repo := NewPostRepo(db)
	ctx := context.Background()

	post := &model.Post{Title: "To delete", Content: "x", AuthorID: 4, Status: "draft"}
	if err := repo.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

	if err := repo.Delete(ctx, int(post.ID)); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := repo.GetByID(ctx, int(post.ID))
	if !errors.Is(err, apperr.ErrPostNotFound) {
		t.Errorf("Expected NotFound after Delete, got: %v", err)
	}
}

func TestPostRepo_ListByAuthor_And_CountByAuthor(t *testing.T) {
	db := getTestDB(t)
	repo := NewPostRepo(db)
	ctx := context.Background()

	authorID := 5
	for i := 0; i < 7; i++ {
		p := &model.Post{
			Title:    "Post " + string(rune('A'+i)),
			Content:  "content",
			AuthorID: authorID,
			Status:   "draft",
		}
		if err := repo.Create(ctx, p); err != nil {
			t.Fatal(err)
		}
	}

	total, err := repo.CountByAuthor(ctx, authorID)
	if err != nil || total != 7 {
		t.Errorf("CountByAuthor: got %d, want 7", total)
	}

	list, err := repo.ListByAuthor(ctx, authorID, 3, 0)
	if err != nil || len(list) != 3 {
		t.Errorf("ListByAuthor limit=3: got %d items, want 3", len(list))
	}

	page2, err := repo.ListByAuthor(ctx, authorID, 3, 3)
	if err != nil || len(page2) != 3 {
		t.Errorf("ListByAuthor offset=3: got %d items, want 3", len(page2))
	}

	page3, err := repo.ListByAuthor(ctx, authorID, 3, 6)
	if err != nil || len(page3) != 1 {
		t.Errorf("ListByAuthor offset=6: got %d items, want 1", len(page3))
	}
}

func TestPostRepo_GetPostsReadyToPublish(t *testing.T) {
	db := getTestDB(t)
	repo := NewPostRepo(db)
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	// Draft с publish_at в прошлом — должен попасть в выборку
	p1 := &model.Post{Title: "Ready now", Content: "x", AuthorID: 6, Status: "draft", PublishAt: &past}
	// Draft без publish_at — не должен
	p2 := &model.Post{Title: "No time", Content: "x", AuthorID: 6, Status: "draft", PublishAt: nil}
	// Published — не должен (в запросе status='draft')
	p3 := &model.Post{Title: "Already published", Content: "x", AuthorID: 6, Status: "published", PublishAt: &past}
	// Draft с будущим publish_at — не должен
	p4 := &model.Post{Title: "Future", Content: "x", AuthorID: 6, Status: "draft", PublishAt: &future}

	for _, p := range []*model.Post{p1, p2, p3, p4} {
		if err := repo.Create(ctx, p); err != nil {
			t.Fatal(err)
		}
	}

	ready, err := repo.GetPostsReadyToPublish(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 1 || ready[0].Title != "Ready now" {
		t.Errorf("GetPostsReadyToPublish mismatch: %+v", ready)
	}
}

func TestPostRepo_MarkAsPublished(t *testing.T) {
	db := getTestDB(t)
	repo := NewPostRepo(db)
	ctx := context.Background()

	post := &model.Post{Title: "Draft to publish", Content: "x", AuthorID: 7, Status: "draft"}
	if err := repo.Create(ctx, post); err != nil {
		t.Fatal(err)
	}

	if err := repo.MarkAsPublished(ctx, int64(post.ID)); err != nil {
		t.Fatalf("MarkAsPublished failed: %v", err)
	}

	got, err := repo.GetByID(ctx, int(post.ID))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "published" {
		t.Errorf("Status should be published, got: %s", got.Status)
	}

	// Повторный вызов — ничего не должно сломаться
	if err := repo.MarkAsPublished(ctx, int64(got.ID)); err != nil {
		t.Fatalf("Second MarkAsPublished should not fail: %v", err)
	}
}
