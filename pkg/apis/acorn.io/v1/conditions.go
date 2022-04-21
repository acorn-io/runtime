package v1

type Condition struct {
	Success       bool   `json:"success,omitempty"`
	Error         bool   `json:"error,omitempty"`
	Message       string `json:"message,omitempty"`
	Transitioning bool   `json:"transitioning,omitempty"`
}
