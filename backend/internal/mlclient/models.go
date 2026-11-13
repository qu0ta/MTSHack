package mlclient

import "errors"

var (
	ErrUnavailable     = errors.New("ml service unavailable")
	ErrRejected        = errors.New("ML rejected request data")
	ErrBusy            = errors.New("ML busy")
	ErrTimeout         = errors.New("ml service timeout")
	ErrInvalidResponse = errors.New("invalid ml response")
)

type HistoryItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnswerRequest struct {
	LeadRequested bool          `json:"lead_requested,omitempty"`
	SessionID     string        `json:"session_id"`
	Message       string        `json:"message"`
	History       []HistoryItem `json:"history"`
	ClientData    ClientData    `json:"client_data"`
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

func (d ClientData) Valid() bool {
	return (d.ConnectedPowerKW == nil || *d.ConnectedPowerKW >= 0) && (d.MonthlyConsumptionKWH == nil || *d.MonthlyConsumptionKWH >= 0)
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
