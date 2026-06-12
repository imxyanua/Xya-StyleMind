package order

import "testing"

func TestIsValidStatus(t *testing.T) {
	valid := []string{StatusPending, StatusPaid, StatusShipping, StatusCompleted, StatusCancelled}
	for _, status := range valid {
		if !IsValidStatus(status) {
			t.Fatalf("IsValidStatus(%q) = false, want true", status)
		}
	}

	if IsValidStatus("shipped") {
		t.Fatal("IsValidStatus(shipped) = true, want false")
	}
}

func TestCanTransition_AllowsExpectedOrderFlow(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
	}{
		{name: "pending to paid", current: StatusPending, target: StatusPaid},
		{name: "paid to shipping", current: StatusPaid, target: StatusShipping},
		{name: "shipping to completed", current: StatusShipping, target: StatusCompleted},
		{name: "pending to cancelled", current: StatusPending, target: StatusCancelled},
		{name: "paid to cancelled", current: StatusPaid, target: StatusCancelled},
		{name: "pending idempotent", current: StatusPending, target: StatusPending},
		{name: "paid idempotent", current: StatusPaid, target: StatusPaid},
		{name: "shipping idempotent", current: StatusShipping, target: StatusShipping},
		{name: "completed idempotent", current: StatusCompleted, target: StatusCompleted},
		{name: "cancelled idempotent", current: StatusCancelled, target: StatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !CanTransition(tt.current, tt.target) {
				t.Fatalf("CanTransition(%q, %q) = false, want true", tt.current, tt.target)
			}
		})
	}
}

func TestCanTransition_BlocksInvalidOrderFlow(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
	}{
		{name: "pending cannot skip to shipping", current: StatusPending, target: StatusShipping},
		{name: "pending cannot skip to completed", current: StatusPending, target: StatusCompleted},
		{name: "paid cannot go back to pending", current: StatusPaid, target: StatusPending},
		{name: "shipping cannot go back to paid", current: StatusShipping, target: StatusPaid},
		{name: "completed cannot be cancelled", current: StatusCompleted, target: StatusCancelled},
		{name: "cancelled cannot be paid", current: StatusCancelled, target: StatusPaid},
		{name: "cancelled cannot be shipped", current: StatusCancelled, target: StatusShipping},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if CanTransition(tt.current, tt.target) {
				t.Fatalf("CanTransition(%q, %q) = true, want false", tt.current, tt.target)
			}
		})
	}
}

func TestAllowedCurrentStatuses_ReturnsCopy(t *testing.T) {
	statuses := AllowedCurrentStatuses(StatusPaid)
	statuses[0] = StatusCancelled

	if !CanTransition(StatusPending, StatusPaid) {
		t.Fatal("AllowedCurrentStatuses returned internal slice; transition policy was mutated")
	}
}

func TestIsValidPaymentStatus(t *testing.T) {
	valid := []string{PaymentStatusUnpaid, PaymentStatusPending, PaymentStatusPaid, PaymentStatusFailed, PaymentStatusRefunded}
	for _, status := range valid {
		if !IsValidPaymentStatus(status) {
			t.Fatalf("IsValidPaymentStatus(%q) = false, want true", status)
		}
	}

	if IsValidPaymentStatus("captured") {
		t.Fatal("IsValidPaymentStatus(captured) = true, want false")
	}
}

func TestCanPaymentTransition_AllowsExpectedFlow(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
	}{
		{name: "unpaid to pending", current: PaymentStatusUnpaid, target: PaymentStatusPending},
		{name: "unpaid to paid", current: PaymentStatusUnpaid, target: PaymentStatusPaid},
		{name: "pending to paid", current: PaymentStatusPending, target: PaymentStatusPaid},
		{name: "pending to failed", current: PaymentStatusPending, target: PaymentStatusFailed},
		{name: "failed to pending", current: PaymentStatusFailed, target: PaymentStatusPending},
		{name: "paid to refunded", current: PaymentStatusPaid, target: PaymentStatusRefunded},
		{name: "unpaid idempotent", current: PaymentStatusUnpaid, target: PaymentStatusUnpaid},
		{name: "pending idempotent", current: PaymentStatusPending, target: PaymentStatusPending},
		{name: "paid idempotent", current: PaymentStatusPaid, target: PaymentStatusPaid},
		{name: "failed idempotent", current: PaymentStatusFailed, target: PaymentStatusFailed},
		{name: "refunded idempotent", current: PaymentStatusRefunded, target: PaymentStatusRefunded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !CanPaymentTransition(tt.current, tt.target) {
				t.Fatalf("CanPaymentTransition(%q, %q) = false, want true", tt.current, tt.target)
			}
		})
	}
}

func TestCanPaymentTransition_BlocksInvalidFlow(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
	}{
		{name: "paid cannot fail", current: PaymentStatusPaid, target: PaymentStatusFailed},
		{name: "refunded cannot become paid", current: PaymentStatusRefunded, target: PaymentStatusPaid},
		{name: "failed cannot refund", current: PaymentStatusFailed, target: PaymentStatusRefunded},
		{name: "pending cannot become unpaid", current: PaymentStatusPending, target: PaymentStatusUnpaid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if CanPaymentTransition(tt.current, tt.target) {
				t.Fatalf("CanPaymentTransition(%q, %q) = true, want false", tt.current, tt.target)
			}
		})
	}
}

func TestInitialPaymentStatus(t *testing.T) {
	if got := InitialPaymentStatus("cod"); got != PaymentStatusUnpaid {
		t.Fatalf("InitialPaymentStatus(cod) = %q, want unpaid", got)
	}
	if got := InitialPaymentStatus("demo_payment"); got != PaymentStatusPending {
		t.Fatalf("InitialPaymentStatus(demo_payment) = %q, want pending", got)
	}
}
