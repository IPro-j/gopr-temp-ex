package service

import (
	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/pkg/apperr"
	"advanced-blog-management-system/pkg/auth"
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Mock UserRepository
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	existsByUsername    bool
	existsByUsernameErr error

	existsByEmail    bool
	existsByEmailErr error

	createErr error

	getByEmailUser *model.User
	getByEmailErr  error

	getByIDUser *model.User
	getByIDErr  error

	getByUsernameUser *model.User
	getByUsernameErr  error
}

func (m *mockUserRepo) ExistsByUsername(_ context.Context, _ string) (bool, error) {
	return m.existsByUsername, m.existsByUsernameErr
}

func (m *mockUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return m.existsByEmail, m.existsByEmailErr
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	user.ID = 1 // эмулируем присвоение ID из БД
	return nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return m.getByEmailUser, m.getByEmailErr
}

func (m *mockUserRepo) GetByID(_ context.Context, _ int) (*model.User, error) {
	return m.getByIDUser, m.getByIDErr
}

func (m *mockUserRepo) GetByUsername(_ context.Context, _ string) (*model.User, error) {
	return m.getByUsernameUser, m.getByUsernameErr
}

// ---------------------------------------------------------------------------
// Хелперы
// ---------------------------------------------------------------------------

func newTestService(repo *mockUserRepo) *UserService {
	// time.Hour.Seconds() вернёт int64, приводим к int
	//durationSeconds := int(time.Hour.Seconds())
	jwt, err := auth.NewJWTManager("test-secret-key", 1)

	if err != nil {
		// В тестах это критично: если конструктор падает, тест сразу падает
		panic(err)
	}
	return NewUserService(repo, jwt)
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(h)
}

func validCreateReq() *model.UserCreateRequest {
	return &model.UserCreateRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "!Password123",
	}
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestRegister_Success(t *testing.T) {
	repo := &mockUserRepo{existsByUsername: false, existsByEmail: false}
	svc := newTestService(repo)

	resp, err := svc.Register(context.Background(), validCreateReq())

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}

}

func TestRegister_NilRequest(t *testing.T) {
	repo := &mockUserRepo{}
	svc := newTestService(repo)

	_, err := svc.Register(context.Background(), nil)

	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestRegister_InvalidUsername(t *testing.T) {
	repo := &mockUserRepo{}
	svc := newTestService(repo)

	req := validCreateReq()
	req.Username = "a" // слишком короткий

	_, err := svc.Register(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for invalid username")
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	repo := &mockUserRepo{}
	svc := newTestService(repo)

	req := validCreateReq()
	req.Email = "not-an-email"

	_, err := svc.Register(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestRegister_InvalidPassword(t *testing.T) {
	repo := &mockUserRepo{}
	svc := newTestService(repo)

	req := validCreateReq()
	req.Password = "short"

	_, err := svc.Register(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for invalid password")
	}
}

func TestRegister_UsernameAlreadyExists(t *testing.T) {
	repo := &mockUserRepo{existsByUsername: true}
	svc := newTestService(repo)

	_, err := svc.Register(context.Background(), validCreateReq())

	if !errors.Is(err, apperr.ErrUserAlreadyExists) {
		t.Fatalf("expected ErrUserAlreadyExists, got: %v", err)
	}
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	repo := &mockUserRepo{existsByUsername: false, existsByEmail: true}
	svc := newTestService(repo)

	_, err := svc.Register(context.Background(), validCreateReq())

	if !errors.Is(err, apperr.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got: %v", err)
	}
}

func TestRegister_ExistsByUsernameRepoError(t *testing.T) {
	repo := &mockUserRepo{existsByUsernameErr: errors.New("db connection lost")}
	svc := newTestService(repo)

	_, err := svc.Register(context.Background(), validCreateReq())

	if err == nil {
		t.Fatal("expected error when repo fails on ExistsByUsername")
	}
}

func TestRegister_ExistsByEmailRepoError(t *testing.T) {
	repo := &mockUserRepo{existsByEmailErr: errors.New("db connection lost")}
	svc := newTestService(repo)

	_, err := svc.Register(context.Background(), validCreateReq())

	if err == nil {
		t.Fatal("expected error when repo fails on ExistsByEmail")
	}
}

func TestRegister_CreateRepoError(t *testing.T) {
	repo := &mockUserRepo{createErr: errors.New("insert failed")}
	svc := newTestService(repo)

	_, err := svc.Register(context.Background(), validCreateReq())

	if err == nil {
		t.Fatal("expected error when repo fails on Create")
	}
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	hash := hashPassword(t, "!Password123")
	user := &model.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Password: hash,
	}
	repo := &mockUserRepo{getByEmailUser: user}
	svc := newTestService(repo)

	resp, err := svc.Login(context.Background(), &model.UserLoginRequest{
		Email:    "test@example.com",
		Password: "!Password123",
	})

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockUserRepo{getByEmailErr: apperr.ErrUserNotFound}
	svc := newTestService(repo)

	_, err := svc.Login(context.Background(), &model.UserLoginRequest{
		Email:    "nobody@example.com",
		Password: "!Password123",
	})

	if err == nil {
		t.Fatal("expected error when user not found")
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	hash := hashPassword(t, "!Password123")
	user := &model.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Password: hash,
	}
	repo := &mockUserRepo{getByEmailUser: user}
	svc := newTestService(repo)

	_, err := svc.Login(context.Background(), &model.UserLoginRequest{
		Email:    "test@example.com",
		Password: "WrongPassword123",
	})

	if !errors.Is(err, apperr.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestGetByID_Success(t *testing.T) {
	user := &model.User{ID: 42, Username: "alice", Email: "alice@example.com"}
	repo := &mockUserRepo{getByIDUser: user}
	svc := newTestService(repo)

	result, err := svc.GetByID(context.Background(), 42)

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if result == nil || result.ID != 42 {
		t.Fatalf("expected user with ID 42, got: %+v", result)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &mockUserRepo{getByIDErr: apperr.ErrUserNotFound}
	svc := newTestService(repo)

	_, err := svc.GetByID(context.Background(), 999)

	if err == nil {
		t.Fatal("expected error when user not found")
	}
}

// ---------------------------------------------------------------------------
// GetByEmail
// ---------------------------------------------------------------------------

func TestGetByEmail_Success(t *testing.T) {
	user := &model.User{ID: 7, Username: "bob", Email: "bob@example.com"}
	repo := &mockUserRepo{getByEmailUser: user}
	svc := newTestService(repo)

	result, err := svc.GetByEmail(context.Background(), "bob@example.com")

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if result == nil || result.Email != "bob@example.com" {
		t.Fatalf("expected user with email bob@example.com, got: %+v", result)
	}
}

func TestGetByEmail_NotFound(t *testing.T) {
	repo := &mockUserRepo{getByEmailErr: apperr.ErrUserNotFound}
	svc := newTestService(repo)

	_, err := svc.GetByEmail(context.Background(), "nobody@example.com")

	if err == nil {
		t.Fatal("expected error when user not found")
	}
}

// ---------------------------------------------------------------------------
// ValidateUserCreateRequest
// ---------------------------------------------------------------------------

func TestValidateUserCreateRequest_NilRequest(t *testing.T) {
	err := ValidateUserCreateRequest(nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestValidateUserCreateRequest_Valid(t *testing.T) {
	err := ValidateUserCreateRequest(validCreateReq())
	if err != nil {
		t.Fatalf("expected nil error for valid request, got: %v", err)
	}
}

func TestValidateUserCreateRequest_InvalidUsername(t *testing.T) {
	req := validCreateReq()
	req.Username = ""
	err := ValidateUserCreateRequest(req)
	if err == nil {
		t.Fatal("expected error for empty username")
	}
}

func TestValidateUserCreateRequest_InvalidEmail(t *testing.T) {
	req := validCreateReq()
	req.Email = "bad-email"
	err := ValidateUserCreateRequest(req)
	if !errors.Is(err, apperr.ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got: %v", err)
	}
}

func TestValidateUserCreateRequest_InvalidPassword(t *testing.T) {
	req := validCreateReq()
	req.Password = ""
	err := ValidateUserCreateRequest(req)
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}
