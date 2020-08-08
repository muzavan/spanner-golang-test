package spanner

import (
	"time"

	span "cloud.google.com/go/spanner"
)

// SingerRow represent the row in Singer table
type SingerRow struct {
	SingerID   int64
	FirstName  span.NullString
	LastName   span.NullString
	SingerInfo []byte
	BirthDate  time.Time
}
