package models

// REST API request and response structures

// APIRequest represents a generic API request
type APIRequest struct {
	Action     string            `json:"action"`
	ObjectType string            `json:"object_type,omitempty"`
	ObjectName string            `json:"object_name,omitempty"`
	Args       []string          `json:"args,omitempty"`
	Prompt     string            `json:"prompt,omitempty"`
	Config     map[string]string `json:"config,omitempty"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// GenerateRequest for AI generation endpoints
type GenerateRequest struct {
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream,omitempty"`
}

// ChatRequest for conversational AI endpoints
type ChatRequest struct {
	Message string `json:"message"`
	Context string `json:"context,omitempty"`
	Stream  bool   `json:"stream,omitempty"`
}
