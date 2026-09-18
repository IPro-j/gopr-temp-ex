package model

import (
	"time"

	"github.com/go-playground/validator/v10"
)

const (
	PostStatusDraft     = "draft"
	PostStatusPublished = "published"
)

// User представляет модель пользователя в системе
type User struct {
	ID        int       `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Post представляет модель поста в блоге
type Post struct {
	ID        int        `json:"id" db:"id"`
	Title     string     `json:"title" db:"title"`
	Content   string     `json:"content" db:"content"`
	AuthorID  int        `json:"author_id" db:"author_id"`
	Status    string     `json:"status" db:"status"`
	PublishAt *time.Time `json:"publish_at,omitempty" db:"publish_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// Comment представляет модель комментария к посту
type Comment struct {
	ID        int       `json:"id" db:"id"`
	Content   string    `json:"content" db:"content"`
	PostID    int       `json:"post_id" db:"post_id"`
	AuthorID  int       `json:"author_id" db:"author_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UserCreateRequest представляет запрос на создание пользователя
type UserCreateRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// UserLoginRequest представляет запрос на вход пользователя
type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// PostCreateRequest представляет запрос на создание поста
type PostCreateRequest struct {
	Title     string     `json:"title" validate:"required,min=1,max=200"`
	Content   string     `json:"content" validate:"required,min=1"`
	PublishAt *time.Time `json:"publish_at,omitempty"`
}

// PostUpdateRequest представляет запрос на обновление поста
type PostUpdateRequest struct {
	Title     string     `json:"title" validate:"required,min=1,max=200"`
	Content   string     `json:"content" validate:"required,min=1"`
	Status    string     `json:"status,omitempty"`
	PublishAt *time.Time `json:"publish_at,omitempty"`
}

// CommentCreateRequest представляет запрос на создание комментария
type CommentCreateRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
	PostID  int    `json:"post_id" validate:"required,gt=0"`
}

// CommentUpdateRequest представляет запрос на обновление комментария
type CommentUpdateRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

// UserResponse - структура для ответа с данными пользователя (без пароля)
type UserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenResponse - структура для ответа с JWT токеном
type TokenResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

// PostResponse - структура для ответа с данными поста
type PostResponse struct {
	ID        int          `json:"id"`
	Title     string       `json:"title"`
	Content   string       `json:"content"`
	Author    UserResponse `json:"author"`
	Status    string       `json:"status"`
	PublishAt *time.Time   `json:"publish_at,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// CommentResponse - структура для ответа с данными комментария
type CommentResponse struct {
	ID        int          `json:"id"`
	Content   string       `json:"content"`
	PostID    int          `json:"post_id"`
	Author    UserResponse `json:"author"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// ToResponse преобразует User в UserResponse (без пароля)
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

func (p *Post) CanBeEditedBy(userID int) bool {
	return p.AuthorID == userID
}

func (p *Post) CanBeDeletedBy(userID int) bool {
	return p.AuthorID == userID
}

func (c *Comment) CanBeEditedBy(userID int) bool {
	return c.AuthorID == userID
}

func (c *Comment) CanBeDeletedBy(userID int) bool {
	return c.AuthorID == userID
}

func (p *Post) IsScheduled() bool {
	return p.Status == PostStatusDraft &&
		p.PublishAt != nil &&
		p.PublishAt.After(time.Now())
}

func (p *Post) ShouldPublishNow() bool {
	return p.Status == PostStatusDraft &&
		p.PublishAt != nil &&
		!p.PublishAt.After(time.Now())
}

var validate = validator.New()

func (r *UserCreateRequest) Validate() error {
	return validate.Struct(r)
}

func (r *UserLoginRequest) Validate() error {
	return validate.Struct(r)
}

func (r *PostCreateRequest) Validate() error {
	return validate.Struct(r)
}

func (r *PostUpdateRequest) Validate() error {
	return validate.Struct(r)
}

func (r *CommentCreateRequest) Validate() error {
	return validate.Struct(r)
}

func (r *CommentUpdateRequest) Validate() error {
	return validate.Struct(r)
}
