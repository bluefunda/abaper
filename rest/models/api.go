package models

// REST API request and response structures

// APIRequest represents a generic API request
type APIRequest struct {
	Action      string            `json:"action"`
	ObjectType  string            `json:"object_type,omitempty"`
	ObjectName  string            `json:"object_name,omitempty"`
	Description string            `json:"description,omitempty"`
	Source      string            `json:"source,omitempty"`
	Package     string            `json:"package,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Prompt      string            `json:"prompt,omitempty"`
	Config      map[string]string `json:"config,omitempty"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ObjectRequest for object retrieval requests
type ObjectRequest struct {
	ObjectType string   `json:"object_type"`
	ObjectName string   `json:"object_name"`
	Args       []string `json:"args,omitempty"` // For function groups, etc.
}

// SearchRequest for object search requests
type SearchRequest struct {
	Pattern     string   `json:"pattern"`
	ObjectTypes []string `json:"object_types,omitempty"`
}

// ListRequest for object listing requests
type ListRequest struct {
	ObjectType string `json:"object_type"` // "packages", etc.
	Pattern    string `json:"pattern,omitempty"`
}

// ConnectionResponse for connection test results
type ConnectionResponse struct {
	Status        string `json:"status"`
	Authenticated bool   `json:"authenticated"`
	Timestamp     string `json:"timestamp"`
	Message       string `json:"message"`
}

// GenerateRequest for AI generation endpoints (removed but kept for compatibility)
type GenerateRequest struct {
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream,omitempty"`
}

// ChatRequest for conversational AI endpoints (removed but kept for compatibility)
type ChatRequest struct {
	Message string `json:"message"`
	Context string `json:"context,omitempty"`
	Stream  bool   `json:"stream,omitempty"`
}
