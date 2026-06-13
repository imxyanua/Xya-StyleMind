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

func TestOrderCheckoutDetailsMigrationIsIdempotent(t *testing.T) {
	sql := readMigrationForTest(t, "0008_add_order_checkout_details.sql")

	required := []string{
		"ADD COLUMN IF NOT EXISTS recipient_name",
		"ADD COLUMN IF NOT EXISTS phone",
		"ADD COLUMN IF NOT EXISTS address_line",
		"orders_shipping_method_check",
		"orders_payment_method_check",
		"CREATE INDEX IF NOT EXISTS idx_orders_shipping_method",
		"CREATE INDEX IF NOT EXISTS idx_orders_payment_method",
	}

	for _, text := range required {
		if !strings.Contains(sql, text) {
			t.Fatalf("checkout details migration missing %q", text)
		}
	}
}

func TestUserAddressesMigrationHasDefaultConstraint(t *testing.T) {
	sql := readMigrationForTest(t, "0009_create_user_addresses.sql")

	required := []string{
		"CREATE TABLE IF NOT EXISTS user_addresses",
		"REFERENCES users(id) ON DELETE CASCADE",
		"CREATE INDEX IF NOT EXISTS idx_user_addresses_user_id",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_user_addresses_one_default",
		"WHERE is_default = TRUE",
	}

	for _, text := range required {
		if !strings.Contains(sql, text) {
			t.Fatalf("user addresses migration missing %q", text)
		}
	}
}

func TestOrderPaymentStatusMigrationIsIdempotent(t *testing.T) {
	sql := readMigrationForTest(t, "0010_add_order_payment_status.sql")

	required := []string{
		"ADD COLUMN IF NOT EXISTS payment_status",
		"orders_payment_status_check",
		"status IN ('paid', 'shipping', 'completed') THEN 'paid'",
		"CREATE INDEX IF NOT EXISTS idx_orders_payment_status",
		"CREATE INDEX IF NOT EXISTS idx_orders_payment_status_created_at",
	}

	for _, text := range required {
		if !strings.Contains(sql, text) {
			t.Fatalf("payment status migration missing %q", text)
		}
	}
}

func TestReturnRequestsMigrationHasConstraints(t *testing.T) {
	sql := readMigrationForTest(t, "0011_create_return_requests.sql")

	required := []string{
		"CREATE TABLE IF NOT EXISTS return_requests",
		"order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE",
		"user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"CHECK (status IN ('requested', 'approved', 'rejected', 'cancelled'))",
		"CREATE INDEX IF NOT EXISTS idx_return_requests_user_created_at",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_return_requests_active_order",
		"WHERE status IN ('requested', 'approved')",
	}

	for _, text := range required {
		if !strings.Contains(sql, text) {
			t.Fatalf("return requests migration missing %q", text)
		}
	}
}

func TestCouponsMigrationHasConstraints(t *testing.T) {
	sql := readMigrationForTest(t, "0012_create_coupons.sql")

	required := []string{
		"CREATE TABLE IF NOT EXISTS coupons",
		"CHECK (type IN ('percent', 'fixed'))",
		"CHECK (type <> 'percent' OR value <= 100)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_coupons_code_lower",
		"ADD COLUMN IF NOT EXISTS subtotal_amount",
		"ADD COLUMN IF NOT EXISTS discount_amount",
		"ADD COLUMN IF NOT EXISTS coupon_id UUID REFERENCES coupons(id) ON DELETE SET NULL",
		"ADD COLUMN IF NOT EXISTS coupon_code",
		"CREATE INDEX IF NOT EXISTS idx_orders_coupon_id",
	}

	for _, text := range required {
		if !strings.Contains(sql, text) {
			t.Fatalf("coupons migration missing %q", text)
		}
	}
}

func TestNotificationsMigrationHasIndexes(t *testing.T) {
	sql := readMigrationForTest(t, "0013_create_notifications.sql")

	required := []string{
		"CREATE TABLE IF NOT EXISTS notifications",
		"user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"metadata JSONB NOT NULL DEFAULT '{}'::jsonb",
		"read_at TIMESTAMPTZ",
		"CREATE INDEX IF NOT EXISTS idx_notifications_user_created_at",
		"CREATE INDEX IF NOT EXISTS idx_notifications_user_read_created_at",
	}

	for _, text := range required {
		if !strings.Contains(sql, text) {
			t.Fatalf("notifications migration missing %q", text)
		}
	}
}

func TestNotificationPreferencesMigrationHasDefaults(t *testing.T) {
	sql := readMigrationForTest(t, "0014_create_notification_preferences.sql")

	required := []string{
		"CREATE TABLE IF NOT EXISTS notification_preferences",
		"user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE",
		"order_updates_enabled BOOLEAN NOT NULL DEFAULT TRUE",
		"payment_updates_enabled BOOLEAN NOT NULL DEFAULT TRUE",
		"return_updates_enabled BOOLEAN NOT NULL DEFAULT TRUE",
		"promotion_enabled BOOLEAN NOT NULL DEFAULT TRUE",
		"CREATE INDEX IF NOT EXISTS idx_notification_preferences_updated_at",
	}

	for _, text := range required {
		if !strings.Contains(sql, text) {
			t.Fatalf("notification preferences migration missing %q", text)
		}
	}
}

func TestInventoryReservationsMigrationHasConstraints(t *testing.T) {
	sql := readMigrationForTest(t, "0015_create_inventory_reservations.sql")

	required := []string{
		"CREATE TABLE IF NOT EXISTS inventory_reservations",
		"user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE",
		"quantity INTEGER NOT NULL CHECK (quantity > 0)",
		"expires_at TIMESTAMPTZ NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_inventory_reservations_user_expires_at",
		"CREATE INDEX IF NOT EXISTS idx_inventory_reservations_product_expires_at",
	}

	for _, text := range required {
		if !strings.Contains(sql, text) {
			t.Fatalf("inventory reservations migration missing %q", text)
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
