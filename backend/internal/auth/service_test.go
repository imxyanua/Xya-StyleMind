package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"stylemind/internal/errs"

	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepository struct {
	usersByEmail map[string]*User
	createErr    error
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{usersByEmail: make(map[string]*User)}
}

func (r *fakeUserRepository) CreateUser(_ context.Context, user *User) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.usersByEmail[strings.ToLower(user.Email)] = user
	return nil
}

func (r *fakeUserRepository) GetUserByEmail(_ context.Context, email string) (*User, error) {
	user, ok := r.usersByEmail[strings.ToLower(email)]
	if !ok {
		return nil, errs.ErrUserNotFound
	}
	return user, nil
}

func (r *fakeUserRepository) GetUserByID(_ context.Context, id string) (*User, error) {
	for _, user := range r.usersByEmail {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, errs.ErrUserNotFound
}

func TestServiceRegister_CreatesUserWithNormalizedEmailAndToken(t *testing.T) {
	repo := newFakeUserRepository()
	service := NewService(repo, "test-secret")

	result, err := service.Register(context.Background(), RegisterRequest{
		Email:    "  USER@Example.COM ",
		FullName: "  Test User ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register error = %v", err)
	}

	userData := result["user"].(map[string]interface{})
	if userData["email"] != "user@example.com" {
		t.Fatalf("email = %v, want user@example.com", userData["email"])
	}
	if userData["role"] != "user" {
		t.Fatalf("role = %v, want user", userData["role"])
	}

	token := result["token"].(string)
	claims, err := ParseToken("test-secret", token)
	if err != nil {
		t.Fatalf("ParseToken error = %v", err)
	}
	if claims.Role != "user" {
		t.Fatalf("claims.Role = %q, want user", claims.Role)
	}

	stored := repo.usersByEmail["user@example.com"]
	if stored == nil {
		t.Fatal("stored user not found")
	}
	if stored.PasswordHash == "password123" {
		t.Fatal("password was stored in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("password123")); err != nil {
		t.Fatalf("stored password hash is invalid: %v", err)
	}
}

func TestServiceRegister_DuplicateEmail(t *testing.T) {
	repo := newFakeUserRepository()
	repo.usersByEmail["user@example.com"] = &User{Email: "user@example.com"}
	service := NewService(repo, "test-secret")

	_, err := service.Register(context.Background(), RegisterRequest{
		Email:    "USER@example.com",
		FullName: "Test User",
		Password: "password123",
	})
	if !errors.Is(err, errs.ErrEmailAlreadyExists) {
		t.Fatalf("err = %v, want ErrEmailAlreadyExists", err)
	}
}

func TestServiceLogin_SuccessWithNormalizedEmail(t *testing.T) {
	repo := newFakeUserRepository()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword error = %v", err)
	}
	repo.usersByEmail["user@example.com"] = &User{
		ID:           "user-1",
		Email:        "user@example.com",
		FullName:     "Test User",
		PasswordHash: string(hash),
		Role:         "admin",
	}
	service := NewService(repo, "test-secret")

	result, err := service.Login(context.Background(), LoginRequest{
		Email:    "  USER@example.com ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login error = %v", err)
	}

	claims, err := ParseToken("test-secret", result["token"].(string))
	if err != nil {
		t.Fatalf("ParseToken error = %v", err)
	}
	if claims.UserID != "user-1" || claims.Role != "admin" {
		t.Fatalf("claims = %+v, want user-1/admin", claims)
	}
}

func TestServiceLogin_InvalidPassword(t *testing.T) {
	repo := newFakeUserRepository()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword error = %v", err)
	}
	repo.usersByEmail["user@example.com"] = &User{
		Email:        "user@example.com",
		PasswordHash: string(hash),
	}
	service := NewService(repo, "test-secret")

	_, err = service.Login(context.Background(), LoginRequest{
		Email:    "user@example.com",
		Password: "wrong-password",
	})
	if !errors.Is(err, errs.ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestServiceLogin_UserNotFound(t *testing.T) {
	service := NewService(newFakeUserRepository(), "test-secret")

	_, err := service.Login(context.Background(), LoginRequest{
		Email:    "missing@example.com",
		Password: "password123",
	})
	if !errors.Is(err, errs.ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}
