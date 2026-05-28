// Package types defines the ADTClient interface and shared data structures
// for interacting with SAP ABAP Development Tools (ADT) REST APIs.
// These types are shared across the CLI, REST server, and LSP server.
package types

import "encoding/xml"

// ADT Response structures - shared between CLI and REST
type ADTObject struct {
	Name        string `json:"name" xml:"name,attr"`
	Type        string `json:"type" xml:"type,attr"`
	Description string `json:"description" xml:"description,attr"`
	Package     string `json:"package" xml:"packageName,attr"`
	Responsible string `json:"responsible" xml:"responsible,attr"`
	CreatedBy   string `json:"created_by" xml:"createdBy,attr"`
	CreatedOn   string `json:"created_on" xml:"createdOn,attr"`
	ChangedBy   string `json:"changed_by" xml:"changedBy,attr"`
	ChangedOn   string `json:"changed_on" xml:"changedOn,attr"`
}

type ADTPackage struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Objects     []ADTObject `json:"objects"`
}

type ADTSourceCode struct {
	ObjectName string `json:"object_name"`
	ObjectType string `json:"object_type"`
	Source     string `json:"source"`
	Version    string `json:"version"`
	ETag       string `json:"etag"`
}

type ADTSearchResult struct {
	XMLName xml.Name    `xml:"http://www.sap.com/adt/core objectReferences"` // root element with namespace
	Objects []ADTObject `xml:"objectReference"`
	Total   int         `xml:"total,attr"`
}

type ADTNode struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Children    []ADTNode `json:"children,omitempty"`
}

type ADTTransport struct {
	RequestID   string      `json:"request_id"`
	Description string      `json:"description"`
	Status      string      `json:"status"`
	Owner       string      `json:"owner"`
	Objects     []ADTObject `json:"objects"`
}

// ADT Configuration
type ADTConfig struct {
	Host            string `json:"host"`
	Client          string `json:"client"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Language        string `json:"language"`
	AllowSelfSigned bool   `json:"allow_self_signed"`
	ConnectTimeout  int    `json:"connect_timeout"`
	RequestTimeout  int    `json:"request_timeout"`
	Debug           bool   `json:"debug"`
}

// Additional data structures for extended services
type ADTTransactionInfo struct {
	TransactionCode string            `json:"transaction_code"`
	Description     string            `json:"description"`
	Package         string            `json:"package"`
	Application     string            `json:"application"`
	Program         string            `json:"program"`
	Properties      map[string]string `json:"properties"`
}

type ADTTableData struct {
	TableName string           `json:"table_name"`
	RowCount  int              `json:"row_count"`
	Columns   []ADTTableColumn `json:"columns"`
	Rows      []map[string]any `json:"rows"`
}

type ADTTableColumn struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Length   int    `json:"length"`
	Decimals int    `json:"decimals"`
}

type ADTTypeInfo struct {
	TypeName    string                 `json:"type_name"`
	TypeKind    string                 `json:"type_kind"` // "DOMAIN", "DATA_ELEMENT", etc.
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Properties  map[string]interface{} `json:"properties"`
}

// ActivationResult holds the result of an object activation
type ActivationResult struct {
	ObjectName string              `json:"object_name"`
	ObjectType string              `json:"object_type"`
	Success    bool                `json:"success"`
	Messages   []ActivationMessage `json:"messages,omitempty"`
}

// ActivationMessage represents a single message from the activation response
type ActivationMessage struct {
	Severity string `json:"severity"` // "error", "warning", "info"
	Text     string `json:"text"`
	Line     int    `json:"line,omitempty"`
}

// UnitTestResult holds the result of running ABAP unit tests
type UnitTestResult struct {
	ObjectName  string            `json:"object_name"`
	TotalTests  int               `json:"total_tests"`
	Passed      int               `json:"passed"`
	Failed      int               `json:"failed"`
	AllPassed   bool              `json:"all_passed"`
	TestClasses []TestClassResult `json:"test_classes"`
}

// TestClassResult holds results for a single test class
type TestClassResult struct {
	Name    string             `json:"name"`
	Methods []TestMethodResult `json:"methods"`
}

// TestMethodResult holds result for a single test method
type TestMethodResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "passed", "failed", "error"
	Message string `json:"message,omitempty"`
}

// Session management
type SessionType string

const (
	SessionStateful  SessionType = "stateful"
	SessionStateless SessionType = "stateless"
)

// ADTClient interface - shared contract
type ADTClient interface {
	// Core object retrieval methods
	GetProgram(name string) (*ADTSourceCode, error)
	GetClass(name string) (*ADTSourceCode, error)
	GetFunction(name, functionGroup string) (*ADTSourceCode, error)
	GetInclude(name string) (*ADTSourceCode, error)
	GetInterface(name string) (*ADTSourceCode, error)
	GetStructure(name string) (*ADTSourceCode, error)
	GetTable(name string) (*ADTSourceCode, error)
	GetFunctionGroup(name string) (*ADTSourceCode, error)

	// Package and search operations
	GetPackageContents(name string) (*ADTPackage, error)
	SearchObjects(pattern string, objectTypes []string) (*ADTSearchResult, error)
	ListPackages(pattern string) ([]ADTPackage, error)

	// Connection and session management
	TestConnection() error
	IsAuthenticated() bool
	Authenticate() error
	SetSessionType(sessionType SessionType)

	// Extended operations (optional implementations)
	GetTypeInfo(typeName string) (*ADTTypeInfo, error)
	GetTransaction(transactionName string) (*ADTTransactionInfo, error)
	GetTableContents(tableName string, maxRows int) (*ADTTableData, error)
	GetTransports() ([]ADTTransport, error)
	CreateProgram(name, description, packageName, source string) error
	CreateInclude(name, description, source string) error
	CreateClass(name, description, packageName, source string) error
	CreateInterface(name, description, source string) error
	CreateStructure(name, description, source string) error
	CreateTable(name, description, source string) error
	CreateFunctionGroup(name, description, source string) error
	// Update operations
	UpdateProgram(name, source string) error
	UpdateClass(name, source string) error
	UpdateInclude(name, source string) error
	UpdateInterface(name, source string) error
	// Object existence checking
	CheckObjectExists(objectType, objectName string) (bool, error)
	GetObjectSource(objectType, objectName string) (string, error)

	// Activation and testing
	ActivateObject(objectType, objectName string) (*ActivationResult, error)
	RunUnitTests(objectType, objectName string) (*UnitTestResult, error)

	// LSP support - syntax check, code completion, navigation
	SyntaxCheck(objectType, objectName, source string) (*SyntaxCheckResult, error)
	GetCompletionProposals(objectType, objectName, source string, line, column int) ([]CompletionProposal, error)
	GetNavigationTarget(objectType, objectName, source string, line, column int) (*NavigationTarget, error)
}
