package mlclient

import "errors"

var (
	ErrUnavailable     = errors.New("ml service unavailable")
	ErrTimeout         = errors.New("ml service timeout")
	ErrInvalidResponse = errors.New("invalid ml response")
)

type HistoryItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnswerRequest struct {
	SessionID  string        `json:"session_id"`
	Message    string        `json:"message"`
	History    []HistoryItem `json:"history"`
	ClientData ClientData    `json:"client_data"`
}

type ClientData struct {
	CompanyName           *string  `json:"company_name,omitempty"`
	ContactName           *string  `json:"contact_name,omitempty"`
	Phone                 *string  `json:"phone,omitempty"`
	Industry              *string  `json:"industry,omitempty"`
	Region                *string  `json:"region,omitempty"`
	ConnectedPowerKW      *float64 `json:"connected_power_kw,omitempty"`
	MonthlyConsumptionKWH *float64 `json:"monthly_consumption_kwh,omitempty"`
}

type Lead struct {
	ContactName string `json:"contact_name"`
	Phone       string `json:"phone"`
	CompanyName string `json:"company_name"`
	Summary     string `json:"summary"`
}

type AnswerResponse struct {
	Answer         string     `json:"answer"`
	ClientData     ClientData `json:"client_data"`
	Sources        []string   `json:"sources"`
	Lead           *Lead      `json:"lead"`
	ResponseTimeMS int        `json:"response_time_ms"`
}
