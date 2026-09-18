package model

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPost_CanBeEditedBy(t *testing.T) {
	t.Run("author can edit", func(t *testing.T) {
		post := &Post{AuthorID: 42}
		assert.True(t, post.CanBeEditedBy(42))
	})

	t.Run("other user cannot edit", func(t *testing.T) {
		post := &Post{AuthorID: 42}
		assert.False(t, post.CanBeEditedBy(99))
	})
}

func TestPost_CanBeDeletedBy(t *testing.T) {
	t.Run("author can delete", func(t *testing.T) {
		post := &Post{AuthorID: 42}
		assert.True(t, post.CanBeDeletedBy(42))
	})

	t.Run("other user cannot delete", func(t *testing.T) {
		post := &Post{AuthorID: 42}
		assert.False(t, post.CanBeDeletedBy(99))
	})
}

func TestPost_IsScheduled(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	t.Run("draft + future PublishAt → scheduled", func(t *testing.T) {
		post := &Post{
			Status:    PostStatusDraft,
			PublishAt: &future,
		}
		assert.True(t, post.IsScheduled())
	})

	t.Run("published + future PublishAt → not scheduled", func(t *testing.T) {
		post := &Post{
			Status:    PostStatusPublished,
			PublishAt: &future,
		}
		assert.False(t, post.IsScheduled())
	})

	t.Run("draft + nil PublishAt → not scheduled", func(t *testing.T) {
		post := &Post{
			Status: PostStatusDraft,
		}
		assert.False(t, post.IsScheduled())
	})

	t.Run("draft + past PublishAt → not scheduled", func(t *testing.T) {
		post := &Post{
			Status:    PostStatusDraft,
			PublishAt: &past,
		}
		assert.False(t, post.IsScheduled())
	})
}

func TestPost_ShouldPublishNow(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	t.Run("draft + past/now PublishAt → should publish now", func(t *testing.T) {
		// past
		postPast := &Post{
			Status:    PostStatusDraft,
			PublishAt: &past,
		}
		assert.True(t, postPast.ShouldPublishNow())

		// exactly now (nil check + time comparison)
		publishAtNow := now
		postNow := &Post{
			Status:    PostStatusDraft,
			PublishAt: &publishAtNow,
		}
		assert.True(t, postNow.ShouldPublishNow())
	})

	t.Run("draft + future PublishAt → should not publish now", func(t *testing.T) {
		post := &Post{
			Status:    PostStatusDraft,
			PublishAt: &future,
		}
		assert.False(t, post.ShouldPublishNow())
	})

	t.Run("published + past PublishAt → should not publish now", func(t *testing.T) {
		post := &Post{
			Status:    PostStatusPublished,
			PublishAt: &past,
		}
		assert.False(t, post.ShouldPublishNow())
	})

	t.Run("draft + nil PublishAt → should not publish now", func(t *testing.T) {
		post := &Post{
			Status: PostStatusDraft,
		}
		assert.False(t, post.ShouldPublishNow())
	})
}

func TestUser_ToResponse(t *testing.T) {
	user := &User{
		ID:        123,
		Username:  "alice",
		Email:     "alice@example.com",
		Password:  "secret", // должно быть исключено в ответе
		CreatedAt: time.Date(2024, 1, 15, 10, 20, 30, 0, time.UTC),
	}

	resp := user.ToResponse()

	assert.Equal(t, 123, resp.ID)
	assert.Equal(t, "alice", resp.Username)
	assert.Equal(t, "alice@example.com", resp.Email)
	assert.Equal(t, user.CreatedAt, resp.CreatedAt)
	// Password отсутствует в структуре UserResponse — это правильно
}

func TestRequestValidation(t *testing.T) {
	t.Run("PostCreateRequest valid", func(t *testing.T) {
		req := &PostCreateRequest{
			Title:   "Good title",
			Content: "Good content",
		}
		err := req.Validate()
		require.NoError(t, err)
	})

	t.Run("PostCreateRequest invalid title", func(t *testing.T) {
		req := &PostCreateRequest{
			Title:   "", // required
			Content: "Content",
		}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Title")
	})

	t.Run("PostCreateRequest invalid content", func(t *testing.T) {
		req := &PostCreateRequest{
			Title:   "Title",
			Content: "", // required
		}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Content")
	})

	t.Run("PostUpdateRequest valid with empty optional fields", func(t *testing.T) {
		req := &PostUpdateRequest{
			Title:   "New title",
			Content: "New content",
			// Status и PublishAt могут быть пустыми — они omitempty и не обязательны
		}
		err := req.Validate()
		require.NoError(t, err)
	})

	t.Run("PostUpdateRequest invalid title", func(t *testing.T) {
		req := &PostUpdateRequest{
			Title:   "",
			Content: "Content",
		}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Title")
	})

	t.Run("UserCreateRequest valid", func(t *testing.T) {
		req := &UserCreateRequest{
			Username: "alice",
			Email:    "alice@example.com",
			Password: "secret123",
		}
		err := req.Validate()
		require.NoError(t, err)
	})

	t.Run("UserCreateRequest invalid email", func(t *testing.T) {
		req := &UserCreateRequest{
			Username: "alice",
			Email:    "not-an-email",
			Password: "secret123",
		}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Email")
	})
}

func TestComment_Permissions(t *testing.T) {
	t.Run("comment author can edit", func(t *testing.T) {
		c := &Comment{AuthorID: 42}
		assert.True(t, c.CanBeEditedBy(42))
	})

	t.Run("comment other user cannot edit", func(t *testing.T) {
		c := &Comment{AuthorID: 42}
		assert.False(t, c.CanBeEditedBy(99))
	})

	t.Run("comment author can delete", func(t *testing.T) {
		c := &Comment{AuthorID: 42}
		assert.True(t, c.CanBeDeletedBy(42))
	})

	t.Run("comment other user cannot delete", func(t *testing.T) {
		c := &Comment{AuthorID: 42}
		assert.False(t, c.CanBeDeletedBy(99))
	})
}

func TestRequestValidation_EdgeCases(t *testing.T) {
	// Граница min=3 для Username
	t.Run("UserCreateRequest username min length", func(t *testing.T) {
		req := &UserCreateRequest{
			Username: "ali", // ровно 3 символа
			Email:    "test@example.com",
			Password: "secret123",
		}
		require.NoError(t, req.Validate())
	})

	// Граница max=50 для Username
	t.Run("UserCreateRequest username max length", func(t *testing.T) {
		maxLen := strings.Repeat("a", 50)
		req := &UserCreateRequest{
			Username: maxLen,
			Email:    "test@example.com",
			Password: "secret123",
		}
		require.NoError(t, req.Validate())
	})

	// Слишком длинный username
	t.Run("UserCreateRequest username too long", func(t *testing.T) {
		tooLong := strings.Repeat("a", 51)
		req := &UserCreateRequest{
			Username: tooLong,
			Email:    "test@example.com",
			Password: "secret123",
		}
		require.Error(t, req.Validate())
	})
}
