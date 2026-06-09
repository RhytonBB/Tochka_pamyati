package service

type ContentValidationResult struct {
	RequiresAck bool              `json:"requires_ack"`
	Reasons     []string          `json:"reasons"`
	Fields      map[string]string `json:"fields"`
	HighRisk    bool              `json:"high_risk"`
}

func (r ContentValidationResult) HasBlockingIssues() bool {
	return len(r.Fields) > 0
}
