package order

func availableStock(stock, activeReservations int) int {
	available := stock - activeReservations
	if available < 0 {
		return 0
	}
	return available
}

func canReserveQuantity(stock, activeReservations, quantity int) bool {
	return quantity > 0 && quantity <= availableStock(stock, activeReservations)
}
