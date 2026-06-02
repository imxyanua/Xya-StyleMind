package order

var validOrderStatuses = map[string]struct{}{
	StatusPending:   {},
	StatusPaid:      {},
	StatusShipping:  {},
	StatusCompleted: {},
	StatusCancelled: {},
}

var allowedCurrentStatusesByTarget = map[string][]string{
	StatusPending:   {StatusPending},
	StatusPaid:      {StatusPending, StatusPaid},
	StatusShipping:  {StatusPaid, StatusShipping},
	StatusCompleted: {StatusShipping, StatusCompleted},
	StatusCancelled: {StatusPending, StatusPaid, StatusCancelled},
}

func IsValidStatus(status string) bool {
	_, ok := validOrderStatuses[status]
	return ok
}

func AllowedCurrentStatuses(target string) []string {
	statuses := allowedCurrentStatusesByTarget[target]
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
