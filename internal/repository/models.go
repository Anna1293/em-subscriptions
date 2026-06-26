package repository

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID          int        `json:"id"`
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"`
	UserID      uuid.UUID  `json:"user_id"`
	StartDate   string     `json:"start_date"` // MM-YYYY
	EndDate     *string    `json:"end_date,omitempty"`
}

type Filter struct {
	UserID      *uuid.UUID
	ServiceName *string
}

type PeriodFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	From        time.Time
	To          time.Time
}
