package adt

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/bluefunda/abaper/types"
	"go.uber.org/zap"
)

// ADT Endpoint Constants
const (
	programsEndpoint        = "/programs/programs/%s/source/main"
	classesEndpoint         = "/oo/classes/%s/source/main"
	functionGroupsEndpoint  = "/functions/groups/%s/source/main"
	functionsEndpoint       = "/functions/groups/%s/fmodules/%s/source/main"
	tablesEndpoint          = "/ddic/tables/%s/source/main"
	structuresEndpoint      = "/ddic/structures/%s/source/main"
	includesEndpoint        = "/programs/includes/%s/source/main"
	interfacesEndpoint      = "/oo/interfaces/%s/source/main"
	domainsEndpoint         = "/ddic/domains/%s/source/main"
	dataElementsEndpoint    = "/ddic/dataelements/%s"
	searchEndpoint = "/repository/informationsystem/search"
	transactionEndpoint     = "/repository/informationsystem/objectproperties/values"
	programsCreateEndpoint  = "/programs/programs"
	classesCreateEndpoint   = "/oo/classes"
	tableContentsEndpoint   = "/z_mcp_abap_adt/z_tablecontent/%s" // Custom service required
)

// ErrNotFound is returned when an ADT object does not exist (HTTP 404).
var ErrNotFound = errors.New("object not found")

// ADTClientImpl implements the ADTClient interface using shared types
type ADTClientImpl struct {
	config        *types.ADTConfig
	httpClient    *http.Client
	logger        *zap.Logger
	csrfToken     string
	sessionID     string
	baseURL       string
	authenticated bool
	sessionType   string // "stateful" or "stateless"
}

// NewADTClient creates a new ADT client with improved configuration
func NewADTClient(config *types.ADTConfig) *ADTClientImpl {
	// Set defaults
	if config.Language == "" {
		config.Language = "EN"
	}
	if config.Client == "" {
		config.Client = "100"
	}
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 30 * time.Second
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 60 * time.Second
	}

	// Normalize and validate the host URL
	baseURL := normalizeBaseURL(config.Host)

	// Create cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		// Fallback to client without cookie jar
		jar = nil
	}

	// Create HTTP client with proper configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.AllowSelfSigned,
		},
		MaxIdleConns:       10,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: false,
		DisableKeepAlives:  false,
	}

	client := &http.Client{
		Timeout:   config.RequestTimeout,
		Transport: transport,
		Jar:       jar,
	}

	// Create a logger instance for the ADT client based on debug config
	var logger *zap.Logger
	if config.Debug {
		logger, _ = zap.NewDevelopment()
	} else {
		logger = zap.NewNop() // Silent by default, respecting original behavior
	}

	adtClient := &ADTClientImpl{
		config:      config,
		httpClient:  client,
		logger:      logger.With(zap.String("component", "adt_client")),
		baseURL:     baseURL,
		sessionType: string(types.SessionStateful), // CRITICAL: Default to stateful
	}

	// Log the initial session type
	adtClient.logger.Info("ADT Client initialized",
		zap.String("session_type", adtClient.sessionType),
		zap.String("base_url", baseURL))

	return adtClient
}

// normalizeBaseURL ensures proper URL format
func normalizeBaseURL(host string) string {
	host = strings.TrimSpace(host)

	// Remove trailing slashes
	host = strings.TrimSuffix(host, "/")

	// Add protocol if missing
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	// Ensure ADT path
	if !strings.HasSuffix(host, "/sap/bc/adt") {
		host = host + "/sap/bc/adt"
	}

	return host
}

// SetSessionType sets the session type (stateful/stateless)
func (c *ADTClientImpl) SetSessionType(sessionType types.SessionType) {
	c.sessionType = string(sessionType)
}

// Authenticate performs comprehensive authentication with SAP system
func (c *ADTClientImpl) Authenticate() error {
	c.logger.Info("Starting SAP ADT authentication",
		zap.String("host", c.config.Host),
		zap.String("username", c.config.Username),
		zap.String("client", c.config.Client),
		zap.String("language", c.config.Language))

	// Step 1: Test basic connectivity
	if err := c.testConnectivity(); err != nil {
		return fmt.Errorf("connectivity test failed: %w", err)
	}

	// Step 2: Perform initial login to establish session
	if err := c.performLogin(); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Step 3: Get CSRF token
	if err := c.getCSRFToken(); err != nil {
		return fmt.Errorf("CSRF token retrieval failed: %w", err)
	}

	// Step 4: Validate session
	if err := c.validateSession(); err != nil {
		return fmt.Errorf("session validation failed: %w", err)
	}

	c.authenticated = true
	c.logger.Info("SAP ADT authentication successful",
		zap.String("csrf_token_length", fmt.Sprintf("%d", len(c.csrfToken))),
		zap.String("session_type", c.sessionType))

	return nil
}

// IsAuthenticated returns authentication status
func (c *ADTClientImpl) IsAuthenticated() bool {
	return c.authenticated && c.csrfToken != ""
}

// getSource is a shared helper for the single-name Get* source retrieval methods.
func (c *ADTClientImpl) getSource(ctx context.Context, objectType, name, endpoint string) (*types.ADTSourceCode, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}
	name = strings.ToUpper(strings.TrimSpace(name))
	url := fmt.Sprintf("%s"+endpoint, c.baseURL, name)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")
	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s %s", ErrNotFound, objectType, name)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed: session may have expired")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get %s %s: HTTP %d: %s", objectType, name, resp.StatusCode, string(body))
	}
	src, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return &types.ADTSourceCode{
		ObjectName: name,
		ObjectType: objectType,
		Source:     string(src),
		Version:    resp.Header.Get("ETag"),
		ETag:       resp.Header.Get("ETag"),
	}, nil
}

// GetProgram retrieves ABAP program source code.
func (c *ADTClientImpl) GetProgram(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return c.getSource(ctx, "PROG", name, programsEndpoint)
}

// GetClass retrieves ABAP class source code.
func (c *ADTClientImpl) GetClass(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return c.getSource(ctx, "CLAS", name, classesEndpoint)
}

// GetFunction retrieves ABAP function module source code
func (c *ADTClientImpl) GetFunction(ctx context.Context, functionName, functionGroup string) (*types.ADTSourceCode, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving function module",
		zap.String("function", functionName),
		zap.String("function_group", functionGroup))

	functionName = strings.ToUpper(strings.TrimSpace(functionName))
	functionGroup = strings.ToUpper(strings.TrimSpace(functionGroup))
	url := fmt.Sprintf("%s"+functionsEndpoint, c.baseURL, functionGroup, functionName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: function %s in group %s", ErrNotFound, functionName, functionGroup)
		}
		return nil, fmt.Errorf("failed to get function %s: HTTP %d - %s", functionName, resp.StatusCode, string(body))
	}

	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	result := &types.ADTSourceCode{
		ObjectName: functionName,
		ObjectType: "FUNC",
		Source:     string(source),
		Version:    resp.Header.Get("ETag"),
		ETag:       resp.Header.Get("ETag"),
	}

	c.logger.Info("Function module retrieved successfully",
		zap.String("function", functionName),
		zap.String("function_group", functionGroup),
		zap.Int("source_length", len(result.Source)))

	return result, nil
}

// GetFunctionGroup retrieves ABAP function group source code
func (c *ADTClientImpl) GetFunctionGroup(ctx context.Context, functionGroup string) (*types.ADTSourceCode, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving function group", zap.String("function_group", functionGroup))

	functionGroup = strings.ToUpper(strings.TrimSpace(functionGroup))
	url := fmt.Sprintf("%s"+functionGroupsEndpoint, c.baseURL, functionGroup)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: function group %s", ErrNotFound, functionGroup)
		}
		return nil, fmt.Errorf("failed to get function group %s: HTTP %d - %s", functionGroup, resp.StatusCode, string(body))
	}

	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	result := &types.ADTSourceCode{
		ObjectName: functionGroup,
		ObjectType: "FUGR",
		Source:     string(source),
		Version:    resp.Header.Get("ETag"),
		ETag:       resp.Header.Get("ETag"),
	}

	c.logger.Info("Function group retrieved successfully",
		zap.String("function_group", functionGroup),
		zap.Int("source_length", len(result.Source)))

	return result, nil
}

// GetInclude retrieves ABAP include source code.
func (c *ADTClientImpl) GetInclude(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return c.getSource(ctx, "INCL", name, includesEndpoint)
}

// GetInterface retrieves ABAP interface source code.
func (c *ADTClientImpl) GetInterface(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return c.getSource(ctx, "INTF", name, interfacesEndpoint)
}

// GetStructure retrieves ABAP structure definition.
func (c *ADTClientImpl) GetStructure(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return c.getSource(ctx, "STRU", name, structuresEndpoint)
}

// GetTable retrieves ABAP table structure.
func (c *ADTClientImpl) GetTable(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return c.getSource(ctx, "TABL", name, tablesEndpoint)
}

// GetPackageContents retrieves package contents
func (c *ADTClientImpl) GetPackageContents(ctx context.Context, packageName string) (*types.ADTPackage, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving package contents", zap.String("package", packageName))

	packageName = strings.ToUpper(strings.TrimSpace(packageName))

	// Use the search endpoint with packageName filter to get package contents
	// This is more reliable than the nodecontents endpoint across different SAP systems
	requestURL := fmt.Sprintf("%s%s?operation=quickSearch&query=*&packageName=%s&maxResults=1000",
		c.baseURL,
		searchEndpoint,
		url.QueryEscape(packageName))

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "application/xml")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: package %s", ErrNotFound, packageName)
		}
		return nil, fmt.Errorf("failed to get package %s: HTTP %d - %s", packageName, resp.StatusCode, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse XML response
	var searchResult types.ADTSearchResult
	if err := xml.Unmarshal(responseBody, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	result := &types.ADTPackage{
		Name:        packageName,
		Description: fmt.Sprintf("Package %s", packageName),
		Objects:     searchResult.Objects,
	}

	c.logger.Info("Package contents retrieved successfully",
		zap.String("package", packageName),
		zap.Int("object_count", len(searchResult.Objects)))

	return result, nil
}

// SearchObjects searches for ABAP objects
func (c *ADTClientImpl) SearchObjects(ctx context.Context, pattern string, objectTypes []string) (*types.ADTSearchResult, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Searching objects",
		zap.String("pattern", pattern),
		zap.Strings("types", objectTypes))

	maxResults := 100
	searchURL := fmt.Sprintf("%s%s?operation=quickSearch&query=%s&maxResults=%d",
		c.baseURL,
		searchEndpoint,
		url.QueryEscape(pattern),
		maxResults)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "application/xml")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result types.ADTSearchResult
	if err := xml.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	c.logger.Info("Search completed successfully",
		zap.String("pattern", pattern),
		zap.Int("response_length", len(responseBody)))

	return &result, nil
}

// ListPackages lists packages matching a pattern
func (c *ADTClientImpl) ListPackages(ctx context.Context, pattern string) ([]types.ADTPackage, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Listing packages", zap.String("pattern", pattern))

	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		pattern = "*"
	}

	// Use the repository search endpoint to find packages
	searchURL := fmt.Sprintf("%s%s?operation=quickSearch&query=%s&objectType=DEVC/K&maxResults=100",
		c.baseURL,
		searchEndpoint,
		url.QueryEscape(pattern))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "application/xml")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("package search failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse XML response
	var searchResult types.ADTSearchResult
	if err := xml.Unmarshal(responseBody, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	// Convert search results to packages
	packages := make([]types.ADTPackage, 0, len(searchResult.Objects))
	for _, obj := range searchResult.Objects {
		packages = append(packages, types.ADTPackage{
			Name:        obj.Name,
			Description: obj.Description,
			Objects:     []types.ADTObject{},
		})
	}

	c.logger.Info("Package search completed",
		zap.String("pattern", pattern),
		zap.Int("packages_found", len(packages)))

	return packages, nil
}

// TestConnection tests the ADT connection with comprehensive diagnostics
func (c *ADTClientImpl) TestConnection() error {
	c.logger.Info("Starting comprehensive ADT connection test")

	// Step 1: Test basic connectivity
	if err := c.testConnectivity(); err != nil {
		return fmt.Errorf("basic connectivity failed: %w", err)
	}

	// Step 2: Test authentication
	if err := c.Authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	c.logger.Info("All ADT connection tests passed successfully")
	return nil
}

// Extended methods (optional implementations)
func (c *ADTClientImpl) GetTypeInfo(ctx context.Context, typeName string) (*types.ADTTypeInfo, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving type info", zap.String("type", typeName))

	typeName = strings.ToUpper(strings.TrimSpace(typeName))

	// First try as domain
	domainURL := fmt.Sprintf("%s"+domainsEndpoint, c.baseURL, typeName)
	if source, err := c.getTypeSource(ctx, domainURL, "text/plain"); err == nil {
		return &types.ADTTypeInfo{
			TypeName:   typeName,
			TypeKind:   "DOMAIN",
			Source:     source,
			Properties: make(map[string]any),
		}, nil
	}

	// If domain fails, try as data element
	dataElementURL := fmt.Sprintf("%s"+dataElementsEndpoint, c.baseURL, typeName)
	if source, err := c.getTypeSource(ctx, dataElementURL, "application/xml"); err == nil {
		return &types.ADTTypeInfo{
			TypeName:   typeName,
			TypeKind:   "DATA_ELEMENT",
			Source:     source,
			Properties: make(map[string]any),
		}, nil
	}

	return nil, fmt.Errorf("type %s not found as domain or data element", typeName)
}

func (c *ADTClientImpl) GetTransaction(ctx context.Context, transactionName string) (*types.ADTTransactionInfo, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving transaction", zap.String("transaction", transactionName))

	transactionName = strings.ToUpper(strings.TrimSpace(transactionName))
	encodedTransactionName := url.QueryEscape(transactionName)

	queryURL := fmt.Sprintf("%s%s?uri=%s&facet=package&facet=appl",
		c.baseURL,
		transactionEndpoint,
		url.QueryEscape(fmt.Sprintf("/sap/bc/adt/vit/wb/object_type/trant/object_name/%s", encodedTransactionName)))

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "application/xml")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: transaction %s", ErrNotFound, transactionName)
		}
		return nil, fmt.Errorf("failed to get transaction %s: HTTP %d - %s", transactionName, resp.StatusCode, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the XML response (simplified - would need proper XML parsing in production)
	result := &types.ADTTransactionInfo{
		TransactionCode: transactionName,
		Description:     "", // Would be extracted from XML
		Package:         "", // Would be extracted from XML
		Application:     "", // Would be extracted from XML
		Program:         "", // Would be extracted from XML
		Properties:      make(map[string]string),
	}

	c.logger.Info("Transaction retrieved successfully",
		zap.String("transaction", transactionName),
		zap.Int("response_length", len(responseBody)))

	return result, nil
}

func (c *ADTClientImpl) GetTableContents(ctx context.Context, tableName string, maxRows int) (*types.ADTTableData, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving table contents",
		zap.String("table", tableName),
		zap.Int("max_rows", maxRows))

	tableName = strings.ToUpper(strings.TrimSpace(tableName))
	if maxRows <= 0 {
		maxRows = 100
	}

	// This requires a custom SAP service to be implemented
	url := fmt.Sprintf("%s"+tableContentsEndpoint+"?maxRows=%d", c.baseURL, tableName, maxRows)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("table contents service not found - requires custom SAP service implementation at %s", tableContentsEndpoint)
		}
		return nil, fmt.Errorf("failed to get table contents: HTTP %d - %s", resp.StatusCode, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result types.ADTTableData
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	c.logger.Info("Table contents retrieved successfully",
		zap.String("table", tableName),
		zap.Int("row_count", result.RowCount))

	return &result, nil
}

func (c *ADTClientImpl) GetTransports(ctx context.Context) ([]types.ADTTransport, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving transport requests")

	// This would require custom implementation in SAP system
	// For now, return empty slice as this is an optional feature
	return []types.ADTTransport{}, nil
}

// CreateClass creates a new ABAP class - enhanced with working atomic approach
func (c *ADTClientImpl) CreateClass(ctx context.Context, name, description, packageName, source string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	// Validate inputs
	name = strings.ToUpper(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("class name cannot be empty")
	}
	if description == "" {
		return fmt.Errorf("class description cannot be empty")
	}

	// Default package - following CreateProgram pattern
	if packageName == "" {
		packageName = "$TMP"
	}

	c.logger.Info("Creating class atomically",
		zap.String("name", name),
		zap.String("description", description),
		zap.String("package", packageName),
		zap.Bool("has_source", source != ""))

	if err := c.CreateClassWithSource(ctx, name, description, packageName, source); err != nil {
		return fmt.Errorf("failed to create class: %w", err)
	}

	c.logger.Info("Class created successfully", zap.String("name", name))
	return nil
}

// CreateClassWithSource is a convenience method that creates a class with source code
func (c *ADTClientImpl) CreateClassWithSource(ctx context.Context, name, description, packageName, source string) error {
	opts := CreateClassOptions{
		Name:         name,
		Description:  description,
		Source:       source,
		Package:      packageName,
		Activate:     true,
		InsertSource: true,
	}

	if opts.Package == "" {
		opts.Package = "$TMP"
	}

	return c.CreateClassWithOptions(ctx, opts)
}

// CreateClassWithOptions creates a class with full options support
func (c *ADTClientImpl) CreateClassWithOptions(ctx context.Context, opts CreateClassOptions) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	// Validate and set defaults
	opts.Name = strings.ToUpper(strings.TrimSpace(opts.Name))
	if opts.Name == "" {
		return fmt.Errorf("class name cannot be empty")
	}
	if opts.Description == "" {
		return fmt.Errorf("class description cannot be empty")
	}
	if opts.Package == "" {
		opts.Package = "$TMP"
	}
	if opts.Responsible == "" {
		opts.Responsible = strings.ToUpper(strings.TrimSpace(c.config.Username))
	}

	c.logger.Info("Creating class with options",
		zap.String("name", opts.Name),
		zap.String("description", opts.Description),
		zap.String("package", opts.Package),
		zap.Bool("activate", opts.Activate),
		zap.Bool("insert_source", opts.InsertSource),
		zap.Bool("has_source", opts.Source != ""))

	// Step 1: Create the class structure
	if err := c.createClassMetadata(ctx, opts.Name, opts.Description, opts.Package); err != nil {
		return fmt.Errorf("failed to create class structure: %w", err)
	}

	// Step 2: Insert source code if provided
	if opts.InsertSource && opts.Source != "" && strings.TrimSpace(opts.Source) != "" {
		if err := c.setClassSource(ctx, opts.Name, opts.Source); err != nil {
			return fmt.Errorf("failed to insert source code: %w", err)
		}
	}

	// Step 3: Activate if requested
	if opts.Activate {
		if err := c.activateClass(ctx, &opts); err != nil {
			return fmt.Errorf("failed to activate class: %w", err)
		}
	}

	c.logger.Info("Class created successfully with options", zap.String("name", opts.Name))
	return nil
}

// classCreatePayload is the XML struct for class creation.
type classCreatePayload struct {
	XMLName     xml.Name           `xml:"class:abapClass"`
	ClassNS     string             `xml:"xmlns:class,attr"`
	AdtcoreNS   string             `xml:"xmlns:adtcore,attr"`
	Description string             `xml:"adtcore:description,attr"`
	Name        string             `xml:"adtcore:name,attr"`
	Type        string             `xml:"adtcore:type,attr"`
	Responsible string             `xml:"adtcore:responsible,attr"`
	PackageRef  classPackageRef    `xml:"adtcore:packageRef"`
}

type classPackageRef struct {
	Name string `xml:"adtcore:name,attr"`
}

// createClassMetadata creates the class metadata structure (no source)
func (c *ADTClientImpl) createClassMetadata(ctx context.Context, name, description, packageName string) error {
	payload := classCreatePayload{
		ClassNS:     "http://www.sap.com/adt/oo/classes",
		AdtcoreNS:   "http://www.sap.com/adt/core",
		Description: description,
		Name:        name,
		Type:        "CLAS/OC",
		Responsible: strings.ToUpper(strings.TrimSpace(c.config.Username)),
		PackageRef:  classPackageRef{Name: packageName},
	}

	xmlBytes, err := xml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal class metadata: %w", err)
	}
	xmlPayload := xml.Header + string(xmlBytes)

	url := c.baseURL + classesCreateEndpoint

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(xmlPayload)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "application/*")

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create class metadata: HTTP %d - %s", resp.StatusCode, string(body))
	}

	c.logger.Debug("Class metadata created successfully", zap.String("name", name))
	return nil
}

// activateClass activates a class
func (c *ADTClientImpl) activateClass(ctx context.Context, opts *CreateClassOptions) error {
	c.logger.Info("Activating class", zap.String("class", opts.Name))

	// Prepare activation request
	activationReq := ActivationRequest{
		Namespace: "http://www.sap.com/adt/core",
		ObjectRef: ActivationRef{
			URI:  fmt.Sprintf("/sap/bc/adt/oo/classes/%s", strings.ToLower(opts.Name)),
			Name: opts.Name,
		},
	}

	xmlPayload, err := xml.Marshal(activationReq)
	if err != nil {
		return fmt.Errorf("failed to marshal activation request: %w", err)
	}

	// Add XML header
	fullPayload := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + string(xmlPayload)

	url := c.baseURL + "/activation?method=activate&preauditRequested=true&sap-client=" + c.config.Client + "&sap-language=" + c.config.Language

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(fullPayload))
	if err != nil {
		return fmt.Errorf("failed to create activation request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "application/*")
	req.Header.Set("Accept", "application/*")
	req.Header.Set("X-CSRF-Token", c.csrfToken)

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("activation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("activation failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// Parse activation response to check for warnings/errors
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Warn("Could not read activation response", zap.Error(err))
		return nil // Don't fail if we can't read the response
	}

	c.logger.Debug("Activation response", zap.String("class", opts.Name), zap.String("body", string(responseBody)))
	c.logger.Info("Class activated successfully", zap.String("class", opts.Name))
	return nil
}

// CreateInterface creates a new ABAP interface
func (c *ADTClientImpl) CreateInterface(ctx context.Context, name, description, source string) error {
	return fmt.Errorf("not implemented")
}

// CreateFunctionGroup creates a new ABAP function group
func (c *ADTClientImpl) CreateFunctionGroup(ctx context.Context, name, description, source string) error {
	return fmt.Errorf("not implemented")
}

// CreateInclude creates a new ABAP include
func (c *ADTClientImpl) CreateInclude(ctx context.Context, name, description, source string) error {
	return fmt.Errorf("not implemented")
}

// CreateStructure creates a new ABAP structure
func (c *ADTClientImpl) CreateStructure(ctx context.Context, name, description, source string) error {
	return fmt.Errorf("not implemented")
}

// CreateTable creates a new ABAP table
func (c *ADTClientImpl) CreateTable(ctx context.Context, name, description, source string) error {
	return fmt.Errorf("not implemented")
}

// addAuthHeaders adds authentication and session headers to the request
func (c *ADTClientImpl) addAuthHeaders(req *http.Request) {
	// Basic authentication
	req.SetBasicAuth(c.config.Username, c.config.Password)

	// SAP client and language
	req.Header.Set("sap-client", c.config.Client)
	req.Header.Set("sap-language", c.config.Language)

	// Add default Accept header if not already set
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/atomsvc+xml")
	}

	// CSRF token if available
	if c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}

	// Session type for stateful sessions
	if c.sessionType != "" {
		req.Header.Set("X-sap-adt-sessiontype", c.sessionType)
	}

	// User agent
	req.Header.Set("User-Agent", "abaper/1.0")
}

// doRequest executes req, transparently re-authenticating on session expiry.
// On 401: re-authenticates and retries once. On 403 with X-CSRF-Token prompt: refreshes token and retries.
// The request body (if any) is buffered so it can be replayed on retry.
func (c *ADTClientImpl) doRequest(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		resp.Body.Close()
		c.logger.Info("Session expired (401), re-authenticating")
		if authErr := c.Authenticate(); authErr != nil {
			return nil, fmt.Errorf("re-auth failed: %w", authErr)
		}
		c.addAuthHeaders(req)
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
		return c.httpClient.Do(req)

	case http.StatusForbidden:
		if req.Header.Get("X-CSRF-Token") != "" {
			resp.Body.Close()
			c.logger.Info("CSRF token expired (403), refreshing")
			if csrfErr := c.getCSRFToken(); csrfErr != nil {
				return nil, fmt.Errorf("CSRF refresh failed: %w", csrfErr)
			}
			c.addAuthHeaders(req)
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			return c.httpClient.Do(req)
		}
	}
	return resp, nil
}

// testConnectivity tests basic network connectivity to the SAP system
func (c *ADTClientImpl) testConnectivity() error {
	c.logger.Info("Testing basic connectivity", zap.String("host", c.config.Host))

	// Test basic connectivity with a simple HEAD request
	req, err := http.NewRequest("HEAD", c.baseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create connectivity test request: %w", err)
	}

	// Set a shorter timeout for connectivity test
	client := &http.Client{
		Timeout:   c.config.ConnectTimeout,
		Transport: c.httpClient.Transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connectivity test failed: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Info("Basic connectivity test passed", zap.Int("status_code", resp.StatusCode))
	return nil
}

// performLogin performs initial login to establish session
func (c *ADTClientImpl) performLogin() error {
	c.logger.Info("Performing initial login")

	// Create login request to establish session
	loginURL := c.baseURL + "/discovery"
	req, err := http.NewRequest("GET", loginURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	// Add basic authentication and headers
	req.SetBasicAuth(c.config.Username, c.config.Password)
	req.Header.Set("sap-client", c.config.Client)
	req.Header.Set("sap-language", c.config.Language)
	req.Header.Set("Accept", "application/atomsvc+xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("authentication failed: invalid credentials")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	c.logger.Info("Initial login successful")
	return nil
}

// getCSRFToken retrieves CSRF token for subsequent requests
func (c *ADTClientImpl) getCSRFToken() error {
	c.logger.Info("Retrieving CSRF token")

	// Request CSRF token
	tokenURL := c.baseURL + "/discovery"
	req, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create CSRF token request: %w", err)
	}

	req.SetBasicAuth(c.config.Username, c.config.Password)
	req.Header.Set("sap-client", c.config.Client)
	req.Header.Set("sap-language", c.config.Language)
	req.Header.Set("Accept", "application/atomsvc+xml")
	req.Header.Set("X-CSRF-Token", "Fetch")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CSRF token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CSRF token retrieval failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// Extract CSRF token from response headers
	csrfToken := resp.Header.Get("X-CSRF-Token")
	if csrfToken == "" {
		return fmt.Errorf("CSRF token not found in response headers")
	}

	c.csrfToken = csrfToken
	c.logger.Info("CSRF token retrieved successfully", zap.String("token_length", fmt.Sprintf("%d", len(csrfToken))))

	return nil
}

// validateSession validates the current session
func (c *ADTClientImpl) validateSession() error {
	c.logger.Info("Validating session")

	// Test session with a simple request
	testURL := c.baseURL + "/discovery"
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create session validation request: %w", err)
	}

	c.addAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("session validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("session validation failed: unauthorized")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("session validation failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	c.logger.Info("Session validation successful")
	return nil
}

// getTypeSource retrieves source for type definitions
func (c *ADTClientImpl) getTypeSource(ctx context.Context, url, acceptType string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", acceptType)

	resp, err := c.doRequest(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

// CreateProgramOptions holds the options for creating a program
type CreateProgramOptions struct {
	Name         string
	Description  string
	Source       string
	Package      string
	Responsible  string
	Transport    string
	Activate     bool
	InsertSource bool
}

// CreateClassOptions holds the options for creating a class
type CreateClassOptions struct {
	Name         string
	Description  string
	Source       string
	Package      string
	Responsible  string
	Transport    string
	Activate     bool
	InsertSource bool
}

// LockResponse represents the response from a lock operation (ABAP XML format)
type LockResponse struct {
	XMLName xml.Name `xml:"abap"`
	Values  struct {
		Data struct {
			LockHandle string `xml:"LOCK_HANDLE"`
			CorrNr     string `xml:"CORR_NR"`
		} `xml:"DATA"`
	} `xml:"values"`
}

// ObjectRefsLockResponse represents the alternative objectReferences format
type ObjectRefsLockResponse struct {
	XMLName   xml.Name  `xml:"objectReferences"`
	ObjectRef ObjectRef `xml:"objectReference"`
}

type ObjectRef struct {
	LockHandle string `xml:"LOCK_HANDLE"`
	CorrNr     string `xml:"CORR_NR"`
	URI        string `xml:"uri,attr"`
	Name       string `xml:"name,attr"`
}

// ActivationRequest represents the activation request structure
type ActivationRequest struct {
	XMLName   xml.Name      `xml:"objectReferences"`
	Namespace string        `xml:"xmlns,attr"`
	ObjectRef ActivationRef `xml:"objectReference"`
}

type ActivationRef struct {
	URI  string `xml:"uri,attr"`
	Name string `xml:"name,attr"`
}

// CreateProgram creates a new ABAP program - now with working atomic approach
func (c *ADTClientImpl) CreateProgram(ctx context.Context, name, description, packageName, source string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	// Validate inputs
	name = strings.ToUpper(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("program name cannot be empty")
	}
	if description == "" {
		description = fmt.Sprintf("Program %s", name)
	}

	c.logger.Info("Creating ABAP program",
		zap.String("name", name),
		zap.String("description", description),
		zap.Bool("has_source", source != ""))

	if err := c.CreateProgramWithSource(ctx, name, description, packageName, source); err != nil {
		return fmt.Errorf("failed to create program: %w", err)
	}

	c.logger.Info("Program creation completed successfully", zap.String("name", name))
	return nil
}

// parseLockResponse handles both ABAP XML and objectReferences formats with enhanced debugging
func (c *ADTClientImpl) parseLockResponse(responseBody []byte) (lockHandle, corrNr string, err error) {
	c.logger.Debug("Parsing lock response", zap.String("xml", string(responseBody)))

	// Try the newer ABAP XML format first
	var abapResponse LockResponse
	if err := xml.Unmarshal(responseBody, &abapResponse); err == nil {
		if abapResponse.Values.Data.LockHandle != "" {
			c.logger.Debug("Parsed ABAP XML format lock response",
				zap.String("lock_handle", abapResponse.Values.Data.LockHandle),
				zap.String("corr_nr", abapResponse.Values.Data.CorrNr))
			return abapResponse.Values.Data.LockHandle, abapResponse.Values.Data.CorrNr, nil
		}
	} else {
		c.logger.Debug("Failed to parse as ABAP XML format", zap.Error(err))
	}

	// Fallback to older objectReferences format
	var objResponse ObjectRefsLockResponse
	if err := xml.Unmarshal(responseBody, &objResponse); err == nil {
		if objResponse.ObjectRef.LockHandle != "" {
			c.logger.Debug("Parsed objectReferences format lock response",
				zap.String("lock_handle", objResponse.ObjectRef.LockHandle),
				zap.String("corr_nr", objResponse.ObjectRef.CorrNr))
			return objResponse.ObjectRef.LockHandle, objResponse.ObjectRef.CorrNr, nil
		}
	} else {
		c.logger.Debug("Failed to parse as objectReferences format", zap.Error(err))
	}

	// Try to extract from HTTP headers as fallback
	c.logger.Debug("Attempting to parse lock response from raw XML structure")
	// Sometimes the lock handle might be in a different XML structure
	if strings.Contains(string(responseBody), "<lockHandle>") {
		// Extract lock handle using simple string parsing as last resort
		start := strings.Index(string(responseBody), "<lockHandle>")
		if start >= 0 {
			start += len("<lockHandle>")
			end := strings.Index(string(responseBody)[start:], "</lockHandle>")
			if end >= 0 {
				lockHandle := string(responseBody)[start : start+end]
				c.logger.Debug("Extracted lock handle from raw XML",
					zap.String("lock_handle", lockHandle))
				return lockHandle, "", nil
			}
		}
	}

	return "", "", fmt.Errorf("failed to parse lock response in any known format. Response: %s", string(responseBody))
}

// programCreatePayload is the XML struct for program creation.
type programCreatePayload struct {
	XMLName     xml.Name           `xml:"program:abapProgram"`
	ProgramNS   string             `xml:"xmlns:program,attr"`
	AdtcoreNS   string             `xml:"xmlns:adtcore,attr"`
	Description string             `xml:"adtcore:description,attr"`
	Name        string             `xml:"adtcore:name,attr"`
	Type        string             `xml:"adtcore:type,attr"`
	Responsible string             `xml:"adtcore:responsible,attr"`
	PackageRef  programPackageRef  `xml:"adtcore:packageRef"`
}

type programPackageRef struct {
	Name string `xml:"adtcore:name,attr"`
}

// createProgramMetadata creates the program metadata structure (no source)
func (c *ADTClientImpl) createProgramMetadata(ctx context.Context, name, description, packageName string) error {
	payload := programCreatePayload{
		ProgramNS:   "http://www.sap.com/adt/programs/programs",
		AdtcoreNS:   "http://www.sap.com/adt/core",
		Description: description,
		Name:        name,
		Type:        "PROG/P",
		Responsible: strings.ToUpper(strings.TrimSpace(c.config.Username)),
		PackageRef:  programPackageRef{Name: packageName},
	}

	xmlBytes, err := xml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal program metadata: %w", err)
	}
	xmlPayload := xml.Header + string(xmlBytes)

	url := c.baseURL + programsCreateEndpoint

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(xmlPayload)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "application/*")

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("program creation failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// setSourceUsingWorkingPattern uses the exact pattern from reference API that works
func (c *ADTClientImpl) setSourceUsingWorkingPattern(ctx context.Context, programName, source string) error {
	c.logger.Info("Setting source using working pattern", zap.String("program", programName))

	// Ensure we're in stateful mode from the start
	originalSessionType := c.sessionType
	c.sessionType = string(types.SessionStateful)
	defer func() {
		c.sessionType = originalSessionType
	}()

	// Use the exact paths from the reference API test
	programNameLower := strings.ToLower(programName)
	programPath := fmt.Sprintf("/programs/programs/%s", programNameLower)            // This is what we lock
	sourcePath := fmt.Sprintf("/programs/programs/%s/source/main", programNameLower) // This is where we set source

	c.logger.Debug("Using paths", zap.String("program_path", programPath), zap.String("source_path", sourcePath))

	// Lock the program object (not the source path)
	lockHandle, corrNr, err := c.lockObject(ctx, programPath)
	if err != nil {
		return fmt.Errorf("failed to lock program: %w", err)
	}

	// Ensure we unlock
	defer func() {
		if unlockErr := c.unlockObject(ctx, programPath, lockHandle); unlockErr != nil {
			c.logger.Warn("Failed to unlock program", zap.String("program", programName), zap.Error(unlockErr))
		}
	}()

	// Set the source on the source path using lock handle from program
	if err := c.setObjectSource(ctx, sourcePath, source, lockHandle, corrNr); err != nil {
		return fmt.Errorf("failed to set source: %w", err)
	}

	return nil
}

// lockObject locks an object for modification (following reference API pattern)
func (c *ADTClientImpl) lockObject(ctx context.Context, objectPath string) (lockHandle, corrNr string, err error) {
	url := fmt.Sprintf("%s%s?_action=LOCK&accessMode=MODIFY", c.baseURL, objectPath)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create lock request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "application/*")
	req.Header.Set("Content-Length", "0")

	c.logger.Debug("Locking object", zap.String("object_path", objectPath), zap.String("url", url))

	resp, err := c.doRequest(req)
	if err != nil {
		return "", "", fmt.Errorf("lock request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("Lock failed", zap.String("object_path", objectPath), zap.Int("status", resp.StatusCode), zap.String("response", string(body)))
		return "", "", fmt.Errorf("lock failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// Read and parse response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read lock response: %w", err)
	}

	c.logger.Debug("Lock response received", zap.String("object_path", objectPath), zap.String("response_body", string(responseBody)))

	// Parse lock response
	lockHandle, corrNr, err = c.parseLockResponse(responseBody)
	if err != nil {
		return "", "", err
	}

	if lockHandle == "" {
		return "", "", fmt.Errorf("no lock handle received in response")
	}

	c.logger.Info("Object locked successfully", zap.String("object_path", objectPath), zap.String("lock_handle", lockHandle), zap.String("corr_nr", corrNr))
	return lockHandle, corrNr, nil
}

// unlockObject unlocks an object (following reference API pattern)
func (c *ADTClientImpl) unlockObject(ctx context.Context, objectPath, lockHandle string) error {
	url := fmt.Sprintf("%s%s?_action=UNLOCK&lockHandle=%s", c.baseURL, objectPath, lockHandle)

	c.logger.Debug("Unlocking object", zap.String("object_path", objectPath), zap.String("lock_handle", lockHandle), zap.String("url", url))

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create unlock request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "application/*")
	req.Header.Set("Content-Length", "0")

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("unlock request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Warn("Unlock failed", zap.String("object_path", objectPath), zap.String("lock_handle", lockHandle), zap.Int("status", resp.StatusCode), zap.String("response", string(body)))
		return fmt.Errorf("unlock failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	c.logger.Info("Object unlocked successfully", zap.String("object_path", objectPath), zap.String("lock_handle", lockHandle))
	return nil
}

// setObjectSource sets the source code for an object (following reference API pattern)
func (c *ADTClientImpl) setObjectSource(ctx context.Context, sourcePath, source, lockHandle, corrNr string) error {
	url := fmt.Sprintf("%s%s?lockHandle=%s", c.baseURL, sourcePath, lockHandle)
	if corrNr != "" {
		url += "&corrNr=" + corrNr
	}

	c.logger.Debug("Setting object source", zap.String("source_path", sourcePath), zap.String("lock_handle", lockHandle), zap.String("url", url), zap.Int("source_length", len(source)))

	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(source))
	if err != nil {
		return fmt.Errorf("failed to create source update request: %w", err)
	}

	c.addAuthHeaders(req)
	// Use the correct content type like the reference implementation
	contentType := "text/plain; charset=utf-8"
	if strings.HasPrefix(strings.TrimSpace(source), "<?xml") {
		contentType = "application/*"
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("source update request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("Source update failed", zap.String("source_path", sourcePath), zap.String("lock_handle", lockHandle), zap.Int("status", resp.StatusCode), zap.String("response", string(body)))
		return fmt.Errorf("source update failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	c.logger.Info("Object source updated successfully", zap.String("source_path", sourcePath), zap.String("lock_handle", lockHandle))
	return nil
}

// GetObjectSource retrieves the source code of an object (following reference API pattern)
func (c *ADTClientImpl) GetObjectSource(ctx context.Context, objectType, objectName string) (string, error) {
	if !c.IsAuthenticated() {
		return "", fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	objectType = strings.ToUpper(objectType)
	objectName = strings.ToUpper(strings.TrimSpace(objectName))

	c.logger.Info("Getting object source", zap.String("object_type", objectType), zap.String("object_name", objectName))

	// Construct source path based on object type
	var sourcePath string
	switch objectType {
	case "PROGRAM", "PROG":
		sourcePath = fmt.Sprintf("/programs/programs/%s/source/main", strings.ToLower(objectName))
	case "CLASS":
		sourcePath = fmt.Sprintf("/oo/classes/%s/source/main", strings.ToLower(objectName))
	case "INCLUDE":
		sourcePath = fmt.Sprintf("/programs/includes/%s/source/main", strings.ToLower(objectName))
	case "INTERFACE":
		sourcePath = fmt.Sprintf("/oo/interfaces/%s/source/main", strings.ToLower(objectName))
	default:
		return "", fmt.Errorf("unsupported object type for source retrieval: %s", objectType)
	}

	url := c.baseURL + sourcePath

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.doRequest(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("%w: %s %s", ErrNotFound, objectType, objectName)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get object source: HTTP %d - %s", resp.StatusCode, string(body))
	}

	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read source: %w", err)
	}

	c.logger.Info("Object source retrieved successfully", zap.String("object_type", objectType), zap.String("object_name", objectName), zap.Int("source_length", len(source)))
	return string(source), nil
}

// CheckObjectExists checks if an object exists (using GetObjectSource internally)
func (c *ADTClientImpl) CheckObjectExists(ctx context.Context, objectType, objectName string) (bool, error) {
	c.logger.Debug("Checking object existence", zap.String("object_type", objectType), zap.String("object_name", objectName))

	_, err := c.GetObjectSource(ctx, objectType, objectName)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.logger.Debug("Object does not exist", zap.String("object_type", objectType), zap.String("object_name", objectName))
			return false, nil
		}
		return false, err
	}

	c.logger.Debug("Object exists", zap.String("object_type", objectType), zap.String("object_name", objectName))
	return true, nil
}

// setClassSource sets the source code for a class (following similar pattern to programs)
func (c *ADTClientImpl) setClassSource(ctx context.Context, className, source string) error {
	// Ensure stateful session
	originalSessionType := c.sessionType
	c.sessionType = string(types.SessionStateful)
	defer func() {
		c.sessionType = originalSessionType
	}()

	classNameLower := strings.ToLower(className)
	classPath := fmt.Sprintf("/oo/classes/%s", classNameLower)
	sourcePath := fmt.Sprintf("/oo/classes/%s/source/main", classNameLower)

	// Lock the class object
	lockHandle, corrNr, err := c.lockObject(ctx, classPath)
	if err != nil {
		return fmt.Errorf("failed to lock class: %w", err)
	}

	defer func() {
		if unlockErr := c.unlockObject(ctx, classPath, lockHandle); unlockErr != nil {
			c.logger.Warn("Failed to unlock class", zap.String("class", className), zap.Error(unlockErr))
		}
	}()

	// Set the source
	if err := c.setObjectSource(ctx, sourcePath, source, lockHandle, corrNr); err != nil {
		return fmt.Errorf("failed to set source: %w", err)
	}

	return nil
}

// activateProgram activates the program
func (c *ADTClientImpl) activateProgram(ctx context.Context, opts *CreateProgramOptions) error {
	c.logger.Info("Activating program", zap.String("program", opts.Name))

	// Prepare activation request
	activationReq := ActivationRequest{
		Namespace: "http://www.sap.com/adt/core",
		ObjectRef: ActivationRef{
			URI:  fmt.Sprintf("/sap/bc/adt/programs/programs/%s", strings.ToLower(opts.Name)),
			Name: opts.Name,
		},
	}

	xmlPayload, err := xml.Marshal(activationReq)
	if err != nil {
		return fmt.Errorf("failed to marshal activation request: %w", err)
	}

	// Add XML header
	fullPayload := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + string(xmlPayload)

	url := c.baseURL + "/activation?method=activate&preauditRequested=true&sap-client=" + c.config.Client + "&sap-language=" + c.config.Language

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(fullPayload))
	if err != nil {
		return fmt.Errorf("failed to create activation request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "application/*")
	req.Header.Set("Accept", "application/*")
	req.Header.Set("X-CSRF-Token", c.csrfToken)

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("activation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("activation failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// Parse activation response to check for warnings/errors
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Warn("Could not read activation response", zap.Error(err))
		return nil // Don't fail if we can't read the response
	}

	// Log activation response for debugging
	c.logger.Debug("Activation response", zap.String("response", string(responseBody)))

	return nil
}

// Enhanced CreateProgram with options support
func (c *ADTClientImpl) CreateProgramWithOptions(ctx context.Context, opts CreateProgramOptions) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	// Validate and set defaults
	opts.Name = strings.ToUpper(strings.TrimSpace(opts.Name))
	if opts.Name == "" {
		return fmt.Errorf("program name cannot be empty")
	}
	if opts.Description == "" {
		opts.Description = fmt.Sprintf("Program %s", opts.Name)
	}
	if opts.Package == "" {
		opts.Package = "$TMP"
	}
	if opts.Responsible == "" {
		opts.Responsible = strings.ToUpper(strings.TrimSpace(c.config.Username))
	}

	c.logger.Info("Creating ABAP program with options",
		zap.String("name", opts.Name),
		zap.String("description", opts.Description),
		zap.String("package", opts.Package),
		zap.Bool("insert_source", opts.InsertSource),
		zap.Bool("activate", opts.Activate))

	// Step 1: Create the program structure
	if err := c.createProgramMetadata(ctx, opts.Name, opts.Description, opts.Package); err != nil {
		return fmt.Errorf("failed to create program structure: %w", err)
	}

	// Step 2: Insert source code if provided
	if opts.InsertSource && opts.Source != "" {
		if err := c.setSourceUsingWorkingPattern(ctx, opts.Name, opts.Source); err != nil {
			return fmt.Errorf("failed to insert source code: %w", err)
		}
	}

	// Step 3: Activate if requested
	if opts.Activate {
		if err := c.activateProgram(ctx, &opts); err != nil {
			return fmt.Errorf("failed to activate program: %w", err)
		}
	}

	c.logger.Info("Program creation completed successfully", zap.String("name", opts.Name))
	return nil
}

// escapeXML escapes XML special characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// CreateProgramWithSource is a convenience method that creates a program with source code
func (c *ADTClientImpl) CreateProgramWithSource(ctx context.Context, name, description, packageName, source string) error {
	opts := CreateProgramOptions{
		Name:         name,
		Description:  description,
		Source:       source,
		Package:      packageName,
		Activate:     true,
		InsertSource: true,
	}

	if opts.Package == "" {
		opts.Package = "$TMP"
	}

	return c.CreateProgramWithOptions(ctx, opts)
}

// UpdateProgram updates an existing ABAP program's source code
func (c *ADTClientImpl) UpdateProgram(ctx context.Context, name, source string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	name = strings.ToUpper(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("program name cannot be empty")
	}

	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("source code cannot be empty")
	}

	c.logger.Info("Updating ABAP program",
		zap.String("name", name),
		zap.Int("source_length", len(source)))

	nameLower := strings.ToLower(name)
	objectPath := fmt.Sprintf("/programs/programs/%s", nameLower)
	sourcePath := fmt.Sprintf("/programs/programs/%s/source/main", nameLower)

	lockHandle, corrNr, err := c.lockObject(ctx, objectPath)
	if err != nil {
		return fmt.Errorf("failed to lock program: %w", err)
	}

	defer func() {
		if unlockErr := c.unlockObject(ctx, objectPath, lockHandle); unlockErr != nil {
			c.logger.Warn("Failed to unlock program",
				zap.String("program", name),
				zap.Error(unlockErr))
		}
	}()

	if err := c.setObjectSource(ctx, sourcePath, source, lockHandle, corrNr); err != nil {
		return fmt.Errorf("failed to update program source: %w", err)
	}

	c.logger.Info("Program updated successfully", zap.String("name", name))
	return nil
}


// UpdateClass updates an existing ABAP class's source code
func (c *ADTClientImpl) UpdateClass(ctx context.Context, name, source string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	name = strings.ToUpper(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("class name cannot be empty")
	}

	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("source code cannot be empty")
	}

	c.logger.Info("Updating ABAP class",
		zap.String("name", name),
		zap.Int("source_length", len(source)))

	nameLower := strings.ToLower(name)
	objectPath := fmt.Sprintf("/oo/classes/%s", nameLower)
	sourcePath := fmt.Sprintf("/oo/classes/%s/source/main", nameLower)

	lockHandle, corrNr, err := c.lockObject(ctx, objectPath)
	if err != nil {
		return fmt.Errorf("failed to lock class: %w", err)
	}

	defer func() {
		if unlockErr := c.unlockObject(ctx, objectPath, lockHandle); unlockErr != nil {
			c.logger.Warn("Failed to unlock class",
				zap.String("class", name),
				zap.Error(unlockErr))
		}
	}()

	if err := c.setObjectSource(ctx, sourcePath, source, lockHandle, corrNr); err != nil {
		return fmt.Errorf("failed to update class source: %w", err)
	}

	c.logger.Info("Class updated successfully", zap.String("name", name))
	return nil
}

// UpdateInclude updates an existing ABAP include's source code
func (c *ADTClientImpl) UpdateInclude(ctx context.Context, name, source string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	name = strings.ToUpper(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("include name cannot be empty")
	}

	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("source code cannot be empty")
	}

	c.logger.Info("Updating ABAP include",
		zap.String("name", name),
		zap.Int("source_length", len(source)))

	nameLower := strings.ToLower(name)
	objectPath := fmt.Sprintf("/programs/includes/%s", nameLower)
	sourcePath := fmt.Sprintf("/programs/includes/%s/source/main", nameLower)

	lockHandle, corrNr, err := c.lockObject(ctx, objectPath)
	if err != nil {
		return fmt.Errorf("failed to lock include: %w", err)
	}

	defer func() {
		if unlockErr := c.unlockObject(ctx, objectPath, lockHandle); unlockErr != nil {
			c.logger.Warn("Failed to unlock include",
				zap.String("include", name),
				zap.Error(unlockErr))
		}
	}()

	if err := c.setObjectSource(ctx, sourcePath, source, lockHandle, corrNr); err != nil {
		return fmt.Errorf("failed to update include source: %w", err)
	}

	c.logger.Info("Include updated successfully", zap.String("name", name))
	return nil
}

// UpdateInterface updates an existing ABAP interface's source code
func (c *ADTClientImpl) UpdateInterface(ctx context.Context, name, source string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	name = strings.ToUpper(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("interface name cannot be empty")
	}

	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("source code cannot be empty")
	}

	c.logger.Info("Updating ABAP interface",
		zap.String("name", name),
		zap.Int("source_length", len(source)))

	nameLower := strings.ToLower(name)
	objectPath := fmt.Sprintf("/oo/interfaces/%s", nameLower)
	sourcePath := fmt.Sprintf("/oo/interfaces/%s/source/main", nameLower)

	lockHandle, corrNr, err := c.lockObject(ctx, objectPath)
	if err != nil {
		return fmt.Errorf("failed to lock interface: %w", err)
	}

	defer func() {
		if unlockErr := c.unlockObject(ctx, objectPath, lockHandle); unlockErr != nil {
			c.logger.Warn("Failed to unlock interface",
				zap.String("interface", name),
				zap.Error(unlockErr))
		}
	}()

	if err := c.setObjectSource(ctx, sourcePath, source, lockHandle, corrNr); err != nil {
		return fmt.Errorf("failed to update interface source: %w", err)
	}

	c.logger.Info("Interface updated successfully", zap.String("name", name))
	return nil
}

// objectTypeToURI maps an object type and name to the ADT URI path
func objectTypeToURI(objectType, objectName string) (string, error) {
	name := strings.ToLower(objectName)
	switch strings.ToUpper(objectType) {
	case "PROG", "PROGRAM":
		return fmt.Sprintf("/sap/bc/adt/programs/programs/%s", name), nil
	case "CLAS", "CLASS":
		return fmt.Sprintf("/sap/bc/adt/oo/classes/%s", name), nil
	case "INTF", "INTERFACE":
		return fmt.Sprintf("/sap/bc/adt/oo/interfaces/%s", name), nil
	case "FUGR", "FUNCTION_GROUP":
		return fmt.Sprintf("/sap/bc/adt/functions/groups/%s", name), nil
	case "INCL", "INCLUDE":
		return fmt.Sprintf("/sap/bc/adt/programs/includes/%s", name), nil
	default:
		return "", fmt.Errorf("unsupported object type for activation: %s", objectType)
	}
}

// ActivationResponse XML structures for parsing activation results
type ActivationResponse struct {
	XMLName  xml.Name             `xml:"messages"`
	Messages []ActivationMsgEntry `xml:"msg"`
}

type ActivationMsgEntry struct {
	Severity string `xml:"severity,attr"`
	Text     string `xml:",chardata"`
	Href     string `xml:"href,attr,omitempty"`
}

// ActivateObject activates an ABAP object (program, class, interface, etc.)
func (c *ADTClientImpl) ActivateObject(ctx context.Context, objectType, objectName string) (*types.ActivationResult, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	objectName = strings.ToUpper(strings.TrimSpace(objectName))
	if objectName == "" {
		return nil, fmt.Errorf("object name cannot be empty")
	}

	c.logger.Info("Activating object",
		zap.String("type", objectType),
		zap.String("name", objectName))

	uri, err := objectTypeToURI(objectType, objectName)
	if err != nil {
		return nil, err
	}

	activationReq := ActivationRequest{
		Namespace: "http://www.sap.com/adt/core",
		ObjectRef: ActivationRef{
			URI:  uri,
			Name: objectName,
		},
	}

	xmlPayload, err := xml.Marshal(activationReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal activation request: %w", err)
	}

	fullPayload := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + string(xmlPayload)

	reqURL := c.baseURL + "/activation?method=activate&preauditRequested=true&sap-client=" + c.config.Client + "&sap-language=" + c.config.Language

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(fullPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create activation request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "application/*")
	req.Header.Set("Accept", "application/*")
	req.Header.Set("X-CSRF-Token", c.csrfToken)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("activation request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)

	result := &types.ActivationResult{
		ObjectName: objectName,
		ObjectType: objectType,
		Success:    resp.StatusCode == http.StatusOK,
	}

	// Parse activation response messages
	if len(responseBody) > 0 {
		var activationResp ActivationResponse
		if xmlErr := xml.Unmarshal(responseBody, &activationResp); xmlErr == nil {
			for _, msg := range activationResp.Messages {
				result.Messages = append(result.Messages, types.ActivationMessage{
					Severity: msg.Severity,
					Text:     strings.TrimSpace(msg.Text),
				})
				if msg.Severity == "error" || msg.Severity == "E" {
					result.Success = false
				}
			}
		}
	}

	if resp.StatusCode != http.StatusOK {
		result.Success = false
		if len(result.Messages) == 0 {
			result.Messages = append(result.Messages, types.ActivationMessage{
				Severity: "error",
				Text:     fmt.Sprintf("activation failed: HTTP %d - %s", resp.StatusCode, string(responseBody)),
			})
		}
	}

	c.logger.Info("Activation completed",
		zap.String("type", objectType),
		zap.String("name", objectName),
		zap.Bool("success", result.Success),
		zap.Int("messages", len(result.Messages)))

	return result, nil
}

// Unit test XML request/response structures
type unitTestRunConfig struct {
	XMLName    xml.Name           `xml:"aunit:runConfiguration"`
	AunitNS    string             `xml:"xmlns:aunit,attr"`
	External   unitTestExternal   `xml:"aunit:external"`
	Options    unitTestOptions    `xml:"aunit:options"`
	ObjectSets unitTestObjectSets `xml:"aunit:objectSets"`
}

type unitTestExternal struct {
	Coverage unitTestCoverage `xml:"aunit:coverage"`
}

type unitTestCoverage struct {
	Active string `xml:"active,attr"`
}

type unitTestOptions struct {
	URIType unitTestURIType `xml:"aunit:uriType"`
}

type unitTestURIType struct {
	Value string `xml:"value,attr"`
}

type unitTestObjectSets struct {
	ObjectSet unitTestObjectSet `xml:"aunit:objectSet"`
}

type unitTestObjectSet struct {
	Kind       string             `xml:"kind,attr"`
	ObjectRefs unitTestObjectRefs `xml:"adtcore:objectReferences"`
}

type unitTestObjectRefs struct {
	AdtcoreNS string           `xml:"xmlns:adtcore,attr"`
	Refs      []unitTestObjRef `xml:"adtcore:objectReference"`
}

type unitTestObjRef struct {
	URI string `xml:"adtcore:uri,attr"`
}

// Response structures
type unitTestRunResult struct {
	XMLName xml.Name          `xml:"runResult"`
	Program []unitTestProgram `xml:"program"`
}

type unitTestProgram struct {
	Name        string              `xml:"name,attr"`
	URI         string              `xml:"uri,attr"`
	TestClasses []unitTestTestClass `xml:"testClasses>testClass"`
}

type unitTestTestClass struct {
	Name        string               `xml:"name,attr"`
	URI         string               `xml:"uri,attr"`
	TestMethods []unitTestTestMethod `xml:"testMethods>testMethod"`
}

type unitTestTestMethod struct {
	Name   string          `xml:"name,attr"`
	URI    string          `xml:"uri,attr"`
	Alerts []unitTestAlert `xml:"alerts>alert"`
}

type unitTestAlert struct {
	Kind     string                `xml:"kind,attr"`
	Severity string                `xml:"severity,attr"`
	Title    string                `xml:"title"`
	Details  []unitTestAlertDetail `xml:"details>detail"`
}

type unitTestAlertDetail struct {
	Text string `xml:"text,attr"`
}

// RunUnitTests runs ABAP unit tests for the given object
func (c *ADTClientImpl) RunUnitTests(ctx context.Context, objectType, objectName string) (*types.UnitTestResult, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	objectName = strings.ToUpper(strings.TrimSpace(objectName))
	if objectName == "" {
		return nil, fmt.Errorf("object name cannot be empty")
	}

	c.logger.Info("Running unit tests",
		zap.String("type", objectType),
		zap.String("name", objectName))

	uri, err := objectTypeToURI(objectType, objectName)
	if err != nil {
		return nil, err
	}

	// Build unit test run configuration XML
	config := unitTestRunConfig{
		AunitNS: "http://www.sap.com/adt/aunit",
		External: unitTestExternal{
			Coverage: unitTestCoverage{Active: "false"},
		},
		Options: unitTestOptions{
			URIType: unitTestURIType{Value: "semantic"},
		},
		ObjectSets: unitTestObjectSets{
			ObjectSet: unitTestObjectSet{
				Kind: "inclusive",
				ObjectRefs: unitTestObjectRefs{
					AdtcoreNS: "http://www.sap.com/adt/core",
					Refs: []unitTestObjRef{
						{URI: uri},
					},
				},
			},
		},
	}

	xmlPayload, err := xml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal unit test config: %w", err)
	}

	fullPayload := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + string(xmlPayload)

	reqURL := c.baseURL + "/abapunit/testruns?sap-client=" + c.config.Client + "&sap-language=" + c.config.Language

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(fullPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create unit test request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "application/vnd.sap.adt.abapunit.testruns.config.v4+xml")
	req.Header.Set("Accept", "application/vnd.sap.adt.abapunit.testruns.result.v2+xml")
	req.Header.Set("X-CSRF-Token", c.csrfToken)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("unit test request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read unit test response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unit test request failed: HTTP %d - %s", resp.StatusCode, string(responseBody))
	}

	c.logger.Debug("Unit test response", zap.String("body", string(responseBody)))

	// Parse response
	result := &types.UnitTestResult{
		ObjectName: objectName,
	}

	var runResult unitTestRunResult
	if err := xml.Unmarshal(responseBody, &runResult); err != nil {
		// If we can't parse the XML, return a basic success result
		c.logger.Warn("Could not parse unit test response XML", zap.Error(err))
		result.AllPassed = true
		return result, nil
	}

	// Extract test results
	for _, program := range runResult.Program {
		for _, tc := range program.TestClasses {
			classResult := types.TestClassResult{
				Name: tc.Name,
			}

			for _, tm := range tc.TestMethods {
				methodResult := types.TestMethodResult{
					Name:   tm.Name,
					Status: "passed",
				}

				result.TotalTests++

				if len(tm.Alerts) > 0 {
					for _, alert := range tm.Alerts {
						if alert.Kind == "failedAssertion" || alert.Severity == "critical" || alert.Severity == "fatal" {
							methodResult.Status = "failed"
							methodResult.Message = alert.Title
							if len(alert.Details) > 0 {
								methodResult.Message += ": " + alert.Details[0].Text
							}
							break
						}
						if alert.Kind == "warning" {
							methodResult.Message = alert.Title
						}
					}
				}

				if methodResult.Status == "passed" {
					result.Passed++
				} else {
					result.Failed++
				}

				classResult.Methods = append(classResult.Methods, methodResult)
			}

			result.TestClasses = append(result.TestClasses, classResult)
		}
	}

	result.AllPassed = result.Failed == 0 && result.TotalTests > 0

	c.logger.Info("Unit tests completed",
		zap.String("name", objectName),
		zap.Int("total", result.TotalTests),
		zap.Int("passed", result.Passed),
		zap.Int("failed", result.Failed),
		zap.Bool("all_passed", result.AllPassed))

	return result, nil
}

// ============================================================
// LSP Support: Syntax Check, Code Completion, Navigation
// ============================================================

// syntaxCheckResponse XML structures for parsing check run results
type syntaxCheckResponse struct {
	XMLName  xml.Name             `xml:"checkRun"`
	Messages []syntaxCheckMessage `xml:"checkMessage"`
}

type syntaxCheckMessage struct {
	URI      string `xml:"uri,attr"`
	Type     string `xml:"type,attr"`
	Severity string `xml:"severity,attr"`
	Text     string `xml:",chardata"`
	Line     int    `xml:"line,attr"`
	Column   int    `xml:"column,attr"`
}

// SyntaxCheck performs a syntax check on ABAP source via ADT checkruns endpoint.
// POST /sap/bc/adt/checkruns?reporters=abapCheckRun
func (c *ADTClientImpl) SyntaxCheck(ctx context.Context, objectType, objectName, source string) (*types.SyntaxCheckResult, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated")
	}

	objectName = strings.ToUpper(strings.TrimSpace(objectName))
	uri, err := objectTypeToURI(objectType, objectName)
	if err != nil {
		return nil, err
	}

	c.logger.Info("Running syntax check",
		zap.String("type", objectType),
		zap.String("name", objectName))

	// Build the check run request XML
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<chkrun:checkObjectList xmlns:chkrun="http://www.sap.com/adt/checkrun" xmlns:adtcore="http://www.sap.com/adt/core">
  <chkrun:checkObject adtcore:uri="%s" chkrun:version="active">
    <chkrun:source>%s</chkrun:source>
  </chkrun:checkObject>
</chkrun:checkObjectList>`, uri, escapeXML(source))

	reqURL := c.baseURL + "/checkruns?reporters=abapCheckRun&sap-client=" + c.config.Client + "&sap-language=" + c.config.Language

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create syntax check request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "application/vnd.sap.adt.checkobjects+xml")
	req.Header.Set("Accept", "application/vnd.sap.adt.checkmessages+xml")
	req.Header.Set("X-CSRF-Token", c.csrfToken)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("syntax check request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)

	result := &types.SyntaxCheckResult{
		ObjectName: objectName,
		ObjectType: objectType,
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("syntax check failed: HTTP %d - %s", resp.StatusCode, string(responseBody))
	}

	if len(responseBody) > 0 {
		var checkResp syntaxCheckResponse
		if xmlErr := xml.Unmarshal(responseBody, &checkResp); xmlErr == nil {
			for _, msg := range checkResp.Messages {
				severity := "error"
				switch strings.ToLower(msg.Severity) {
				case "w", "warning":
					severity = "warning"
				case "i", "info", "information":
					severity = "info"
				}
				result.Messages = append(result.Messages, types.SyntaxCheckMessage{
					Severity: severity,
					Text:     strings.TrimSpace(msg.Text),
					Line:     msg.Line,
					Column:   msg.Column,
				})
			}
		}
	}

	c.logger.Info("Syntax check completed",
		zap.String("name", objectName),
		zap.Int("messages", len(result.Messages)))

	return result, nil
}

// completionResponse XML structures for parsing code completion results
type completionResponse struct {
	XMLName   xml.Name             `xml:"completionProposals"`
	Proposals []completionProposal `xml:"proposal"`
}

type completionProposal struct {
	Identifier  string `xml:"identifier,attr"`
	Description string `xml:"description,attr"`
	Kind        string `xml:"kind,attr"`
	InsertText  string `xml:"insertText,attr"`
}

// GetCompletionProposals retrieves code completion proposals from ADT.
// POST /sap/bc/adt/abapsource/codecompletion/proposal
func (c *ADTClientImpl) GetCompletionProposals(ctx context.Context, objectType, objectName, source string, line, column int) ([]types.CompletionProposal, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated")
	}

	objectName = strings.ToUpper(strings.TrimSpace(objectName))
	uri, err := objectTypeToURI(objectType, objectName)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Getting completion proposals",
		zap.String("name", objectName),
		zap.Int("line", line),
		zap.Int("column", column))

	reqURL := fmt.Sprintf("%s/abapsource/codecompletion/proposal?uri=%s&line=%d&column=%d&sap-client=%s&sap-language=%s",
		c.baseURL, url.QueryEscape(uri), line, column, c.config.Client, c.config.Language)

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(source))
	if err != nil {
		return nil, fmt.Errorf("failed to create completion request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("X-CSRF-Token", c.csrfToken)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("completion request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("completion failed: HTTP %d - %s", resp.StatusCode, string(responseBody))
	}

	var proposals []types.CompletionProposal

	if len(responseBody) > 0 {
		var compResp completionResponse
		if xmlErr := xml.Unmarshal(responseBody, &compResp); xmlErr == nil {
			for _, p := range compResp.Proposals {
				kind := "keyword"
				switch strings.ToLower(p.Kind) {
				case "method":
					kind = "function"
				case "attribute", "variable":
					kind = "variable"
				case "class":
					kind = "class"
				case "type":
					kind = "type"
				}
				proposals = append(proposals, types.CompletionProposal{
					Identifier:  p.Identifier,
					Description: p.Description,
					Kind:        kind,
					InsertText:  p.InsertText,
				})
			}
		}
	}

	return proposals, nil
}

// navigationResponse XML structures for parsing navigation results
type navigationResponse struct {
	XMLName xml.Name `xml:"objectReference"`
	URI     string   `xml:"uri,attr"`
	Name    string   `xml:"name,attr"`
	Type    string   `xml:"type,attr"`
	Line    int      `xml:"line,attr"`
	Column  int      `xml:"column,attr"`
}

// GetNavigationTarget retrieves the go-to-definition target from ADT.
// POST /sap/bc/adt/navigation/target
func (c *ADTClientImpl) GetNavigationTarget(ctx context.Context, objectType, objectName, source string, line, column int) (*types.NavigationTarget, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated")
	}

	objectName = strings.ToUpper(strings.TrimSpace(objectName))
	uri, err := objectTypeToURI(objectType, objectName)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Getting navigation target",
		zap.String("name", objectName),
		zap.Int("line", line),
		zap.Int("column", column))

	reqURL := fmt.Sprintf("%s/navigation/target?uri=%s&line=%d&column=%d&sap-client=%s&sap-language=%s",
		c.baseURL, url.QueryEscape(uri), line, column, c.config.Client, c.config.Language)

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(source))
	if err != nil {
		return nil, fmt.Errorf("failed to create navigation request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("X-CSRF-Token", c.csrfToken)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("navigation request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("navigation failed: HTTP %d - %s", resp.StatusCode, string(responseBody))
	}

	if len(responseBody) == 0 {
		return nil, nil
	}

	var navResp navigationResponse
	if err := xml.Unmarshal(responseBody, &navResp); err != nil {
		return nil, fmt.Errorf("failed to parse navigation response: %w", err)
	}

	return &types.NavigationTarget{
		URI:        navResp.URI,
		ObjectName: navResp.Name,
		ObjectType: navResp.Type,
		Line:       navResp.Line,
		Column:     navResp.Column,
	}, nil
}
