package dto

import "time"

// OrderResponse represents an order as exposed via transport layers.
type OrderResponse struct {
	ID        int64     `json:"id"`
	Number    string    `json:"number"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
