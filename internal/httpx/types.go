package httpx

type Response struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}
