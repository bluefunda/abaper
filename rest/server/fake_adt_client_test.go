package server

import (
	"context"

	"github.com/bluefunda/abaper/types"
)

// fakeADTClient is a scriptable stand-in for types.ADTClient used to unit
// test REST handlers without a live SAP system. Each field defaults to a
// zero-value success response; tests override only what they need.
type fakeADTClient struct {
	authenticated bool
	testConnErr   error

	getProgramFn         func(ctx context.Context, name string) (*types.ADTSourceCode, error)
	getPackageContentsFn func(ctx context.Context, name string) (*types.ADTPackage, error)
	getNodeContentsFn    func(ctx context.Context, name string) (*types.PackageContentsResult, error)
	listPackagesFn       func(ctx context.Context, pattern string) ([]types.ADTPackage, error)
	searchObjectsFn      func(ctx context.Context, pattern string, objectTypes []string) (*types.ADTSearchResult, error)
	updateProgramFn      func(ctx context.Context, name, source string) error
	updateClassFn        func(ctx context.Context, name, source string) error
	createProgramFn      func(ctx context.Context, name, description, packageName, source string) error
	createClassFn        func(ctx context.Context, name, description, packageName, source string) error
	activateObjectFn     func(ctx context.Context, objectType, objectName string) (*types.ActivationResult, error)
	runUnitTestsFn       func(ctx context.Context, objectType, objectName string) (*types.UnitTestResult, error)
	syntaxCheckFn        func(ctx context.Context, objectType, objectName, source string) (*types.SyntaxCheckResult, error)
	formatSourceFn       func(ctx context.Context, source string) (string, error)
	completionFn         func(ctx context.Context, objectType, objectName, source string, line, col int) ([]types.CompletionProposal, error)
	navigationFn         func(ctx context.Context, objectType, objectName, source string, line, col int) (*types.NavigationTarget, error)
	transportInfoFn      func(ctx context.Context, objectType, objectName string) (*types.ADTTransportInfo, error)
	createTransportFn    func(ctx context.Context, objectType, objectName, description, packageName string) (string, error)
}

func (f *fakeADTClient) Authenticate() error              { return nil }
func (f *fakeADTClient) IsAuthenticated() bool            { return f.authenticated }
func (f *fakeADTClient) TestConnection() error            { return f.testConnErr }
func (f *fakeADTClient) SetSessionType(types.SessionType) {}

func (f *fakeADTClient) GetProgram(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	if f.getProgramFn != nil {
		return f.getProgramFn(ctx, name)
	}
	return &types.ADTSourceCode{ObjectName: name, ObjectType: "PROGRAM"}, nil
}
func (f *fakeADTClient) GetClass(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return &types.ADTSourceCode{ObjectName: name, ObjectType: "CLASS"}, nil
}
func (f *fakeADTClient) GetInclude(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return &types.ADTSourceCode{ObjectName: name, ObjectType: "INCLUDE"}, nil
}
func (f *fakeADTClient) GetInterface(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return &types.ADTSourceCode{ObjectName: name, ObjectType: "INTERFACE"}, nil
}
func (f *fakeADTClient) GetStructure(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return &types.ADTSourceCode{ObjectName: name, ObjectType: "STRUCTURE"}, nil
}
func (f *fakeADTClient) GetTable(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return &types.ADTSourceCode{ObjectName: name, ObjectType: "TABLE"}, nil
}
func (f *fakeADTClient) GetFunction(ctx context.Context, functionName, functionGroup string) (*types.ADTSourceCode, error) {
	return &types.ADTSourceCode{ObjectName: functionName, ObjectType: "FUNCTION"}, nil
}
func (f *fakeADTClient) GetFunctionGroup(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return &types.ADTSourceCode{ObjectName: name, ObjectType: "FUNCTIONGROUP"}, nil
}
func (f *fakeADTClient) GetDDLSource(ctx context.Context, name string) (*types.ADTSourceCode, error) {
	return &types.ADTSourceCode{ObjectName: name, ObjectType: "DDLS"}, nil
}
func (f *fakeADTClient) GetObjectSource(ctx context.Context, objectType, objectName string) (string, error) {
	return "", nil
}
func (f *fakeADTClient) CheckObjectExists(ctx context.Context, objectType, objectName string) (bool, error) {
	return true, nil
}

func (f *fakeADTClient) CreateProgram(ctx context.Context, name, description, packageName, source string) error {
	if f.createProgramFn != nil {
		return f.createProgramFn(ctx, name, description, packageName, source)
	}
	return nil
}
func (f *fakeADTClient) CreateClass(ctx context.Context, name, description, packageName, source string) error {
	if f.createClassFn != nil {
		return f.createClassFn(ctx, name, description, packageName, source)
	}
	return nil
}
func (f *fakeADTClient) CreateInclude(ctx context.Context, name, description, source string) error {
	return nil
}
func (f *fakeADTClient) CreateInterface(ctx context.Context, name, description, source string) error {
	return nil
}
func (f *fakeADTClient) CreateStructure(ctx context.Context, name, description, source string) error {
	return nil
}
func (f *fakeADTClient) CreateTable(ctx context.Context, name, description, source string) error {
	return nil
}
func (f *fakeADTClient) CreateFunctionGroup(ctx context.Context, name, description, source string) error {
	return nil
}
func (f *fakeADTClient) UpdateProgram(ctx context.Context, name, source string) error {
	if f.updateProgramFn != nil {
		return f.updateProgramFn(ctx, name, source)
	}
	return nil
}
func (f *fakeADTClient) UpdateClass(ctx context.Context, name, source string) error {
	if f.updateClassFn != nil {
		return f.updateClassFn(ctx, name, source)
	}
	return nil
}
func (f *fakeADTClient) UpdateInclude(ctx context.Context, name, source string) error { return nil }
func (f *fakeADTClient) UpdateInterface(ctx context.Context, name, source string) error {
	return nil
}
func (f *fakeADTClient) UpdateFunction(ctx context.Context, functionName, functionGroup, source string) error {
	return nil
}
func (f *fakeADTClient) UpdateFunctionGroup(ctx context.Context, name, source string) error {
	return nil
}

func (f *fakeADTClient) SearchObjects(ctx context.Context, pattern string, objectTypes []string) (*types.ADTSearchResult, error) {
	if f.searchObjectsFn != nil {
		return f.searchObjectsFn(ctx, pattern, objectTypes)
	}
	return &types.ADTSearchResult{}, nil
}
func (f *fakeADTClient) ListPackages(ctx context.Context, pattern string) ([]types.ADTPackage, error) {
	if f.listPackagesFn != nil {
		return f.listPackagesFn(ctx, pattern)
	}
	return nil, nil
}
func (f *fakeADTClient) GetPackageContents(ctx context.Context, name string) (*types.ADTPackage, error) {
	if f.getPackageContentsFn != nil {
		return f.getPackageContentsFn(ctx, name)
	}
	return &types.ADTPackage{Name: name}, nil
}
func (f *fakeADTClient) GetNodeContents(ctx context.Context, name string) (*types.PackageContentsResult, error) {
	if f.getNodeContentsFn != nil {
		return f.getNodeContentsFn(ctx, name)
	}
	return &types.PackageContentsResult{
		Nodes:       []types.PackageNode{{Name: name, Type: "DEVC/K", Expandable: true, URI: "/sap/bc/adt/packages/" + name}},
		ObjectTypes: []types.PackageObjectType{{Type: "DEVC/K", Label: "Packages"}},
	}, nil
}

func (f *fakeADTClient) ActivateObject(ctx context.Context, objectType, objectName string) (*types.ActivationResult, error) {
	if f.activateObjectFn != nil {
		return f.activateObjectFn(ctx, objectType, objectName)
	}
	return &types.ActivationResult{ObjectName: objectName, ObjectType: objectType, Success: true}, nil
}
func (f *fakeADTClient) RunUnitTests(ctx context.Context, objectType, objectName string) (*types.UnitTestResult, error) {
	if f.runUnitTestsFn != nil {
		return f.runUnitTestsFn(ctx, objectType, objectName)
	}
	return &types.UnitTestResult{ObjectName: objectName, AllPassed: true}, nil
}

func (f *fakeADTClient) SyntaxCheck(ctx context.Context, objectType, objectName, source string) (*types.SyntaxCheckResult, error) {
	if f.syntaxCheckFn != nil {
		return f.syntaxCheckFn(ctx, objectType, objectName, source)
	}
	return &types.SyntaxCheckResult{ObjectName: objectName, ObjectType: objectType}, nil
}
func (f *fakeADTClient) GetCompletionProposals(ctx context.Context, objectType, objectName, source string, line, col int) ([]types.CompletionProposal, error) {
	if f.completionFn != nil {
		return f.completionFn(ctx, objectType, objectName, source, line, col)
	}
	return nil, nil
}
func (f *fakeADTClient) GetNavigationTarget(ctx context.Context, objectType, objectName, source string, line, col int) (*types.NavigationTarget, error) {
	if f.navigationFn != nil {
		return f.navigationFn(ctx, objectType, objectName, source, line, col)
	}
	return &types.NavigationTarget{}, nil
}
func (f *fakeADTClient) FormatSource(ctx context.Context, source string) (string, error) {
	if f.formatSourceFn != nil {
		return f.formatSourceFn(ctx, source)
	}
	return source, nil
}

func (f *fakeADTClient) GetTypeInfo(ctx context.Context, typeName string) (*types.ADTTypeInfo, error) {
	return &types.ADTTypeInfo{TypeName: typeName}, nil
}
func (f *fakeADTClient) GetTransaction(ctx context.Context, transactionName string) (*types.ADTTransactionInfo, error) {
	return &types.ADTTransactionInfo{TransactionCode: transactionName}, nil
}
func (f *fakeADTClient) GetTableContents(ctx context.Context, tableName string, maxRows int) (*types.ADTTableData, error) {
	return &types.ADTTableData{TableName: tableName}, nil
}
func (f *fakeADTClient) GetTransports(ctx context.Context) ([]types.ADTTransport, error) {
	return nil, nil
}
func (f *fakeADTClient) GetTransportInfo(ctx context.Context, objectType, objectName string) (*types.ADTTransportInfo, error) {
	if f.transportInfoFn != nil {
		return f.transportInfoFn(ctx, objectType, objectName)
	}
	return &types.ADTTransportInfo{ObjectName: objectName}, nil
}
func (f *fakeADTClient) CreateTransport(ctx context.Context, objectType, objectName, description, packageName string) (string, error) {
	if f.createTransportFn != nil {
		return f.createTransportFn(ctx, objectType, objectName, description, packageName)
	}
	return "TR0001", nil
}

var _ types.ADTClient = (*fakeADTClient)(nil)
