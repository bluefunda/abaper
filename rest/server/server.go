package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bluefunda/abaper/rest/models"
	"go.uber.org/zap"
)

// Config represents the application configuration
type Config struct {
	APIKey      string
	ADTHost     string
	ADTClient   string
	ADTUsername string
	ADTPassword string
	Verbose     bool
	Quiet       bool
}

// ADTClient interface for dependency injection
type ADTClient interface {
	GetProgram(name string) (*ADTSourceCode, error)
	GetClass(name string) (*ADTSourceCode, error)
	GetFunction(name, functionGroup string) (*ADTSourceCode, error)
	GetInclude(name string) (*ADTSourceCode, error)
	GetInterface(name string) (*ADTSourceCode, error)
	GetStructure(name string) (*ADTSourceCode, error)
	GetTable(name string) (*ADTSourceCode, error)
	GetPackageContents(name string) (*ADTPackageInfo, error)
	SearchObjects(pattern string, objectTypes []string) (*ADTSearchResults, error)
	ListPackages(pattern string) ([]*ADTPackage, error)
	TestConnection() error
	CreateProgram(name, description, source string) error
}

// ADTSourceCode represents source code from ADT
type ADTSourceCode struct {
	ObjectType string
	Source     string
	Version    string
}

// ADTPackageInfo represents package information
type ADTPackageInfo struct {
	Name        string
	Description string
	Objects     []ADTObject
}

// ADTObject represents an ABAP object
type ADTObject struct {
	Name        string
	Type        string
	Description string
	Package     string
}

// ADTSearchResults represents search results
type ADTSearchResults struct {
	Total   int
	Objects []ADTObject
}

// ADTPackage represents a package
type ADTPackage struct {
	Name        string
	Description string
}

// RestServer handles REST API requests with CLI feature parity (no AI)
type RestServer struct {
	logger *zap.Logger
	config *Config
}

// NewRestServer creates a new REST server instance
func NewRestServer(config *Config, logger *zap.Logger) *RestServer {
	return &RestServer{
		logger: logger.With(zap.String("component", "rest_server")),
		config: config,
	}
}

// Start starts the REST server
func (rs *RestServer) Start(port string) {
	rs.logger.Info("Starting REST server with CLI feature parity", zap.String("port", port))

	// API endpoints for CLI parity (no AI)
	http.HandleFunc("/api/v1/objects/get", rs.corsHandler(rs.getObjectHandler))
	http.HandleFunc("/api/v1/objects/search", rs.corsHandler(rs.searchObjectsHandler))
	http.HandleFunc("/api/v1/objects/list", rs.corsHandler(rs.listObjectsHandler))
	http.HandleFunc("/api/v1/system/connect", rs.corsHandler(rs.connectHandler))

	// Removed AI endpoints - return feature removed messages
	http.HandleFunc("/api/v1/ai/analyze", rs.corsHandler(rs.removedAIHandler))
	http.HandleFunc("/api/v1/ai/review", rs.corsHandler(rs.removedAIHandler))
	http.HandleFunc("/api/v1/ai/optimize", rs.corsHandler(rs.removedAIHandler))
	http.HandleFunc("/api/v1/ai/create", rs.corsHandler(rs.removedAIHandler))

	// Legacy AI endpoints
	http.HandleFunc("/generate-code", rs.corsHandler(rs.generateCodeHandler))
	http.HandleFunc("/generate-code-stream", rs.corsHandler(rs.generateCodeStreamHandler))

	// Health and version endpoints
	http.HandleFunc("/health", rs.healthHandler)
	http.HandleFunc("/version", rs.versionHandler)

	rs.logger.Info("REST server endpoints registered (CLI parity + removed AI endpoints)", zap.Int("endpoint_count", 12))

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		rs.logger.Fatal("Failed to start server", zap.Error(err))
	}
}

// corsHandler adds CORS headers to responses
func (rs *RestServer) corsHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rs.logger.Debug("Processing request", zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.String("remote_addr", r.RemoteAddr))

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// sendSuccess sends a successful API response
func (rs *RestServer) sendSuccess(w http.ResponseWriter, data interface{}) {
	response := models.APIResponse{
		Success: true,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// sendError sends an error API response
func (rs *RestServer) sendError(w http.ResponseWriter, message string, statusCode int) {
	rs.logger.Warn("API error", zap.String("error", message), zap.Int("status", statusCode))

	response := models.APIResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// healthHandler handles health check requests
func (rs *RestServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"features": map[string]bool{
			"quiet_mode_default": true,
			"file_logging":       true,
			"adt_integration":    true,
			"cli_parity":         true,
			"ai_removed":         true,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// versionHandler handles version requests
func (rs *RestServer) versionHandler(w http.ResponseWriter, r *http.Request) {
	version := map[string]string{
		"features": "CLI and REST Services (No AI)",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

// Removed legacy AI handlers - now returns appropriate messages
func (rs *RestServer) generateCodeHandler(w http.ResponseWriter, r *http.Request) {
	rs.sendError(w, "AI code generation features have been removed. This endpoint is no longer available.", http.StatusGone)
}

func (rs *RestServer) generateCodeStreamHandler(w http.ResponseWriter, r *http.Request) {
	rs.sendError(w, "AI streaming features have been removed. This endpoint is no longer available.", http.StatusGone)
}

// Placeholder implementations for handlers that require ADT client
func (rs *RestServer) getObjectHandler(w http.ResponseWriter, r *http.Request) {
	rs.sendError(w, "Get object endpoint requires ADT client integration", http.StatusNotImplemented)
}

func (rs *RestServer) searchObjectsHandler(w http.ResponseWriter, r *http.Request) {
	rs.sendError(w, "Search endpoint requires ADT client integration", http.StatusNotImplemented)
}

func (rs *RestServer) listObjectsHandler(w http.ResponseWriter, r *http.Request) {
	rs.sendError(w, "List endpoint requires ADT client integration", http.StatusNotImplemented)
}

func (rs *RestServer) connectHandler(w http.ResponseWriter, r *http.Request) {
	rs.sendError(w, "Connect endpoint requires ADT client integration", http.StatusNotImplemented)
}

// removedAIHandler handles requests to removed AI endpoints
func (rs *RestServer) removedAIHandler(w http.ResponseWriter, r *http.Request) {
	rs.sendError(w, "AI features have been removed from this version. This endpoint is no longer available.", http.StatusGone)
}
