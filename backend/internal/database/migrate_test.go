package database

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestMigrationFilesAreOrderedAndUnique(t *testing.T) {
	files := migrationFilesForTest(t)

	if len(files) < 2 {
		t.Fatalf("migration count = %d, want at least 2", len(files))
	}

	sorted := append([]string(nil), files...)
	sort.Strings(sorted)
	for i := range files {
		if files[i] != sorted[i] {
			t.Fatalf("migration files are not sorted: %v", files)
		}
	}

	seen := make(map[string]struct{})
	for _, file := range files {
		if _, ok := seen[file]; ok {
			t.Fatalf("duplicate migration file: %s", file)
		}
		seen[file] = struct{}{}
		if !strings.HasSuffix(file, ".sql") {
			t.Fatalf("migration file %s does not end with .sql", file)
		}
	}
}

func TestOrderSchemaHardeningMigrationIsIdempotent(t *testing.T) {
	sql := readMigrationForTest(t, "0002_order_schema_hardening.sql")

	required := []string{
		"IF NOT EXISTS",
		"users_role_check",
		"CREATE INDEX IF NOT EXISTS idx_orders_created_at",
		"CREATE INDEX IF NOT EXISTS idx_orders_user_created_at",
		"CREATE INDEX IF NOT EXISTS idx_orders_status_created_at",
		"CREATE INDEX IF NOT EXISTS idx_cart_items_product_id",
		"CREATE INDEX IF NOT EXISTS idx_order_items_product_id",
	}

	for _, text := range required {
		if !strings.Contains(sql, text) {
			t.Fatalf("migration missing %q", text)
		}
	}
}

func TestInitialSchemaContainsCoreConstraints(t *testing.T) {
	sql := readMigrationForTest(t, "0001_init_schema.sql")

	required := []string{
		"CHECK (role IN ('user', 'admin'))",
		"CHECK (status IN ('pending', 'paid', 'shipping', 'completed', 'cancelled'))",
		"CHECK (price > 0)",
		"CHECK (stock >= 0)",
		"UNIQUE(cart_id, product_id)",
		"REFERENCES users(id)",
		"REFERENCES products(id)",
	}

	for _, text := range required {
		if !strings.Contains(sql, text) {
			t.Fatalf("initial schema missing %q", text)
		}
	}
}

func migrationFilesForTest(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("ReadDir migrations error = %v", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, entry.Name())
	}
	return files
}

func readMigrationForTest(t *testing.T, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("../../migrations", name))
	if err != nil {
		t.Fatalf("ReadFile migration %s error = %v", name, err)
	}
	return string(content)
}
