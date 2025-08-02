package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/bluefunda/abaper/rest/models"
	"github.com/bluefunda/abaper/types"
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

// RestServer handles REST API requests with CLI feature parity (no AI)
type RestServer struct {
	logger    *zap.Logger
	config    *Config
	adtClient types.ADTClient // Use shared interface
}

// NewRestServer creates a new REST server instance with ADT client
func NewRestServer(config *Config, logger *zap.Logger, adtClient types.ADTClient) *RestServer {
	return &RestServer{
		logger:    logger.With(zap.String("component", "rest_server")),
		config:    config,
		adtClient: adtClient,
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

// getObjectHandler handles object retrieval requests (CLI get command equivalent)
func (rs *RestServer) getObjectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rs.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rs.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ObjectType == "" || req.ObjectName == "" {
		rs.sendError(w, "object_type and object_name are required", http.StatusBadRequest)
		return
	}

	if !rs.adtClient.IsAuthenticated() {
		rs.sendError(w, "ADT client not authenticated", http.StatusUnauthorized)
		return
	}

	objectType := strings.ToUpper(req.ObjectType)
	objectName := strings.ToUpper(req.ObjectName)

	rs.logger.Info("Getting object via REST API",
		zap.String("type", objectType),
		zap.String("name", objectName))

	var result interface{}
	var err error

	switch objectType {
	case "PROGRAM", "PROG":
		result, err = rs.adtClient.GetProgram(objectName)
	case "CLASS", "CLAS":
		result, err = rs.adtClient.GetClass(objectName)
	case "FUNCTION", "FUNC":
		if len(req.Args) == 0 {
			rs.sendError(w, "function group required in args for function modules", http.StatusBadRequest)
			return
		}
		functionGroup := strings.ToUpper(req.Args[0])
		result, err = rs.adtClient.GetFunction(objectName, functionGroup)
	case "INCLUDE", "INCL":
		result, err = rs.adtClient.GetInclude(objectName)
	case "INTERFACE", "INTF":
		result, err = rs.adtClient.GetInterface(objectName)
	case "STRUCTURE", "STRU":
		result, err = rs.adtClient.GetStructure(objectName)
	case "TABLE", "TABL":
		result, err = rs.adtClient.GetTable(objectName)
	case "PACKAGE", "PACK":
		result, err = rs.adtClient.GetPackageContents(objectName)
	default:
		rs.sendError(w, "unsupported object type: "+objectType, http.StatusBadRequest)
		return
	}

	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rs.sendSuccess(w, result)
}

// searchObjectsHandler handles object search requests (CLI search command equivalent)
func (rs *RestServer) searchObjectsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rs.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rs.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ObjectName == "" {
		rs.sendError(w, "object_name (search pattern) is required", http.StatusBadRequest)
		return
	}

	if !rs.adtClient.IsAuthenticated() {
		rs.sendError(w, "ADT client not authenticated", http.StatusUnauthorized)
		return
	}

	pattern := req.ObjectName
	var objectTypes []string

	// Convert args to object types
	for _, arg := range req.Args {
		objectTypes = append(objectTypes, strings.ToUpper(arg))
	}

	rs.logger.Info("Searching objects via REST API",
		zap.String("pattern", pattern),
		zap.Strings("types", objectTypes))

	results, err := rs.adtClient.SearchObjects(pattern, objectTypes)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rs.sendSuccess(w, results)
}

// listObjectsHandler handles object listing requests (CLI list command equivalent)
func (rs *RestServer) listObjectsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rs.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rs.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ObjectType == "" {
		rs.sendError(w, "object_type is required", http.StatusBadRequest)
		return
	}

	if !rs.adtClient.IsAuthenticated() {
		rs.sendError(w, "ADT client not authenticated", http.StatusUnauthorized)
		return
	}

	listType := strings.ToLower(req.ObjectType)
	pattern := req.ObjectName
	if pattern == "" {
		pattern = "*"
	}

	rs.logger.Info("Listing objects via REST API",
		zap.String("type", listType),
		zap.String("pattern", pattern))

	switch listType {
	case "packages", "package":
		packages, err := rs.adtClient.ListPackages(pattern)
		if err != nil {
			rs.sendError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rs.sendSuccess(w, packages)
	default:
		rs.sendError(w, "unsupported list type: "+listType, http.StatusBadRequest)
	}
}

// connectHandler handles connection test requests (CLI connect command equivalent)
func (rs *RestServer) connectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rs.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rs.logger.Info("Testing ADT connection via REST API")

	if rs.adtClient == nil {
		rs.sendError(w, "ADT client not configured", http.StatusInternalServerError)
		return
	}

	if err := rs.adtClient.TestConnection(); err != nil {
		rs.logger.Error("ADT connection test failed", zap.Error(err))
		rs.sendError(w, "Connection failed: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	connectionStatus := map[string]any{
		"status":        "connected",
		"authenticated": rs.adtClient.IsAuthenticated(),
		"timestamp":     time.Now().UTC(),
		"message":       "ADT connection successful",
	}

	rs.sendSuccess(w, connectionStatus)
}

// healthHandler handles health check requests
func (rs *RestServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	adtStatus := "disconnected"
	if rs.adtClient != nil && rs.adtClient.IsAuthenticated() {
		adtStatus = "connected"
	}

	health := map[string]any{
		"status":     "healthy",
		"timestamp":  time.Now().UTC(),
		"adt_status": adtStatus,
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

// removedAIHandler handles requests to removed AI endpoints
func (rs *RestServer) removedAIHandler(w http.ResponseWriter, r *http.Request) {
	rs.sendError(w, "AI features have been removed from this version. This endpoint is no longer available.", http.StatusGone)
}
