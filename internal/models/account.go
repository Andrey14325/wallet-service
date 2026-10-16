package models

import "time"

type Account struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Transfer struct {
	FromID string `json:"from_account_id"`
	ToID   string `json:"to_account_id"`
	Amount int64  `json:"amount"`
}
