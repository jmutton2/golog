package main

import "time"

type createResponse struct {
	ID string
}

type getResponse struct {
	ID string `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Correlation_id string `json:"correlation_id"`
	Severity LogLevel `json:"severity"`
	Message string `json:"message"`
}
