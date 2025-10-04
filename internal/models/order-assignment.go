package models

import "time"

type CourierResponseStatus string

const (
	ResponseStatusWaiting  CourierResponseStatus = "waiting"
	ResponseStatusAccepted CourierResponseStatus = "accepted"
	ResponseStatusRejected CourierResponseStatus = "rejected"
	ResponsseStatusExpired CourierResponseStatus = "expired"
)

func (s CourierResponseStatus) IsValid() bool {
	switch s {
	case ResponseStatusWaiting, ResponseStatusAccepted, ResponseStatusRejected, ResponsseStatusExpired:
		return true
	default:
		return false
	}
}

func (s CourierResponseStatus) String() string {
	return string(s)
}

type OrderAssignment struct {
	ID                    int                   `json:"id"`
	OrderID               int                   `json:"order_id"`
	CourierID             int                   `json:"courier_id"`
	AssignedAt            time.Time             `json:"assigned_at"`
	ExpiredAt             time.Time             `json:"expired_at"`
	CourierResponseStatus CourierResponseStatus `json:"courier_response_status"`
}
