package links

import (
	"time"

	"github.com/google/uuid"
)

type Link struct {
	ID          uuid.UUID
	Code        string
	OriginalURL string
	IsCustom    bool
	CreatedAt   time.Time
	DisabledAt  *time.Time
}
