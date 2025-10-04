package models

import "time"

type Order struct {
	ID                     int        `json:"id"`
	UserID                 int        `json:"user_id"`
	Surname                string     `json:"surname"`
	Name                   string     `json:"name"`
	PhoneNumber            string     `json:"phone_number"`
	City                   string     `json:"city"`
	Address                string     `json:"address"`
	Flat                   string     `json:"flat"`
	Entrance               string     `json:"entrance"`
	DeliveryPrice          int    `json:"delivery_price"`
	FirstPrice             int    `json:"first_price"`
	FinalPrice             int    `json:"final_price"`
	PaidPrice              int    `json:"paid_price"`
	BonusAccrualPercentage int        `json:"bonus_accrual_percentage"`
	RecievedBonuses        int        `json:"recieved_bonuses"`
	LostBonuses            int        `json:"lost_bonuses"`
	CreatedAt              time.Time  `json:"created_at"`
	DeliverDate            *time.Time `json:"delivery_date"`
	RecievedAt             *time.Time `json:"recieved_at"`
	IsPaid                 bool       `json:"is_paid"`
	IsDelivery             bool       `json:"is_delivery"`
	IsAssembled            bool       `json:"is_assembled"`
	IsReceived             bool       `json:"is_received"`
	PaymentUrl             string     `json:"payment_url"`
	CourierID              *int       `json:"courier_id"`
}
