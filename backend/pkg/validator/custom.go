package validator

import "github.com/google/uuid"

func IsUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}
