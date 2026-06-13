package order

import "testing"

func TestAvailableStockSubtractsActiveReservations(t *testing.T) {
	if got := availableStock(10, 4); got != 6 {
		t.Fatalf("availableStock = %d, want 6", got)
	}
	if got := availableStock(3, 7); got != 0 {
		t.Fatalf("availableStock below zero = %d, want 0", got)
	}
}

func TestCanReserveQuantityPreventsOversell(t *testing.T) {
	tests := []struct {
		name               string
		stock              int
		activeReservations int
		quantity           int
		want               bool
	}{
		{name: "within available stock", stock: 10, activeReservations: 4, quantity: 6, want: true},
		{name: "exceeds available stock", stock: 10, activeReservations: 4, quantity: 7, want: false},
		{name: "active reservations consume all stock", stock: 3, activeReservations: 3, quantity: 1, want: false},
		{name: "reject zero quantity", stock: 3, activeReservations: 0, quantity: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canReserveQuantity(tt.stock, tt.activeReservations, tt.quantity); got != tt.want {
				t.Fatalf("canReserveQuantity(%d,%d,%d) = %v, want %v", tt.stock, tt.activeReservations, tt.quantity, got, tt.want)
			}
		})
	}
}

func TestSequentialCheckoutMathCannotOversell(t *testing.T) {
	stock := 5
	firstCheckoutQuantity := 3
	secondCheckoutQuantity := 3

	if !canReserveQuantity(stock, 0, firstCheckoutQuantity) {
		t.Fatal("first checkout should reserve stock")
	}
	stock -= firstCheckoutQuantity

	if canReserveQuantity(stock, 0, secondCheckoutQuantity) {
		t.Fatal("second checkout should not reserve more than remaining stock")
	}
}
