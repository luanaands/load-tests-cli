package entity

type Report struct {
	TotalRequests   int64          `json:"total_requests"`
	SuccessRequests int64          `json:"success_requests"`
	FinalTime       float64        `json:"final_time"`
	OtherStatus     map[string]int `json:"other_status"`
}
