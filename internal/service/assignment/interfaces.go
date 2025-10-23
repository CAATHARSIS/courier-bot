package assignment

type NotificationSender interface {
	SendMessage(chatID int64, message string) error
	SendMessageWithKeyboard(chatID int64, message string, orderID int) error
}

type Button struct {
	Text string
	Data string
}

type AssignmentResult struct {
	Success      bool
	CourierID    int
	ErrorMessage string
}
