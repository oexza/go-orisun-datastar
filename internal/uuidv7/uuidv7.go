package uuidv7

import "github.com/google/uuid"

func NewString() string {
	return uuid.Must(uuid.NewV7()).String()
}
