package analytics

import (
	"time"

	"github.com/google/uuid"
)

type Click struct {
	ID        uuid.UUID
	LinkID    uuid.UUID
	ClickedAt time.Time
	UserAgent string
	Referer   *string
	IP        *string
}

const MaxRecentClicksLimit = 500

type ClickFilter struct {
	From *time.Time
	To   *time.Time
}

type TimeBucketCount struct {
	Bucket time.Time
	Count  int64
}

type UserAgentCount struct {
	UserAgent string
	Count     int64
}

type LinkAnalytics struct {
	TotalClicks       int64
	ClicksByDay       []TimeBucketCount
	ClicksByMonth     []TimeBucketCount
	ClicksByUserAgent []UserAgentCount
	RecentClicks      []Click
}
