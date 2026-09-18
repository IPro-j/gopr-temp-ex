package service

import (
	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/pkg/apperr"
	"context"
	"errors"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock CommentRepository
// ---------------------------------------------------------------------------

type mockCommentRepo struct {
	createErr error

	getByIDComment *model.Comment
	getByIDErr     error

	getByPostIDComments []*model.Comment
	getByPostIDErr      error

	getCountByPostIDResult int
	getCountByPostIDErr    error

	updateErr error

	deleteErr error
}

func (m *mockCommentRepo) Create(_ context.Context, comment *model.Comment) error {
	if m.createErr != nil {
		return m.createErr
	}
	comment.ID = 1
	return nil
}

func (m *mockCommentRepo) GetByID(_ context.Context, _ int) (*model.Comment, error) {
	return m.getByIDComment, m.getByIDErr
}

func (m *mockCommentRepo) GetByPostID(_ context.Context, _ int, _ int, _ int) ([]*model.Comment, error) {
	return m.getByPostIDComments, m.getByPostIDErr
}

func (m *mockCommentRepo) GetCountByPostID(_ context.Context, _ int) (int, error) {
	return m.getCountByPostIDResult, m.getCountByPostIDErr
}

func (m *mockCommentRepo) Update(_ context.Context, _ *model.Comment) error {
	return m.updateErr
}

func (m *mockCommentRepo) Delete(_ context.Context, _ int) error {
	return m.deleteErr
}

// ---------------------------------------------------------------------------
// Mock PostRepository (отдельный от post_service_test.go)
// ---------------------------------------------------------------------------

type mockCommentPostRepo struct {
	getByIDPost   *model.Post
	getByIDErr    error
	existsResult  bool
	existsErr     error
	getAllPosts   []*model.Post
	getAllErr     error
	totalCountRes int
	totalCountErr error
}

func (m *mockCommentPostRepo) GetByID(_ context.Context, _ int) (*model.Post, error) {
	return m.getByIDPost, m.getByIDErr
}

func (m *mockCommentPostRepo) Exists(_ context.Context, _ int) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	return m.existsResult, nil
}

func (m *mockCommentPostRepo) GetAll(_ context.Context, _ int, _ int) ([]*model.Post, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	if m.getAllPosts == nil {
		return []*model.Post{}, nil
	}
	return m.getAllPosts, nil
}

func (m *mockCommentPostRepo) GetTotalCount(_ context.Context) (int, error) {
	if m.totalCountErr != nil {
		return 0, m.totalCountErr
	}
	return m.totalCountRes, nil
}

func (m *mockCommentPostRepo) Create(_ context.Context, _ *model.Post) error { return nil }
func (m *mockCommentPostRepo) Update(_ context.Context, _ *model.Post) error { return nil }
func (m *mockCommentPostRepo) Delete(_ context.Context, _ int) error         { return nil }

// ---------------------------------------------------------------------------
// Хелперы
// ---------------------------------------------------------------------------

func newCommentService(cRepo *mockCommentRepo, pRepo *mockCommentPostRepo) *CommentService {
	return NewCommentService(cRepo, pRepo)
}

func validCommentCreateReq() *model.CommentCreateRequest {
	return &model.CommentCreateRequest{
		Content: "This is a test comment",
	}
}

func validCommentUpdateReq() *model.CommentUpdateRequest {
	return &model.CommentUpdateRequest{
		Content: "Updated comment content",
	}
}

func sampleComment(id, postID, authorID int) *model.Comment {
	return &model.Comment{
		ID:       id,
		Content:  "Test comment",
		PostID:   postID,
		AuthorID: authorID,
	}
}

func samplePost(id int) *model.Post {
	return &model.Post{
		ID:       id,
		Title:    "Test Post",
		Content:  "Test content",
		AuthorID: 1,
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCommentService_Create_Success(t *testing.T) {
	cRepo := &mockCommentRepo{}
	pRepo := &mockCommentPostRepo{getByIDPost: samplePost(1), existsResult: true}
	svc := newCommentService(cRepo, pRepo)

	comment, err := svc.Create(context.Background(), 1, 10, validCommentCreateReq())

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if comment == nil {
		t.Fatal("expected non-nil comment")
	}
	if comment.PostID != 1 {
		t.Errorf("expected PostID=1, got %d", comment.PostID)
	}
	if comment.AuthorID != 10 {
		t.Errorf("expected AuthorID=10, got %d", comment.AuthorID)
	}
	if comment.ID != 1 {
		t.Errorf("expected ID=1 (assigned by mock), got %d", comment.ID)
	}
}

func TestCommentService_Create_NilRequest(t *testing.T) {
	cRepo := &mockCommentRepo{}
	pRepo := &mockCommentPostRepo{getByIDPost: samplePost(1), existsResult: true}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.Create(context.Background(), 1, 10, nil)

	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestCommentService_Create_EmptyContent(t *testing.T) {
	cRepo := &mockCommentRepo{}
	pRepo := &mockCommentPostRepo{getByIDPost: samplePost(1), existsResult: true}
	svc := newCommentService(cRepo, pRepo)

	req := validCommentCreateReq()
	req.Content = "   "

	_, err := svc.Create(context.Background(), 1, 10, req)

	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestCommentService_Create_ContentTooLong(t *testing.T) {
	cRepo := &mockCommentRepo{}
	pRepo := &mockCommentPostRepo{getByIDPost: samplePost(1), existsResult: true}
	svc := newCommentService(cRepo, pRepo)

	req := validCommentCreateReq()
	longContent := make([]rune, 1001)
	for i := range longContent {
		longContent[i] = 'a'
	}
	req.Content = string(longContent)

	_, err := svc.Create(context.Background(), 1, 10, req)

	if err == nil {
		t.Fatal("expected error for content > 1000 chars")
	}
}

func TestCommentService_Create_PostNotFound(t *testing.T) {
	cRepo := &mockCommentRepo{}
	pRepo := &mockCommentPostRepo{getByIDErr: apperr.ErrPostNotFound}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.Create(context.Background(), 999, 10, validCommentCreateReq())

	if !errors.Is(err, apperr.ErrPostNotExists) {
		t.Fatalf("expected ErrPostNotExists, got: %v", err)
	}
}

func TestCommentService_Create_PostRepoError(t *testing.T) {
	cRepo := &mockCommentRepo{}
	pRepo := &mockCommentPostRepo{getByIDErr: errors.New("db down")}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.Create(context.Background(), 1, 10, validCommentCreateReq())

	if err == nil {
		t.Fatal("expected error when postRepo fails")
	}
}

func TestCommentService_Create_CommentRepoError(t *testing.T) {
	cRepo := &mockCommentRepo{createErr: errors.New("insert failed")}
	pRepo := &mockCommentPostRepo{getByIDPost: samplePost(1), existsResult: true}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.Create(context.Background(), 1, 10, validCommentCreateReq())

	if err == nil {
		t.Fatal("expected error when commentRepo.Create fails")
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestCommentService_GetByID_Success(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDComment: sampleComment(5, 1, 10)}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	comment, err := svc.GetByID(context.Background(), 5)

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if comment == nil || comment.ID != 5 {
		t.Fatalf("expected comment ID=5, got: %+v", comment)
	}
}

func TestCommentService_GetByID_NotFound(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDErr: apperr.ErrCommentNotFound}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.GetByID(context.Background(), 999)

	if !errors.Is(err, apperr.ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got: %v", err)
	}
}

func TestCommentService_GetByID_RepoError(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDErr: errors.New("db error")}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.GetByID(context.Background(), 1)

	if err == nil {
		t.Fatal("expected error when repo fails")
	}
}

// ---------------------------------------------------------------------------
// GetByPost
// ---------------------------------------------------------------------------

func TestCommentService_GetByPost_Success(t *testing.T) {
	comments := []*model.Comment{
		sampleComment(1, 1, 10),
		sampleComment(2, 1, 20),
	}
	cRepo := &mockCommentRepo{
		getByPostIDComments:    comments,
		getCountByPostIDResult: 2,
	}
	pRepo := &mockCommentPostRepo{getByIDPost: samplePost(1), existsResult: true}
	svc := newCommentService(cRepo, pRepo)

	result, total, err := svc.GetByPost(context.Background(), 1, 20, 0)

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 comments, got %d", len(result))
	}
	if total != 2 {
		t.Errorf("expected total=2, got %d", total)
	}
}

func TestCommentService_GetByPost_Empty(t *testing.T) {
	cRepo := &mockCommentRepo{
		getByPostIDComments:    []*model.Comment{},
		getCountByPostIDResult: 0,
	}
	pRepo := &mockCommentPostRepo{getByIDPost: samplePost(1), existsResult: true}
	svc := newCommentService(cRepo, pRepo)

	result, total, err := svc.GetByPost(context.Background(), 1, 20, 0)

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 comments, got %d", len(result))
	}
	if total != 0 {
		t.Errorf("expected total=0, got %d", total)
	}
}

func TestCommentService_GetByPost_DefaultLimit(t *testing.T) {
	cRepo := &mockCommentRepo{
		getByPostIDComments:    []*model.Comment{},
		getCountByPostIDResult: 0,
	}
	pRepo := &mockCommentPostRepo{getByIDPost: samplePost(1), existsResult: true}
	svc := newCommentService(cRepo, pRepo)

	_, _, err := svc.GetByPost(context.Background(), 1, 0, 0)

	if err != nil {
		t.Fatalf("expected nil error with default limit, got: %v", err)
	}
}

func TestCommentService_GetByPost_MaxLimit(t *testing.T) {
	cRepo := &mockCommentRepo{
		getByPostIDComments:    []*model.Comment{},
		getCountByPostIDResult: 0,
	}
	pRepo := &mockCommentPostRepo{getByIDPost: samplePost(1), existsResult: true}
	svc := newCommentService(cRepo, pRepo)

	_, _, err := svc.GetByPost(context.Background(), 1, 200, 0)

	if err != nil {
		t.Fatalf("expected nil error with capped limit, got: %v", err)
	}
}

func TestCommentService_GetByPost_PostNotFound(t *testing.T) {
	cRepo := &mockCommentRepo{}
	pRepo := &mockCommentPostRepo{getByIDErr: apperr.ErrPostNotFound}
	svc := newCommentService(cRepo, pRepo)

	_, _, err := svc.GetByPost(context.Background(), 999, 20, 0)

	if !errors.Is(err, apperr.ErrPostNotExists) {
		t.Fatalf("expected ErrPostNotExists, got: %v", err)
	}
}

func TestCommentService_GetByPost_PostRepoError(t *testing.T) {
	cRepo := &mockCommentRepo{}
	pRepo := &mockCommentPostRepo{getByIDErr: errors.New("db down")}
	svc := newCommentService(cRepo, pRepo)

	_, _, err := svc.GetByPost(context.Background(), 1, 20, 0)

	if err == nil {
		t.Fatal("expected error when postRepo fails")
	}
}

func TestCommentService_GetByPost_CommentRepoError(t *testing.T) {
	cRepo := &mockCommentRepo{getByPostIDErr: errors.New("query failed")}
	pRepo := &mockCommentPostRepo{getByIDPost: samplePost(1), existsResult: true}
	svc := newCommentService(cRepo, pRepo)

	_, _, err := svc.GetByPost(context.Background(), 1, 20, 0)

	if err == nil {
		t.Fatal("expected error when commentRepo.GetByPostID fails")
	}
}

func TestCommentService_GetByPost_CountError(t *testing.T) {
	cRepo := &mockCommentRepo{
		getByPostIDComments: []*model.Comment{},
		getCountByPostIDErr: errors.New("count failed"),
	}
	pRepo := &mockCommentPostRepo{getByIDPost: samplePost(1), existsResult: true}
	svc := newCommentService(cRepo, pRepo)

	_, _, err := svc.GetByPost(context.Background(), 1, 20, 0)

	if err == nil {
		t.Fatal("expected error when commentRepo.GetCountByPostID fails")
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestCommentService_Update_Success(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDComment: sampleComment(1, 1, 10)}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	comment, err := svc.Update(context.Background(), 1, 10, validCommentUpdateReq())

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if comment == nil {
		t.Fatal("expected non-nil comment")
	}
	if comment.Content != "Updated comment content" {
		t.Errorf("expected updated content, got %q", comment.Content)
	}
}

func TestCommentService_Update_NotFound(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDErr: apperr.ErrCommentNotFound}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.Update(context.Background(), 999, 10, validCommentUpdateReq())

	if !errors.Is(err, apperr.ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got: %v", err)
	}
}

func TestCommentService_Update_Forbidden(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDComment: sampleComment(1, 1, 10)}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.Update(context.Background(), 1, 99, validCommentUpdateReq())

	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got: %v", err)
	}
}

func TestCommentService_Update_NilRequest(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDComment: sampleComment(1, 1, 10)}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.Update(context.Background(), 1, 10, nil)

	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestCommentService_Update_EmptyContent(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDComment: sampleComment(1, 1, 10)}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	req := validCommentUpdateReq()
	req.Content = "   "

	_, err := svc.Update(context.Background(), 1, 10, req)

	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestCommentService_Update_ContentTooLong(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDComment: sampleComment(1, 1, 10)}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	req := validCommentUpdateReq()
	longContent := make([]rune, 1001)
	for i := range longContent {
		longContent[i] = 'x'
	}
	req.Content = string(longContent)

	_, err := svc.Update(context.Background(), 1, 10, req)

	if err == nil {
		t.Fatal("expected error for content > 1000 chars")
	}
}

func TestCommentService_Update_RepoError(t *testing.T) {
	cRepo := &mockCommentRepo{
		getByIDComment: sampleComment(1, 1, 10),
		updateErr:      errors.New("update failed"),
	}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.Update(context.Background(), 1, 10, validCommentUpdateReq())

	if err == nil {
		t.Fatal("expected error when repo.Update fails")
	}
}

func TestCommentService_Update_GetByIDRepoError(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDErr: errors.New("db error")}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.Update(context.Background(), 1, 10, validCommentUpdateReq())

	if err == nil {
		t.Fatal("expected error when repo.GetByID fails")
	}
}

func TestCommentService_Update_NotFoundFromRepo(t *testing.T) {
	cRepo := &mockCommentRepo{
		getByIDComment: sampleComment(1, 1, 10),
		updateErr:      apperr.ErrCommentNotFound,
	}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	_, err := svc.Update(context.Background(), 1, 10, validCommentUpdateReq())

	if !errors.Is(err, apperr.ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound from repo.Update, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestCommentService_Delete_Success(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDComment: sampleComment(1, 1, 10)}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	err := svc.Delete(context.Background(), 1, 10)

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestCommentService_Delete_NotFound(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDErr: apperr.ErrCommentNotFound}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	err := svc.Delete(context.Background(), 999, 10)

	if !errors.Is(err, apperr.ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got: %v", err)
	}
}

func TestCommentService_Delete_Forbidden(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDComment: sampleComment(1, 1, 10)}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	err := svc.Delete(context.Background(), 1, 99)

	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got: %v", err)
	}
}

func TestCommentService_Delete_GetByIDRepoError(t *testing.T) {
	cRepo := &mockCommentRepo{getByIDErr: errors.New("db error")}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	err := svc.Delete(context.Background(), 1, 10)

	if err == nil {
		t.Fatal("expected error when repo.GetByID fails")
	}
}

func TestCommentService_Delete_RepoError(t *testing.T) {
	cRepo := &mockCommentRepo{
		getByIDComment: sampleComment(1, 1, 10),
		deleteErr:      errors.New("delete failed"),
	}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	err := svc.Delete(context.Background(), 1, 10)

	if err == nil {
		t.Fatal("expected error when repo.Delete fails")
	}
}

func TestCommentService_Delete_NotFoundFromRepo(t *testing.T) {
	cRepo := &mockCommentRepo{
		getByIDComment: sampleComment(1, 1, 10),
		deleteErr:      apperr.ErrCommentNotFound,
	}
	pRepo := &mockCommentPostRepo{}
	svc := newCommentService(cRepo, pRepo)

	err := svc.Delete(context.Background(), 1, 10)

	if !errors.Is(err, apperr.ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound from repo.Delete, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// validateCommentCreateRequest
// ---------------------------------------------------------------------------

func TestValidateCommentCreateRequest_NilRequest(t *testing.T) {
	err := validateCommentCreateRequest(nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestValidateCommentCreateRequest_Valid(t *testing.T) {
	err := validateCommentCreateRequest(validCommentCreateReq())
	if err != nil {
		t.Fatalf("expected nil error for valid request, got: %v", err)
	}
}

func TestValidateCommentCreateRequest_EmptyContent(t *testing.T) {
	req := validCommentCreateReq()
	req.Content = ""
	err := validateCommentCreateRequest(req)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestValidateCommentCreateRequest_WhitespaceOnly(t *testing.T) {
	req := validCommentCreateReq()
	req.Content = "   "
	err := validateCommentCreateRequest(req)
	if err == nil {
		t.Fatal("expected error for whitespace-only content")
	}
}

func TestValidateCommentCreateRequest_TooLong(t *testing.T) {
	req := validCommentCreateReq()
	longContent := make([]rune, 1001)
	for i := range longContent {
		longContent[i] = 'a'
	}
	req.Content = string(longContent)
	err := validateCommentCreateRequest(req)
	if err == nil {
		t.Fatal("expected error for content > 1000 chars")
	}
}

func TestValidateCommentCreateRequest_ExactMaxLen(t *testing.T) {
	req := validCommentCreateReq()
	content := make([]rune, 1000)
	for i := range content {
		content[i] = 'a'
	}
	req.Content = string(content)
	err := validateCommentCreateRequest(req)
	if err != nil {
		t.Fatalf("expected nil error for exactly 1000 chars, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// validateCommentUpdateRequest
// ---------------------------------------------------------------------------

func TestValidateCommentUpdateRequest_NilRequest(t *testing.T) {
	err := validateCommentUpdateRequest(nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestValidateCommentUpdateRequest_Valid(t *testing.T) {
	err := validateCommentUpdateRequest(validCommentUpdateReq())
	if err != nil {
		t.Fatalf("expected nil error for valid request, got: %v", err)
	}
}

func TestValidateCommentUpdateRequest_EmptyContent(t *testing.T) {
	req := validCommentUpdateReq()
	req.Content = ""
	err := validateCommentUpdateRequest(req)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestValidateCommentUpdateRequest_TooLong(t *testing.T) {
	req := validCommentUpdateReq()
	longContent := make([]rune, 1001)
	for i := range longContent {
		longContent[i] = 'x'
	}
	req.Content = string(longContent)
	err := validateCommentUpdateRequest(req)
	if err == nil {
		t.Fatal("expected error for content > 1000 chars")
	}
}
