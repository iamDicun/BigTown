package usecase

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/crypto/bcrypt"

	"backend/internal/module/auth/entity"
	"backend/internal/module/auth/port"
	userentity "backend/internal/module/user/entity"
)

// ============================================================
// Mocks
// ============================================================

type mockAuthRepo struct {
	findCredentialByUserIDFn   func(ctx context.Context, userID string) (*entity.Credential, error)
	createRefreshTokenFn       func(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error
	findRefreshTokenByHashFn   func(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	revokeRefreshTokenFn       func(ctx context.Context, tokenHash string) error
	blacklistAccessTokenFn     func(ctx context.Context, tokenHash string, expiresAt time.Time) error
	createCredentialWithTxFn   func(ctx context.Context, tx *sql.Tx, userID string, passwordHash string) error
	revokeRefreshTokenWithTxFn func(ctx context.Context, tx *sql.Tx, tokenHash string) error
	createRefreshTokenWithTxFn func(ctx context.Context, tx *sql.Tx, userID string, tokenHash string, expiresAt time.Time) error
}

func (m *mockAuthRepo) FindCredentialByUserID(ctx context.Context, userID string) (*entity.Credential, error) {
	return m.findCredentialByUserIDFn(ctx, userID)
}
func (m *mockAuthRepo) CreateRefreshToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	return m.createRefreshTokenFn(ctx, userID, tokenHash, expiresAt)
}
func (m *mockAuthRepo) FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	return m.findRefreshTokenByHashFn(ctx, tokenHash)
}
func (m *mockAuthRepo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return m.revokeRefreshTokenFn(ctx, tokenHash)
}
func (m *mockAuthRepo) BlacklistAccessToken(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	return m.blacklistAccessTokenFn(ctx, tokenHash, expiresAt)
}
func (m *mockAuthRepo) CreateCredentialWithTx(ctx context.Context, tx *sql.Tx, userID string, passwordHash string) error {
	return m.createCredentialWithTxFn(ctx, tx, userID, passwordHash)
}
func (m *mockAuthRepo) RevokeRefreshTokenWithTx(ctx context.Context, tx *sql.Tx, tokenHash string) error {
	return m.revokeRefreshTokenWithTxFn(ctx, tx, tokenHash)
}
func (m *mockAuthRepo) CreateRefreshTokenWithTx(ctx context.Context, tx *sql.Tx, userID string, tokenHash string, expiresAt time.Time) error {
	return m.createRefreshTokenWithTxFn(ctx, tx, userID, tokenHash, expiresAt)
}

// Unused in these tests — stub
func (m *mockAuthRepo) CreateUserIdentityWithTx(ctx context.Context, tx *sql.Tx, identity entity.UserIdentity) error {
	return nil
}
func (m *mockAuthRepo) FindUserIdentity(ctx context.Context, provider string, tenantID string, externalSubject string) (*entity.UserIdentity, error) {
	return nil, sql.ErrNoRows
}
func (m *mockAuthRepo) UpdatePasswordHash(ctx context.Context, userID string, passwordHash string) error {
	return nil
}
func (m *mockAuthRepo) IsAccessTokenBlacklisted(tokenHash string) (bool, error) {
	return false, nil
}

var _ port.AuthRepository = (*mockAuthRepo)(nil)

type mockUserReader struct {
	findByEmailFn func(ctx context.Context, email string) (*userentity.User, error)
	findByIDFn    func(ctx context.Context, id string) (*userentity.User, error)
}

func (m *mockUserReader) FindByEmail(ctx context.Context, email string) (*userentity.User, error) {
	return m.findByEmailFn(ctx, email)
}
func (m *mockUserReader) FindByID(ctx context.Context, id string) (*userentity.User, error) {
	return m.findByIDFn(ctx, id)
}
func (m *mockUserReader) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, err := m.findByEmailFn(ctx, email)
	return err == nil, nil
}
func (m *mockUserReader) CreateWithTx(ctx context.Context, tx *sql.Tx, fullName string, email string) (*userentity.User, error) {
	return &userentity.User{ID: "user-id", FullName: fullName, Email: email, Role: "User"}, nil
}

var _ port.UserReader = (*mockUserReader)(nil)

// ============================================================
// Helpers
// ============================================================

func bcryptHash(pw string) string {
	b, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b)
}

// newTestUsecase dùng cho test không cần DB transaction
func newTestUsecase(repo port.AuthRepository, reader port.UserReader) *AuthUsecase {
	return &AuthUsecase{
		db:         &sql.DB{},
		authRepo:   repo,
		userReader: reader,
		jwtSecret:  "test-secret",
	}
}

// newTestUsecaseDB dùng cho test cần DB transaction (Register, Refresh)
func newTestUsecaseDB(db *sql.DB, repo port.AuthRepository, reader port.UserReader) *AuthUsecase {
	return &AuthUsecase{
		db:         db,
		authRepo:   repo,
		userReader: reader,
		jwtSecret:  "test-secret",
	}
}

// ============================================================
// Login
// ============================================================

func TestLogin_Success(t *testing.T) {
	ctx := context.Background()

	reader := &mockUserReader{
		findByEmailFn: func(ctx context.Context, email string) (*userentity.User, error) {
			return &userentity.User{ID: "user-1", Email: email, Role: "User", FullName: "Test"}, nil
		},
	}
	repo := &mockAuthRepo{
		findCredentialByUserIDFn: func(ctx context.Context, userID string) (*entity.Credential, error) {
			return &entity.Credential{UserID: userID, PasswordHash: bcryptHash("password123")}, nil
		},
		createRefreshTokenFn: func(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
			return nil
		},
	}
	uc := newTestUsecase(repo, reader)

	result, err := uc.Login(ctx, LoginInput{Email: "test@test.com", Password: "password123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected access token")
	}
	if result.RefreshToken == "" {
		t.Error("expected refresh token")
	}
	if result.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %s", result.TokenType)
	}
	if result.ExpiresIn != int64(accessTokenTTL.Seconds()) {
		t.Errorf("expected ExpiresIn %d, got %d", int64(accessTokenTTL.Seconds()), result.ExpiresIn)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	reader := &mockUserReader{
		findByEmailFn: func(ctx context.Context, email string) (*userentity.User, error) {
			return nil, sql.ErrNoRows
		},
	}
	uc := newTestUsecase(&mockAuthRepo{}, reader)

	_, err := uc.Login(context.Background(), LoginInput{Email: "nobody@test.com", Password: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	reader := &mockUserReader{
		findByEmailFn: func(ctx context.Context, email string) (*userentity.User, error) {
			return &userentity.User{ID: "user-1", Email: email, Role: "User"}, nil
		},
	}
	repo := &mockAuthRepo{
		findCredentialByUserIDFn: func(ctx context.Context, userID string) (*entity.Credential, error) {
			return &entity.Credential{UserID: userID, PasswordHash: bcryptHash("correct")}, nil
		},
	}
	uc := newTestUsecase(repo, reader)

	_, err := uc.Login(context.Background(), LoginInput{Email: "test@test.com", Password: "wrong"})
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestLogin_NoCredential(t *testing.T) {
	reader := &mockUserReader{
		findByEmailFn: func(ctx context.Context, email string) (*userentity.User, error) {
			return &userentity.User{ID: "user-1", Email: email, Role: "User"}, nil
		},
	}
	repo := &mockAuthRepo{
		findCredentialByUserIDFn: func(ctx context.Context, userID string) (*entity.Credential, error) {
			return nil, sql.ErrNoRows
		},
	}
	uc := newTestUsecase(repo, reader)

	_, err := uc.Login(context.Background(), LoginInput{Email: "test@test.com", Password: "x"})
	if err == nil {
		t.Fatal("expected error when credential not found")
	}
}

// ============================================================
// Refresh
// ============================================================

func TestRefresh_Success(t *testing.T) {
	reader := &mockUserReader{
		findByIDFn: func(ctx context.Context, id string) (*userentity.User, error) {
			return &userentity.User{ID: id, FullName: "Test", Email: "t@t.com", Role: "User"}, nil
		},
	}
	repo := &mockAuthRepo{
		findRefreshTokenByHashFn: func(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
			return &entity.RefreshToken{
				ID: "rt-1", UserID: "user-1", TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}, nil
		},
		revokeRefreshTokenWithTxFn: func(ctx context.Context, tx *sql.Tx, tokenHash string) error {
			return nil
		},
		createRefreshTokenWithTxFn: func(ctx context.Context, tx *sql.Tx, userID string, tokenHash string, expiresAt time.Time) error {
			return nil
		},
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectCommit()

	uc := newTestUsecaseDB(db, repo, reader)

	result, err := uc.Refresh(context.Background(), RefreshInput{RefreshToken: "valid-rt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Error("expected tokens")
	}
}

func TestRefresh_TokenNotFound(t *testing.T) {
	repo := &mockAuthRepo{
		findRefreshTokenByHashFn: func(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
			return nil, sql.ErrNoRows
		},
	}
	uc := newTestUsecase(repo, &mockUserReader{})

	_, err := uc.Refresh(context.Background(), RefreshInput{RefreshToken: "unknown"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRefresh_TokenRevoked(t *testing.T) {
	repo := &mockAuthRepo{
		findRefreshTokenByHashFn: func(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
			return &entity.RefreshToken{
				ID: "rt-1", UserID: "user-1", TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(24 * time.Hour),
				RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
			}, nil
		},
	}
	uc := newTestUsecase(repo, &mockUserReader{})

	_, err := uc.Refresh(context.Background(), RefreshInput{RefreshToken: "revoked"})
	if err == nil {
		t.Fatal("expected error for revoked token")
	}
}

// ============================================================
// Logout
// ============================================================

func TestLogout_Success(t *testing.T) {
	repo := &mockAuthRepo{
		findRefreshTokenByHashFn: func(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
			return &entity.RefreshToken{
				ID: "rt-1", UserID: "user-1", TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}, nil
		},
		revokeRefreshTokenFn: func(ctx context.Context, tokenHash string) error { return nil },
		blacklistAccessTokenFn: func(ctx context.Context, tokenHash string, expiresAt time.Time) error {
			return nil
		},
	}
	uc := newTestUsecase(repo, &mockUserReader{})

	result, err := uc.Logout(context.Background(), LogoutInput{RefreshToken: "rt"}, "access-token", time.Now().Add(15*time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message == "" {
		t.Error("expected message")
	}
}

func TestLogout_MissingAccessToken(t *testing.T) {
	uc := newTestUsecase(&mockAuthRepo{}, &mockUserReader{})

	_, err := uc.Logout(context.Background(), LogoutInput{RefreshToken: "rt"}, "", time.Now())
	if err == nil {
		t.Fatal("expected error for missing access token")
	}
}

// ============================================================
// Register
// ============================================================

func TestRegister_Success(t *testing.T) {
	reader := &mockUserReader{
		findByEmailFn: func(ctx context.Context, email string) (*userentity.User, error) {
			return nil, sql.ErrNoRows
		},
	}
	repo := &mockAuthRepo{
		createCredentialWithTxFn: func(ctx context.Context, tx *sql.Tx, userID string, passwordHash string) error {
			return nil
		},
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectCommit()

	uc := newTestUsecaseDB(db, repo, reader)

	result, err := uc.Register(context.Background(), RegisterInput{
		FullName: "New User", Email: "new@test.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Email != "new@test.com" {
		t.Errorf("expected new@test.com, got %s", result.Email)
	}
	if result.ID == "" {
		t.Error("expected user ID")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	reader := &mockUserReader{
		findByEmailFn: func(ctx context.Context, email string) (*userentity.User, error) {
			return &userentity.User{ID: "existing", Email: email}, nil
		},
	}
	uc := newTestUsecase(&mockAuthRepo{}, reader)

	_, err := uc.Register(context.Background(), RegisterInput{
		FullName: "Dup", Email: "dup@test.com", Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}
