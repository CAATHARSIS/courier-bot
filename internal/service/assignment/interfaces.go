package assignment

type Button struct {
	Text string
	Data string
}

type AssignmentResult struct {
	Success      bool
	CourierID    int
	ErrorMessage string
}
