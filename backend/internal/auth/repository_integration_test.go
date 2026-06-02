//go:build integration

package auth

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"stylemind/internal/database"
	"stylemind/internal/errs"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthRepositoryIntegration(t *testing.T) {
	ctx, db := openAuthIntegrationDB(t)
	repo := NewRepository(db)

	userID := uuid.NewString()
	email := "repo-" + userID + "@example.com"
	defer cleanupAuthIntegrationUser(ctx, t, db, userID)

	user := &User{
		ID:           userID,
		Email:        email,
		FullName:     "Repository User",
		PasswordHash: "hashed-password",
		Role:         "user",
	}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser error = %v", err)
	}
	if user.CreatedAt.IsZero() || user.UpdatedAt.IsZero() {
		t.Fatal("CreateUser did not populate timestamps")
	}

	byEmail, err := repo.GetUserByEmail(ctx, strings.ToUpper(email))
	if err != nil {
		t.Fatalf("GetUserByEmail error = %v", err)
	}
	if byEmail.ID != userID || byEmail.Email != email {
		t.Fatalf("byEmail = %+v, want id=%s email=%s", byEmail, userID, email)
	}

	byID, err := repo.GetUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserByID error = %v", err)
	}
	if byID.Email != email {
		t.Fatalf("byID.Email = %s, want %s", byID.Email, email)
	}

	_, err = repo.GetUserByEmail(ctx, "missing-"+email)
	if !errors.Is(err, errs.ErrUserNotFound) {
		t.Fatalf("missing user err = %v, want ErrUserNotFound", err)
	}

	duplicate := &User{
		ID:           uuid.NewString(),
		Email:        email,
		FullName:     "Duplicate User",
		PasswordHash: "hashed-password",
		Role:         "user",
	}
	if err := repo.CreateUser(ctx, duplicate); err == nil {
		t.Fatal("CreateUser duplicate email expected error, got nil")
	}
}

func TestAuthRepositoryRejectsInvalidRoleIntegration(t *testing.T) {
	ctx, db := openAuthIntegrationDB(t)
	repo := NewRepository(db)

	userID := uuid.NewString()
	defer cleanupAuthIntegrationUser(ctx, t, db, userID)

	user := &User{
		ID:           userID,
		Email:        "invalid-role-" + userID + "@example.com",
		FullName:     "Invalid Role User",
		PasswordHash: "hashed-password",
		Role:         "superadmin",
	}
	if err := repo.CreateUser(ctx, user); err == nil {
		t.Fatal("CreateUser invalid role expected constraint error, got nil")
	}
}

func openAuthIntegrationDB(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New error = %v", err)
	}
	t.Cleanup(db.Close)

	if err := database.RunMigrations(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("RunMigrations error = %v", err)
	}
	return ctx, db
}

func cleanupAuthIntegrationUser(ctx context.Context, t *testing.T, db *pgxpool.Pool, userID string) {
	t.Helper()

	_, _ = db.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
}
