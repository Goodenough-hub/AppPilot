package finflow

import (
	"time"

	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID         string          `json:"id"`
	UserID     string          `json:"userId,omitempty"`
	Amount     decimal.Decimal `json:"amount"`
	Type       string          `json:"type"`
	Note       string          `json:"note"`
	Date       string          `json:"date"`
	Time       *string         `json:"time"`
	CreatedAt  time.Time       `json:"createdAt"`
	CategoryID *string         `json:"categoryId"`
	AccountID  *string         `json:"accountId"`
	ToAccount  *string         `json:"toAccountId"`
	SourceID   *string         `json:"sourceId"`
	SourceType *string         `json:"sourceType"`
	Vendor     *string         `json:"vendor"`
	TripID     *string         `json:"tripId"`
}

type Category struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId,omitempty"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Icon      string    `json:"icon"`
	ColorHex  string    `json:"colorHex"`
	SortOrder int       `json:"sortOrder"`
	IsSystem  bool      `json:"isSystem"`
	ParentID  *string   `json:"parentId"`
	Scope     string    `json:"scope"`
	CreatedAt time.Time `json:"createdAt"`
}

type Account struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId,omitempty"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	Icon           string    `json:"icon"`
	ColorHex       string    `json:"colorHex"`
	InitialBalance decimal.Decimal `json:"initialBalance"`
	SortOrder      int       `json:"sortOrder"`
	IsSystem       bool      `json:"isSystem"`
	ParentID       *string   `json:"parentId"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Budget struct {
	ID         string          `json:"id"`
	UserID     string          `json:"userId,omitempty"`
	Amount     decimal.Decimal `json:"amount"`
	Month      int             `json:"month"`
	Year       int             `json:"year"`
	CategoryID string          `json:"categoryId"`
}

type Recurring struct {
	ID         string          `json:"id"`
	UserID     string          `json:"userId,omitempty"`
	Amount     decimal.Decimal `json:"amount"`
	Type       string          `json:"type"`
	Note       string          `json:"note"`
	CategoryID *string         `json:"categoryId"`
	AccountID  *string         `json:"accountId"`
	ToAccount  *string         `json:"toAccountId"`
	Frequency  string          `json:"frequency"`
	Interval   int             `json:"interval"`
	DayOfMonth *int            `json:"dayOfMonth"`
	DayOfWeek  *int            `json:"dayOfWeek"`
	NextDate   string          `json:"nextDate"`
	StartDate  string          `json:"startDate"`
	EndDate    *string         `json:"endDate"`
	IsActive   bool            `json:"isActive"`
	CreatedAt  time.Time       `json:"createdAt"`
}

type Trip struct {
	ID        string          `json:"id"`
	UserID    string          `json:"userId,omitempty"`
	Name      string          `json:"name"`
	StartDate *string         `json:"startDate"`
	EndDate   *string         `json:"endDate"`
	Budget    decimal.Decimal `json:"budget"`
	Note      string          `json:"note"`
	CreatedAt time.Time       `json:"createdAt"`
}
