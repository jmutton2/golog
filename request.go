package main

type createRequest struct {
	Correlation_id string `json:"correlation_id"`
	Severity LogLevel `json:"severity"`
	Message string `json:"message"`
}

type getRequest struct {
	ID string
}
