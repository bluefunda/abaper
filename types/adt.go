package types

import "encoding/xml"

// ADT Response structures - shared between CLI and REST
type ADTObject struct {
	Name        string `json:"name" xml:"name"`
	Type        string `json:"type" xml:"type"`
	Description string `json:"description" xml:"description"`
	Package     string `json:"package" xml:"package"`
	Responsible string `json:"responsible" xml:"responsible"`
	CreatedBy   string `json:"created_by" xml:"createdBy"`
	CreatedOn   string `json:"created_on" xml:"createdOn"`
	ChangedBy   string `json:"changed_by" xml:"changedBy"`
	ChangedOn   string `json:"changed_on" xml:"changedOn"`
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
	XMLName xml.Name    `xml:"objectReferences"` // root element
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
	TableName string                   `json:"table_name"`
	RowCount  int                      `json:"row_count"`
	Columns   []ADTTableColumn         `json:"columns"`
	Rows      []map[string]interface{} `json:"rows"`
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
	CreateProgram(name, description, source string) error
}
