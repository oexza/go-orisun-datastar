package views

type Field struct {
	Name  string
	Type  string
	Label string
}

func sendValidationOTPSSE(userID string) string {
	return postFormSSE("/register/" + userID + "/send-email-validation-otp")
}
