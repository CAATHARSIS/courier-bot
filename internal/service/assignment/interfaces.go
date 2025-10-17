package assignment

type NotificationSender interface {
	SendMessage(chatID int64, message string, orderID int) error
	SendMessageWithKeyboard(chatID int64, message string, orderID int, buttons []Button) error
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
