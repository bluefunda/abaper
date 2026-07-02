package adt

import (
	"encoding/xml"
	"strings"
	"testing"
)

// These tests marshal the create-shell XML payloads used by the various
// Create* methods and assert on the exact type ID / root element abap-adt-api
// (the reference implementation) expects SAP ADT to receive. They exist
// because a prior bug shipped CreateFunctionGroup with the function *module*
// type ID (FUGR/FF) and the wrong root element, which SAP silently rejected
// with "Data is invalid and could not be converted" only when tested live.

func TestFunctionGroupCreatePayload(t *testing.T) {
	payload := functionGroupCreatePayload{
		FgroupNS:    "http://www.sap.com/adt/functions/groups",
		AdtcoreNS:   "http://www.sap.com/adt/core",
		Description: "test group",
		Name:        "ZTEST_FG",
		Type:        "FUGR/F",
		Responsible: "DEVELOPER",
		PackageRef:  classPackageRef{Name: "$TMP"},
	}
	out, err := xml.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	xmlStr := string(out)

	if !strings.Contains(xmlStr, `<group:abapFunctionGroup`) {
		t.Errorf("expected root element group:abapFunctionGroup, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `adtcore:type="FUGR/F"`) {
		t.Errorf("expected adtcore:type=\"FUGR/F\" (function GROUP, not module), got: %s", xmlStr)
	}
	if strings.Contains(xmlStr, `FUGR/FF`) {
		t.Errorf("function group payload must not use the function MODULE type ID FUGR/FF: %s", xmlStr)
	}
}

func TestFunctionModuleCreatePayload(t *testing.T) {
	payload := functionModuleCreatePayload{
		FmoduleNS:   "http://www.sap.com/adt/functions/fmodules",
		AdtcoreNS:   "http://www.sap.com/adt/core",
		Description: "test module",
		Name:        "ZTEST_FM",
		Type:        "FUGR/FF",
		ContainerRef: functionContainerRef{
			Name: "ZTEST_FG",
			Type: "FUGR/F",
			URI:  "/sap/bc/adt/functions/groups/ztest_fg",
		},
	}
	out, err := xml.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	xmlStr := string(out)

	if !strings.Contains(xmlStr, `<fmodule:abapFunctionModule`) {
		t.Errorf("expected root element fmodule:abapFunctionModule, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `adtcore:type="FUGR/FF"`) {
		t.Errorf("expected adtcore:type=\"FUGR/FF\", got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `<adtcore:containerRef`) {
		t.Errorf("function module create must reference its containing group via containerRef, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `adtcore:name="ZTEST_FG"`) {
		t.Errorf("containerRef must name the containing function group, got: %s", xmlStr)
	}
}

func TestDDLSourceCreatePayload(t *testing.T) {
	payload := ddlSourceCreatePayload{
		DdlNS:       "http://www.sap.com/adt/ddic/ddlsources",
		AdtcoreNS:   "http://www.sap.com/adt/core",
		Description: "test CDS view",
		Name:        "ZTEST_CDS",
		Type:        "DDLS/DF",
		Responsible: "DEVELOPER",
		PackageRef:  classPackageRef{Name: "$TMP"},
	}
	out, err := xml.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	xmlStr := string(out)

	if !strings.Contains(xmlStr, `<ddl:ddlSource`) {
		t.Errorf("expected root element ddl:ddlSource, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `adtcore:type="DDLS/DF"`) {
		t.Errorf("expected adtcore:type=\"DDLS/DF\", got: %s", xmlStr)
	}
}
