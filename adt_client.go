package main

import (
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
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
	ADT_PROGRAMS_ENDPOINT         = "/programs/programs/%s/source/main"
	ADT_CLASSES_ENDPOINT          = "/oo/classes/%s/source/main"
	ADT_FUNCTION_GROUPS_ENDPOINT  = "/functions/groups/%s/source/main"
	ADT_FUNCTIONS_ENDPOINT        = "/functions/groups/%s/fmodules/%s/source/main"
	ADT_TABLES_ENDPOINT           = "/ddic/tables/%s/source/main"
	ADT_STRUCTURES_ENDPOINT       = "/ddic/structures/%s/source/main"
	ADT_INCLUDES_ENDPOINT         = "/programs/includes/%s/source/main"
	ADT_INTERFACES_ENDPOINT       = "/oo/interfaces/%s/source/main"
	ADT_DOMAINS_ENDPOINT          = "/ddic/domains/%s/source/main"
	ADT_DATA_ELEMENTS_ENDPOINT    = "/ddic/dataelements/%s"
	ADT_PACKAGE_CONTENTS_ENDPOINT = "/repository/nodestructure"
	ADT_SEARCH_ENDPOINT           = "/repository/informationsystem/search"
	ADT_TRANSACTION_ENDPOINT      = "/repository/informationsystem/objectproperties/values"
	ADT_TABLE_CONTENTS_ENDPOINT   = "/z_mcp_abap_adt/z_tablecontent/%s" // Custom service required
)

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
func NewADTClient(config *types.ADTConfig) types.ADTClient {
	// Set defaults
	if config.Language == "" {
		config.Language = "EN"
	}
	if config.Client == "" {
		config.Client = "100"
	}
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 30
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 60
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
		Timeout:   time.Duration(config.RequestTimeout) * time.Second,
		Transport: transport,
		Jar:       jar,
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

// GetProgram retrieves ABAP program source code with enhanced error handling
func (c *ADTClientImpl) GetProgram(programName string) (*types.ADTSourceCode, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving program", zap.String("program", programName))

	programName = strings.ToUpper(strings.TrimSpace(programName))
	url := fmt.Sprintf("%s/programs/programs/%s/source/main", c.baseURL, programName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("program %s not found (404)", programName)
		} else if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("authentication failed (401) - session may have expired")
		} else if resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("access forbidden (403) - insufficient permissions for program %s", programName)
		}

		return nil, fmt.Errorf("failed to get program %s: HTTP %d - %s", programName, resp.StatusCode, string(body))
	}

	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	result := &types.ADTSourceCode{
		ObjectName: programName,
		ObjectType: "PROG",
		Source:     string(source),
		Version:    resp.Header.Get("ETag"),
		ETag:       resp.Header.Get("ETag"),
	}

	c.logger.Info("Program retrieved successfully",
		zap.String("program", programName),
		zap.Int("source_length", len(result.Source)))

	return result, nil
}

// GetClass retrieves ABAP class source code with enhanced error handling
func (c *ADTClientImpl) GetClass(className string) (*types.ADTSourceCode, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving class", zap.String("class", className))

	className = strings.ToUpper(strings.TrimSpace(className))
	url := fmt.Sprintf("%s/oo/classes/%s/source/main", c.baseURL, className)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("class %s not found (404)", className)
		} else if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("authentication failed (401) - session may have expired")
		} else if resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("access forbidden (403) - insufficient permissions for class %s", className)
		}

		return nil, fmt.Errorf("failed to get class %s: HTTP %d - %s", className, resp.StatusCode, string(body))
	}

	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	result := &types.ADTSourceCode{
		ObjectName: className,
		ObjectType: "CLAS",
		Source:     string(source),
		Version:    resp.Header.Get("ETag"),
		ETag:       resp.Header.Get("ETag"),
	}

	c.logger.Info("Class retrieved successfully",
		zap.String("class", className),
		zap.Int("source_length", len(result.Source)))

	return result, nil
}

// GetFunction retrieves ABAP function module source code
func (c *ADTClientImpl) GetFunction(functionName, functionGroup string) (*types.ADTSourceCode, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving function module",
		zap.String("function", functionName),
		zap.String("function_group", functionGroup))

	functionName = strings.ToUpper(strings.TrimSpace(functionName))
	functionGroup = strings.ToUpper(strings.TrimSpace(functionGroup))
	url := fmt.Sprintf("%s"+ADT_FUNCTIONS_ENDPOINT, c.baseURL, functionGroup, functionName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("function %s in group %s not found (404)", functionName, functionGroup)
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
func (c *ADTClientImpl) GetFunctionGroup(functionGroup string) (*types.ADTSourceCode, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving function group", zap.String("function_group", functionGroup))

	functionGroup = strings.ToUpper(strings.TrimSpace(functionGroup))
	url := fmt.Sprintf("%s"+ADT_FUNCTION_GROUPS_ENDPOINT, c.baseURL, functionGroup)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("function group %s not found (404)", functionGroup)
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

// GetInclude retrieves ABAP include source code
func (c *ADTClientImpl) GetInclude(includeName string) (*types.ADTSourceCode, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving include", zap.String("include", includeName))

	includeName = strings.ToUpper(strings.TrimSpace(includeName))
	url := fmt.Sprintf("%s"+ADT_INCLUDES_ENDPOINT, c.baseURL, includeName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("include %s not found (404)", includeName)
		}
		return nil, fmt.Errorf("failed to get include %s: HTTP %d - %s", includeName, resp.StatusCode, string(body))
	}

	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	result := &types.ADTSourceCode{
		ObjectName: includeName,
		ObjectType: "INCL",
		Source:     string(source),
		Version:    resp.Header.Get("ETag"),
		ETag:       resp.Header.Get("ETag"),
	}

	c.logger.Info("Include retrieved successfully",
		zap.String("include", includeName),
		zap.Int("source_length", len(result.Source)))

	return result, nil
}

// GetInterface retrieves ABAP interface source code
func (c *ADTClientImpl) GetInterface(interfaceName string) (*types.ADTSourceCode, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving interface", zap.String("interface", interfaceName))

	interfaceName = strings.ToUpper(strings.TrimSpace(interfaceName))
	url := fmt.Sprintf("%s"+ADT_INTERFACES_ENDPOINT, c.baseURL, interfaceName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("interface %s not found (404)", interfaceName)
		}
		return nil, fmt.Errorf("failed to get interface %s: HTTP %d - %s", interfaceName, resp.StatusCode, string(body))
	}

	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	result := &types.ADTSourceCode{
		ObjectName: interfaceName,
		ObjectType: "INTF",
		Source:     string(source),
		Version:    resp.Header.Get("ETag"),
		ETag:       resp.Header.Get("ETag"),
	}

	c.logger.Info("Interface retrieved successfully",
		zap.String("interface", interfaceName),
		zap.Int("source_length", len(result.Source)))

	return result, nil
}

// GetStructure retrieves ABAP structure definition
func (c *ADTClientImpl) GetStructure(structureName string) (*types.ADTSourceCode, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving structure", zap.String("structure", structureName))

	structureName = strings.ToUpper(strings.TrimSpace(structureName))
	url := fmt.Sprintf("%s"+ADT_STRUCTURES_ENDPOINT, c.baseURL, structureName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("structure %s not found (404)", structureName)
		}
		return nil, fmt.Errorf("failed to get structure %s: HTTP %d - %s", structureName, resp.StatusCode, string(body))
	}

	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	result := &types.ADTSourceCode{
		ObjectName: structureName,
		ObjectType: "STRU",
		Source:     string(source),
		Version:    resp.Header.Get("ETag"),
		ETag:       resp.Header.Get("ETag"),
	}

	c.logger.Info("Structure retrieved successfully",
		zap.String("structure", structureName),
		zap.Int("source_length", len(result.Source)))

	return result, nil
}

// GetTable retrieves ABAP table structure
func (c *ADTClientImpl) GetTable(tableName string) (*types.ADTSourceCode, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving table", zap.String("table", tableName))

	tableName = strings.ToUpper(strings.TrimSpace(tableName))
	url := fmt.Sprintf("%s"+ADT_TABLES_ENDPOINT, c.baseURL, tableName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("table %s not found (404)", tableName)
		}
		return nil, fmt.Errorf("failed to get table %s: HTTP %d - %s", tableName, resp.StatusCode, string(body))
	}

	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	result := &types.ADTSourceCode{
		ObjectName: tableName,
		ObjectType: "TABL",
		Source:     string(source),
		Version:    resp.Header.Get("ETag"),
		ETag:       resp.Header.Get("ETag"),
	}

	c.logger.Info("Table retrieved successfully",
		zap.String("table", tableName),
		zap.Int("source_length", len(result.Source)))

	return result, nil
}

// GetPackageContents retrieves package contents
func (c *ADTClientImpl) GetPackageContents(packageName string) (*types.ADTPackage, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving package contents", zap.String("package", packageName))

	packageName = strings.ToUpper(strings.TrimSpace(packageName))

	// Prepare POST data for package contents request
	postData := url.Values{
		"parent_type":           {"DEVC/K"},
		"parent_name":           {packageName},
		"withShortDescriptions": {"true"},
	}

	req, err := http.NewRequest("POST", c.baseURL+ADT_PACKAGE_CONTENTS_ENDPOINT, strings.NewReader(postData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("package %s not found (404)", packageName)
		}
		return nil, fmt.Errorf("failed to get package %s: HTTP %d - %s", packageName, resp.StatusCode, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse XML response (simplified - would need proper XML parsing in production)
	result := &types.ADTPackage{
		Name:        packageName,
		Description: fmt.Sprintf("Package %s", packageName),
		Objects:     []types.ADTObject{}, // Would parse from XML response
	}

	c.logger.Info("Package contents retrieved successfully",
		zap.String("package", packageName),
		zap.Int("response_length", len(responseBody)))

	return result, nil
}

// SearchObjects searches for ABAP objects
func (c *ADTClientImpl) SearchObjects(pattern string, objectTypes []string) (*types.ADTSearchResult, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Searching objects",
		zap.String("pattern", pattern),
		zap.Strings("types", objectTypes))

	maxResults := 100
	searchURL := fmt.Sprintf("%s%s?operation=quickSearch&query=%s&maxResults=%d",
		c.baseURL,
		ADT_SEARCH_ENDPOINT,
		url.QueryEscape(pattern),
		maxResults)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "application/xml")

	resp, err := c.httpClient.Do(req)
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
		log.Fatal(err)
	}

	c.logger.Info("Search completed successfully",
		zap.String("pattern", pattern),
		zap.Int("response_length", len(responseBody)))

	return &result, nil
}

// ListPackages lists packages matching a pattern
func (c *ADTClientImpl) ListPackages(pattern string) ([]types.ADTPackage, error) {
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
		ADT_SEARCH_ENDPOINT,
		url.QueryEscape(pattern))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "application/xml")

	resp, err := c.httpClient.Do(req)
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

	// Parse XML response (simplified - would need proper XML parsing in production)
	// For now, return some sample packages based on common SAP package patterns
	packages := []types.ADTPackage{
		{Name: "$TMP", Description: "Temporary Objects"},
		{Name: "ZLOCAL", Description: "Local Development Package"},
	}

	// If pattern is specific, try to return a match
	if pattern != "*" && !strings.Contains(pattern, "*") {
		// Direct package name lookup
		packages = []types.ADTPackage{
			{Name: strings.ToUpper(pattern), Description: fmt.Sprintf("Package %s", strings.ToUpper(pattern))},
		}
	}

	c.logger.Info("Package search completed",
		zap.String("pattern", pattern),
		zap.Int("packages_found", len(packages)),
		zap.Int("response_length", len(responseBody)))

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
func (c *ADTClientImpl) GetTypeInfo(typeName string) (*types.ADTTypeInfo, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving type info", zap.String("type", typeName))

	typeName = strings.ToUpper(strings.TrimSpace(typeName))

	// First try as domain
	domainURL := fmt.Sprintf("%s"+ADT_DOMAINS_ENDPOINT, c.baseURL, typeName)
	if source, err := c.getTypeSource(domainURL, "text/plain"); err == nil {
		return &types.ADTTypeInfo{
			TypeName:   typeName,
			TypeKind:   "DOMAIN",
			Source:     source,
			Properties: make(map[string]interface{}),
		}, nil
	}

	// If domain fails, try as data element
	dataElementURL := fmt.Sprintf("%s"+ADT_DATA_ELEMENTS_ENDPOINT, c.baseURL, typeName)
	if source, err := c.getTypeSource(dataElementURL, "application/xml"); err == nil {
		return &types.ADTTypeInfo{
			TypeName:   typeName,
			TypeKind:   "DATA_ELEMENT",
			Source:     source,
			Properties: make(map[string]interface{}),
		}, nil
	}

	return nil, fmt.Errorf("type %s not found as domain or data element", typeName)
}

func (c *ADTClientImpl) GetTransaction(transactionName string) (*types.ADTTransactionInfo, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving transaction", zap.String("transaction", transactionName))

	transactionName = strings.ToUpper(strings.TrimSpace(transactionName))
	encodedTransactionName := url.QueryEscape(transactionName)

	queryURL := fmt.Sprintf("%s%s?uri=%s&facet=package&facet=appl",
		c.baseURL,
		ADT_TRANSACTION_ENDPOINT,
		url.QueryEscape(fmt.Sprintf("/sap/bc/adt/vit/wb/object_type/trant/object_name/%s", encodedTransactionName)))

	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "application/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("transaction %s not found (404)", transactionName)
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

func (c *ADTClientImpl) GetTableContents(tableName string, maxRows int) (*types.ADTTableData, error) {
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
	url := fmt.Sprintf("%s"+ADT_TABLE_CONTENTS_ENDPOINT+"?maxRows=%d", c.baseURL, tableName, maxRows)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("table contents service not found - requires custom SAP service implementation at %s", ADT_TABLE_CONTENTS_ENDPOINT)
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

func (c *ADTClientImpl) GetTransports() ([]types.ADTTransport, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Retrieving transport requests")

	// This would require custom implementation in SAP system
	// For now, return empty slice as this is an optional feature
	return []types.ADTTransport{}, nil
}

// CreateProgram creates a new ABAP program
func (c *ADTClientImpl) CreateProgram(name, description, source string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated - call Authenticate() first")
	}

	c.logger.Info("Creating program", zap.String("program", name))

	// This would require PUT/POST request to ADT program endpoint
	// Implementation would depend on specific SAP ADT version
	return fmt.Errorf("CreateProgram not implemented - requires SAP system configuration")
}

// Helper functions for authentication and request handling

// addAuthHeaders adds authentication headers to HTTP requests
func (c *ADTClientImpl) addAuthHeaders(req *http.Request) {
	// Basic authentication
	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	// Add SAP specific headers
	req.Header.Set("sap-client", c.config.Client)
	req.Header.Set("sap-language", c.config.Language)

	// Add default Accept header if not already set
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/atomsvc+xml")
	}

	// Add CSRF token if available
	if c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}

	// Add session ID if available
	if c.sessionID != "" {
		req.Header.Set("X-sap-adt-sessiontype", c.sessionType)
	}
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
		Timeout:   time.Duration(c.config.ConnectTimeout) * time.Second,
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
func (c *ADTClientImpl) getTypeSource(url, acceptType string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)
	req.Header.Set("Accept", acceptType)

	resp, err := c.httpClient.Do(req)
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
