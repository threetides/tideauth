package httpx

import (
	"net/http"
	"time"
)

var Client = &http.Client{
	Timeout: 30 * time.Second,
}
