package order

var validOrderStatuses = map[string]struct{}{
	StatusPending:   {},
	StatusPaid:      {},
	StatusShipping:  {},
	StatusCompleted: {},
	StatusCancelled: {},
}

var validPaymentStatuses = map[string]struct{}{
	PaymentStatusUnpaid:   {},
	PaymentStatusPending:  {},
	PaymentStatusPaid:     {},
	PaymentStatusFailed:   {},
	PaymentStatusRefunded: {},
}

var allowedCurrentStatusesByTarget = map[string][]string{
	StatusPending:   {StatusPending},
	StatusPaid:      {StatusPending, StatusPaid},
	StatusShipping:  {StatusPaid, StatusShipping},
	StatusCompleted: {StatusShipping, StatusCompleted},
	StatusCancelled: {StatusPending, StatusPaid, StatusCancelled},
}

var allowedCurrentPaymentStatusesByTarget = map[string][]string{
	PaymentStatusUnpaid:   {PaymentStatusUnpaid},
	PaymentStatusPending:  {PaymentStatusUnpaid, PaymentStatusPending, PaymentStatusFailed},
	PaymentStatusPaid:     {PaymentStatusUnpaid, PaymentStatusPending, PaymentStatusPaid},
	PaymentStatusFailed:   {PaymentStatusPending, PaymentStatusFailed},
	PaymentStatusRefunded: {PaymentStatusPaid, PaymentStatusRefunded},
}

func IsValidStatus(status string) bool {
	_, ok := validOrderStatuses[status]
	return ok
}

func IsValidPaymentStatus(status string) bool {
	_, ok := validPaymentStatuses[status]
	return ok
}

func AllowedCurrentStatuses(target string) []string {
	statuses := allowedCurrentStatusesByTarget[target]
	out := make([]string, len(statuses))
	copy(out, statuses)
	return out
}

func AllowedCurrentPaymentStatuses(target string) []string {
	statuses := allowedCurrentPaymentStatusesByTarget[target]
	out := make([]string, len(statuses))
	copy(out, statuses)
	return out
}

func CanTransition(current, target string) bool {
	for _, allowed := range AllowedCurrentStatuses(target) {
		if current == allowed {
			return true
		}
	}
	return false
}

func CanPaymentTransition(current, target string) bool {
	for _, allowed := range AllowedCurrentPaymentStatuses(target) {
		if current == allowed {
			return true
		}
	}
	return false
}

func InitialPaymentStatus(paymentMethod string) string {
	if paymentMethod == "demo_payment" {
		return PaymentStatusPending
	}
	return PaymentStatusUnpaid
}
