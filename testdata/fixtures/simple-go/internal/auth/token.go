package auth

func SignToken(payload string) string {
	return "signed:" + payload
}
