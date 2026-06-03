package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bluefunda/abaper/rest/models"
	"github.com/bluefunda/abaper/types"
	"go.uber.org/zap"
)

// Config represents the application configuration
type Config struct {
	APIKey          string
	ADTHost         string
	ADTClient       string
	ADTUsername     string
	ADTPassword     string
	AllowSelfSigned bool
	Verbose         bool
	Quiet           bool
}

// RestServer handles REST API requests with CLI feature parity (no AI)
type RestServer struct {
	logger         *zap.Logger
	config         *Config
	pool           *Pool           // nil when running in single-system mode
	adtClient      types.ADTClient // static fallback client (single-system mode)
	sourceReader   types.SourceReader
	sourceWriter   types.SourceWriter
	packageBrowser types.PackageBrowser
}

// NewRestServer creates a new REST server instance with a static ADT client.
// Use NewRestServerWithPool for multi-system support.
func NewRestServer(config *Config, logger *zap.Logger, adtClient types.ADTClient) *RestServer {
	return &RestServer{
		logger:         logger.With(zap.String("component", "rest_server")),
		config:         config,
		adtClient:      adtClient,
		sourceReader:   adtClient,
		sourceWriter:   adtClient,
		packageBrowser: adtClient,
	}
}

// NewRestServerWithPool creates a REST server that selects an ADT client from
// the pool on each request using X-SAP-* headers, with a static fallback.
func NewRestServerWithPool(config *Config, logger *zap.Logger, pool *Pool, fallback types.ADTClient) *RestServer {
	rs := &RestServer{
		logger:    logger.With(zap.String("component", "rest_server")),
		config:    config,
		pool:      pool,
		adtClient: fallback,
	}
	if fallback != nil {
		rs.sourceReader = fallback
		rs.sourceWriter = fallback
		rs.packageBrowser = fallback
	}
	return rs
}

// clientForRequest returns the ADT client to use for this request.
// If X-SAP-* headers are present and a pool is configured, a pooled client is
// returned. Otherwise falls back to the static client.
func (rs *RestServer) clientForRequest(r *http.Request) (types.ADTClient, error) {
	host := r.Header.Get("X-SAP-Host")
	sapClient := r.Header.Get("X-SAP-Client")
	user := r.Header.Get("X-SAP-User")
	password := r.Header.Get("X-SAP-Password")

	if rs.pool != nil && host != "" && user != "" && password != "" {
		if sapClient == "" {
			sapClient = "100"
		}
		cfg := types.ADTConfig{
			Host:            host,
			Client:          sapClient,
			Username:        user,
			Password:        password,
			Language:        "EN",
			AllowSelfSigned: rs.config.AllowSelfSigned,
		}
		return rs.pool.Get(r.Context(), cfg)
	}

	if rs.adtClient == nil {
		return nil, fmt.Errorf("no ADT client configured and no X-SAP-* headers provided")
	}
	return rs.adtClient, nil
}

// Handler returns an http.Handler with all routes registered on a private mux.
// Using a private mux (not http.DefaultServeMux) allows multiple RestServer
// instances and makes the server testable with httptest.NewServer.
func (rs *RestServer) Handler() http.Handler {
	mux := http.NewServeMux()

	// API endpoints for CLI parity (no AI)
	mux.HandleFunc("/api/v1/objects/get", rs.corsHandler(rs.getObjectHandler))
	mux.HandleFunc("/api/v1/objects/create", rs.corsHandler(rs.createObjectHandler))
	mux.HandleFunc("/api/v1/objects/save", rs.corsHandler(rs.saveObjectHandler))
	mux.HandleFunc("/api/v1/objects/search", rs.corsHandler(rs.searchObjectsHandler))
	mux.HandleFunc("/api/v1/objects/list", rs.corsHandler(rs.listObjectsHandler))
	mux.HandleFunc("/api/v1/objects/activate", rs.corsHandler(rs.activateObjectHandler))
	mux.HandleFunc("/api/v1/syntax-check", rs.corsHandler(rs.syntaxCheckHandler))
	mux.HandleFunc("/api/v1/format", rs.corsHandler(rs.formatSourceHandler))
	mux.HandleFunc("/api/v1/completion", rs.corsHandler(rs.completionHandler))
	mux.HandleFunc("/api/v1/navigation", rs.corsHandler(rs.navigationHandler))
	mux.HandleFunc("/api/v1/unit-tests", rs.corsHandler(rs.unitTestsHandler))
	mux.HandleFunc("/api/v1/transports/info", rs.corsHandler(rs.transportInfoHandler))
	mux.HandleFunc("/api/v1/transports/create", rs.corsHandler(rs.createTransportHandler))
	mux.HandleFunc("/api/v1/system/connect", rs.corsHandler(rs.connectHandler))

	// GitHub proxy endpoints
	mux.HandleFunc("/api/v1/github/oauth/callback", rs.corsHandler(rs.githubOAuthCallbackHandler))
	mux.HandleFunc("/api/v1/github/user", rs.corsHandler(rs.githubUserHandler))
	mux.HandleFunc("/api/v1/github/branches", rs.corsHandler(rs.githubBranchesHandler))
	mux.HandleFunc("/api/v1/github/tree", rs.corsHandler(rs.githubTreeHandler))
	mux.HandleFunc("/api/v1/github/file", rs.corsHandler(rs.githubFileHandler))

	// Removed AI endpoints - return feature removed messages
	mux.HandleFunc("/api/v1/ai/analyze", rs.corsHandler(rs.removedAIHandler))
	mux.HandleFunc("/api/v1/ai/review", rs.corsHandler(rs.removedAIHandler))
	mux.HandleFunc("/api/v1/ai/optimize", rs.corsHandler(rs.removedAIHandler))
	mux.HandleFunc("/api/v1/ai/create", rs.corsHandler(rs.removedAIHandler))

	// Legacy AI endpoints
	mux.HandleFunc("/generate-code", rs.corsHandler(rs.generateCodeHandler))
	mux.HandleFunc("/generate-code-stream", rs.corsHandler(rs.generateCodeStreamHandler))

	// Health and version endpoints
	mux.HandleFunc("/health", rs.healthHandler)
	mux.HandleFunc("/version", rs.versionHandler)

	return mux
}

// Start starts the REST server on the given port.
func (rs *RestServer) Start(port string) {
	rs.logger.Info("Starting REST server with CLI feature parity", zap.String("port", port))
	if err := http.ListenAndServe(":"+port, rs.Handler()); err != nil {
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
func (rs *RestServer) sendSuccess(w http.ResponseWriter, data any) {
	response := models.APIResponse[any]{
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

	response := models.APIResponse[any]{
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

	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	objectType := strings.ToUpper(req.ObjectType)
	objectName := strings.ToUpper(req.ObjectName)

	rs.logger.Info("Getting object via REST API",
		zap.String("type", objectType),
		zap.String("name", objectName))

	ctx := r.Context()
	var result any

	switch objectType {
	case "PROGRAM", "PROG":
		result, err = c.GetProgram(ctx, objectName)
	case "CLASS", "CLAS":
		result, err = c.GetClass(ctx, objectName)
	case "FUNCTION", "FUNC":
		if len(req.Args) == 0 {
			rs.sendError(w, "function group required in args for function modules", http.StatusBadRequest)
			return
		}
		functionGroup := strings.ToUpper(req.Args[0])
		result, err = c.GetFunction(ctx, objectName, functionGroup)
	case "INCLUDE", "INCL":
		result, err = c.GetInclude(ctx, objectName)
	case "INTERFACE", "INTF":
		result, err = c.GetInterface(ctx, objectName)
	case "STRUCTURE", "STRU":
		result, err = c.GetStructure(ctx, objectName)
	case "TABLE", "TABL":
		result, err = c.GetTable(ctx, objectName)
	case "DDLS", "DATA_DEFINITION":
		result, err = c.GetDDLSource(ctx, objectName)
	case "PACKAGE", "PACK":
		result, err = c.GetPackageContents(ctx, objectName)
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

// createObjectHandler handles object creation requests (CLI create command equivalent)
func (rs *RestServer) createObjectHandler(w http.ResponseWriter, r *http.Request) {
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

	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	objectType := strings.ToUpper(req.ObjectType)
	objectName := strings.ToUpper(req.ObjectName)

	// Parse creation options from request
	description := req.Description
	if description == "" {
		description = fmt.Sprintf("%s %s", objectType, objectName)
	}

	sourceCode := req.Source
	packageName := req.Package
	if packageName == "" {
		packageName = "$TMP" // Default to local package
	}

	rs.logger.Info("Creating object via REST API",
		zap.String("type", objectType),
		zap.String("name", objectName),
		zap.String("package", packageName),
		zap.Int("source_length", len(sourceCode)))

	ctx := r.Context()

	switch objectType {
	case "PROGRAM", "PROG":
		err = c.CreateProgram(ctx, objectName, description, packageName, sourceCode)
	case "CLASS", "CLAS":
		err = c.CreateClass(ctx, objectName, description, packageName, sourceCode)
	case "INCLUDE", "INCL":
		err = c.CreateInclude(ctx, objectName, description, sourceCode)
	case "INTERFACE", "INTF":
		err = c.CreateInterface(ctx, objectName, description, sourceCode)
	case "STRUCTURE", "STRU":
		err = c.CreateStructure(ctx, objectName, description, sourceCode)
	case "TABLE", "TABL":
		err = c.CreateTable(ctx, objectName, description, sourceCode)
	case "FUNCTIONGROUP", "FUGR":
		err = c.CreateFunctionGroup(ctx, objectName, description, sourceCode)
	default:
		rs.sendError(w, "unsupported object type for creation: "+objectType, http.StatusBadRequest)
		return
	}

	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response with creation details
	result := map[string]any{
		"object_name": objectName,
		"object_type": objectType,
		"description": description,
		"package":     packageName,
		"created":     true,
		"timestamp":   time.Now().UTC(),
		"message":     fmt.Sprintf("%s %s created successfully", objectType, objectName),
	}

	if sourceCode != "" {
		result["source_inserted"] = true
		result["source_length"] = len(sourceCode)
		result["activated"] = true
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

	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	pattern := req.ObjectName
	var objectTypes []string
	if req.ObjectType != "" {
		objectTypes = []string{strings.ToUpper(req.ObjectType)}
	}

	rs.logger.Info("Searching objects via REST API",
		zap.String("pattern", pattern),
		zap.Strings("types", objectTypes))

	results, err := c.SearchObjects(r.Context(), pattern, objectTypes)
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

	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
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
		packages, err := c.ListPackages(r.Context(), pattern)
		if err != nil {
			rs.sendError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rs.sendSuccess(w, packages)
	default:
		rs.sendError(w, "unsupported list type: "+listType, http.StatusBadRequest)
	}
}

func (rs *RestServer) saveObjectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rs.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rs.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.ObjectType == "" || req.ObjectName == "" || req.Source == "" {
		rs.sendError(w, "object_type, object_name, and source are required", http.StatusBadRequest)
		return
	}
	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	objectType := strings.ToUpper(req.ObjectType)
	objectName := strings.ToUpper(req.ObjectName)
	ctx := r.Context()
	switch objectType {
	case "PROGRAM", "PROG":
		err = c.UpdateProgram(ctx, objectName, req.Source)
	case "CLASS", "CLAS":
		err = c.UpdateClass(ctx, objectName, req.Source)
	case "INCLUDE", "INCL":
		err = c.UpdateInclude(ctx, objectName, req.Source)
	case "INTERFACE", "INTF":
		err = c.UpdateInterface(ctx, objectName, req.Source)
	case "FUNCTION", "FUNC":
		if len(req.Args) == 0 {
			rs.sendError(w, "function group required in args for function modules", http.StatusBadRequest)
			return
		}
		err = c.UpdateFunction(ctx, objectName, strings.ToUpper(req.Args[0]), req.Source)
	case "FUNCTIONGROUP", "FUGR":
		err = c.UpdateFunctionGroup(ctx, objectName, req.Source)
	default:
		rs.sendError(w, "unsupported object type for save: "+objectType, http.StatusBadRequest)
		return
	}
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rs.sendSuccess(w, map[string]any{
		"object_name":     objectName,
		"object_type":     objectType,
		"source_inserted": true,
		"source_length":   len(req.Source),
		"message":         fmt.Sprintf("%s %s saved successfully", objectType, objectName),
	})
}

func (rs *RestServer) activateObjectHandler(w http.ResponseWriter, r *http.Request) {
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
	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	objectType := strings.ToUpper(req.ObjectType)
	objectName := strings.ToUpper(req.ObjectName)
	ctx := r.Context()
	// Optionally save source before activating.
	if req.Source != "" {
		switch objectType {
		case "PROGRAM", "PROG":
			err = c.UpdateProgram(ctx, objectName, req.Source)
		case "CLASS", "CLAS":
			err = c.UpdateClass(ctx, objectName, req.Source)
		case "INCLUDE", "INCL":
			err = c.UpdateInclude(ctx, objectName, req.Source)
		case "INTERFACE", "INTF":
			err = c.UpdateInterface(ctx, objectName, req.Source)
		}
		if err != nil {
			rs.sendError(w, "save before activate failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	result, err := c.ActivateObject(ctx, objectType, objectName)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rs.sendSuccess(w, result)
}

func (rs *RestServer) syntaxCheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rs.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rs.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.ObjectType == "" || req.ObjectName == "" || req.Source == "" {
		rs.sendError(w, "object_type, object_name, and source are required", http.StatusBadRequest)
		return
	}
	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	result, err := c.SyntaxCheck(r.Context(), strings.ToUpper(req.ObjectType), strings.ToUpper(req.ObjectName), req.Source)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rs.sendSuccess(w, result)
}

func (rs *RestServer) formatSourceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rs.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rs.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Source == "" {
		rs.sendError(w, "source is required", http.StatusBadRequest)
		return
	}
	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	formatted, err := c.FormatSource(r.Context(), req.Source)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rs.sendSuccess(w, map[string]string{"source": formatted})
}

func (rs *RestServer) completionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rs.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rs.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.ObjectType == "" || req.ObjectName == "" || req.Source == "" {
		rs.sendError(w, "object_type, object_name, source, line, and column are required", http.StatusBadRequest)
		return
	}
	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	proposals, err := c.GetCompletionProposals(r.Context(), strings.ToUpper(req.ObjectType), strings.ToUpper(req.ObjectName), req.Source, req.Line, req.Column)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rs.sendSuccess(w, proposals)
}

func (rs *RestServer) navigationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rs.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rs.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.ObjectType == "" || req.ObjectName == "" || req.Source == "" {
		rs.sendError(w, "object_type, object_name, source, line, and column are required", http.StatusBadRequest)
		return
	}
	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	target, err := c.GetNavigationTarget(r.Context(), strings.ToUpper(req.ObjectType), strings.ToUpper(req.ObjectName), req.Source, req.Line, req.Column)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rs.sendSuccess(w, target)
}

func (rs *RestServer) unitTestsHandler(w http.ResponseWriter, r *http.Request) {
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
	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	result, err := c.RunUnitTests(r.Context(), strings.ToUpper(req.ObjectType), strings.ToUpper(req.ObjectName))
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rs.sendSuccess(w, result)
}

func (rs *RestServer) transportInfoHandler(w http.ResponseWriter, r *http.Request) {
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
	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	result, err := c.GetTransportInfo(r.Context(), strings.ToUpper(req.ObjectType), strings.ToUpper(req.ObjectName))
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rs.sendSuccess(w, result)
}

func (rs *RestServer) createTransportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rs.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rs.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.ObjectType == "" || req.ObjectName == "" || req.Description == "" {
		rs.sendError(w, "object_type, object_name, and description are required", http.StatusBadRequest)
		return
	}
	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	transportNumber, err := c.CreateTransport(r.Context(), strings.ToUpper(req.ObjectType), strings.ToUpper(req.ObjectName), req.Description, strings.ToUpper(req.Package))
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rs.sendSuccess(w, map[string]string{
		"transport_number": transportNumber,
		"description":      req.Description,
		"package":          strings.ToUpper(req.Package),
	})
}

// connectHandler handles connection test requests (CLI connect command equivalent)
func (rs *RestServer) connectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rs.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rs.logger.Info("Testing ADT connection via REST API")

	c, err := rs.clientForRequest(r)
	if err != nil {
		rs.sendError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err := c.TestConnection(); err != nil {
		rs.logger.Error("ADT connection test failed", zap.Error(err))
		rs.sendError(w, "Connection failed: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	connectionStatus := map[string]any{
		"status":        "connected",
		"authenticated": c.IsAuthenticated(),
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

	poolSize := 0
	if rs.pool != nil {
		poolSize = rs.pool.Size()
		if poolSize > 0 {
			adtStatus = "connected"
		}
	}

	health := map[string]any{
		"status":           "healthy",
		"timestamp":        time.Now().UTC(),
		"adt_status":       adtStatus,
		"pool_connections": poolSize,
		"features": map[string]bool{
			"quiet_mode_default": true,
			"file_logging":       true,
			"adt_integration":    true,
			"cli_parity":         true,
			"ai_removed":         true,
			"connection_pool":    rs.pool != nil,
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
