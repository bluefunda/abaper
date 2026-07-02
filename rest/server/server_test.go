package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluefunda/abaper/types"
	"go.uber.org/zap"
)

func newTestServer(t *testing.T, client types.ADTClient) *RestServer {
	t.Helper()
	if client == nil {
		return NewRestServer(&Config{}, zap.NewNop(), nil)
	}
	return NewRestServer(&Config{}, zap.NewNop(), client)
}

func doJSON(t *testing.T, rs *RestServer, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	rec := httptest.NewRecorder()
	rs.Handler().ServeHTTP(rec, req)
	return rec
}

func decodeSuccess[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var resp struct {
		Success bool   `json:"success"`
		Data    T      `json:"data"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, rec.Body.String())
	}
	if !resp.Success {
		t.Fatalf("expected success response, got error: %s", resp.Error)
	}
	return resp.Data
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, rec.Body.String())
	}
	if resp.Success {
		t.Fatalf("expected error response, got success")
	}
	return resp.Error
}

// --- objects/get ---

func TestGetObjectHandler(t *testing.T) {
	t.Run("program", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/get", map[string]string{
			"object_type": "program",
			"object_name": "zhello_world",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeSuccess[types.ADTSourceCode](t, rec)
		if data.ObjectName != "ZHELLO_WORLD" {
			t.Errorf("expected uppercased object name, got %q", data.ObjectName)
		}
	})

	t.Run("package via get route", func(t *testing.T) {
		fake := &fakeADTClient{
			getPackageContentsFn: func(ctx context.Context, name string) (*types.ADTPackage, error) {
				return &types.ADTPackage{Name: name, Objects: []types.ADTObject{{Name: "ZFOO", Type: "PROG/P"}}}, nil
			},
		}
		rs := newTestServer(t, fake)
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/get", map[string]string{
			"object_type": "package",
			"object_name": "$tmp",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeSuccess[types.ADTPackage](t, rec)
		if data.Name != "$TMP" {
			t.Errorf("expected uppercased package name, got %q", data.Name)
		}
		if len(data.Objects) != 1 || data.Objects[0].Name != "ZFOO" {
			t.Errorf("expected one object ZFOO, got %+v", data.Objects)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/get", map[string]string{"object_type": "program"})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/get", map[string]string{
			"object_type": "bogus",
			"object_name": "x",
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		if got := decodeError(t, rec); got != "unsupported object type: BOGUS" {
			t.Errorf("unexpected error message: %q", got)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{})
		rec := doJSON(t, rs, http.MethodGet, "/api/v1/objects/get", nil)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})

	t.Run("no client configured", func(t *testing.T) {
		rs := newTestServer(t, nil)
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/get", map[string]string{
			"object_type": "program",
			"object_name": "x",
		})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

// --- objects/create ---

func TestCreateObjectHandler(t *testing.T) {
	t.Run("program with default package", func(t *testing.T) {
		var gotPackage string
		fake := &fakeADTClient{
			createProgramFn: func(ctx context.Context, name, description, packageName, source string) error {
				gotPackage = packageName
				return nil
			},
		}
		rs := newTestServer(t, fake)
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/create", map[string]string{
			"object_type": "program",
			"object_name": "zfoo",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if gotPackage != "$TMP" {
			t.Errorf("expected default package $TMP, got %q", gotPackage)
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/create", map[string]string{
			"object_type": "function",
			"object_name": "x",
		})
		// FUNCTION has no creation case in the handler switch (only update).
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

// --- objects/save ---

func TestSaveObjectHandler(t *testing.T) {
	t.Run("updates class", func(t *testing.T) {
		var gotSource string
		fake := &fakeADTClient{
			updateClassFn: func(ctx context.Context, name, source string) error {
				gotSource = source
				return nil
			},
		}
		rs := newTestServer(t, fake)
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/save", map[string]string{
			"object_type": "class",
			"object_name": "zcl_demo",
			"source":      "CLASS zcl_demo DEFINITION.",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if gotSource != "CLASS zcl_demo DEFINITION." {
			t.Errorf("unexpected source passed through: %q", gotSource)
		}
	})

	t.Run("missing source", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/save", map[string]string{
			"object_type": "class",
			"object_name": "zcl_demo",
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

// --- objects/search ---

func TestSearchObjectsHandler(t *testing.T) {
	t.Run("returns matches", func(t *testing.T) {
		fake := &fakeADTClient{
			searchObjectsFn: func(ctx context.Context, pattern string, objectTypes []string) (*types.ADTSearchResult, error) {
				if pattern != "ZHELL*" {
					t.Errorf("unexpected pattern: %q", pattern)
				}
				return &types.ADTSearchResult{Objects: []types.ADTObject{{Name: "ZHELLO_WORLD", Type: "PROG/P"}}}, nil
			},
		}
		rs := newTestServer(t, fake)
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/search", map[string]string{
			"object_name": "ZHELL*",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeSuccess[types.ADTSearchResult](t, rec)
		if len(data.Objects) != 1 || data.Objects[0].Name != "ZHELLO_WORLD" {
			t.Errorf("unexpected results: %+v", data.Objects)
		}
	})

	t.Run("missing pattern", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/search", map[string]string{})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

// --- objects/list (regression coverage for the package-filter fix) ---

func TestListObjectsHandler(t *testing.T) {
	t.Run("lists package contents", func(t *testing.T) {
		fake := &fakeADTClient{
			getPackageContentsFn: func(ctx context.Context, name string) (*types.ADTPackage, error) {
				if name != "$TMP" {
					t.Errorf("expected package $TMP, got %q", name)
				}
				return &types.ADTPackage{Name: name, Objects: []types.ADTObject{
					{Name: "ZHELLO_WORLD", Type: "PROG/P"},
					{Name: "ZCL_DEMO", Type: "CLAS/OC"},
				}}, nil
			},
		}
		rs := newTestServer(t, fake)
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/list", map[string]string{
			"package": "$tmp",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeSuccess[[]types.ADTObject](t, rec)
		if len(data) != 2 {
			t.Fatalf("expected 2 objects, got %d", len(data))
		}
	})

	t.Run("filters package contents by type", func(t *testing.T) {
		fake := &fakeADTClient{
			getPackageContentsFn: func(ctx context.Context, name string) (*types.ADTPackage, error) {
				return &types.ADTPackage{Name: name, Objects: []types.ADTObject{
					{Name: "ZHELLO_WORLD", Type: "PROG/P"},
					{Name: "ZHELLO_GO_TEST", Type: "PROG/P"},
					{Name: "ZCL_DEMO", Type: "CLAS/OC"},
				}}, nil
			},
		}
		rs := newTestServer(t, fake)
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/list", map[string]string{
			"package":     "$TMP",
			"object_type": "PROG",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeSuccess[[]types.ADTObject](t, rec)
		if len(data) != 2 {
			t.Fatalf("expected 2 PROG objects, got %d: %+v", len(data), data)
		}
		for _, obj := range data {
			if obj.Type != "PROG/P" {
				t.Errorf("unexpected object leaked through type filter: %+v", obj)
			}
		}
	})

	t.Run("lists packages by pattern", func(t *testing.T) {
		fake := &fakeADTClient{
			listPackagesFn: func(ctx context.Context, pattern string) ([]types.ADTPackage, error) {
				if pattern != "Z*" {
					t.Errorf("expected pattern Z*, got %q", pattern)
				}
				return []types.ADTPackage{{Name: "ZDEMO"}}, nil
			},
		}
		rs := newTestServer(t, fake)
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/list", map[string]string{
			"object_type": "packages",
			"object_name": "Z*",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeSuccess[[]types.ADTPackage](t, rec)
		if len(data) != 1 || data[0].Name != "ZDEMO" {
			t.Errorf("unexpected packages: %+v", data)
		}
	})

	t.Run("requires package when not listing package names", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/list", map[string]string{
			"object_type": "PROG",
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

// --- objects/activate ---

func TestActivateObjectHandler(t *testing.T) {
	t.Run("activates without source", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/activate", map[string]string{
			"object_type": "program",
			"object_name": "zfoo",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeSuccess[types.ActivationResult](t, rec)
		if !data.Success {
			t.Errorf("expected activation success")
		}
	})

	t.Run("saves source before activating", func(t *testing.T) {
		var saved bool
		fake := &fakeADTClient{
			updateProgramFn: func(ctx context.Context, name, source string) error {
				saved = true
				return nil
			},
		}
		rs := newTestServer(t, fake)
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/objects/activate", map[string]string{
			"object_type": "program",
			"object_name": "zfoo",
			"source":      "REPORT zfoo.",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if !saved {
			t.Errorf("expected UpdateProgram to be called before activation")
		}
	})
}

// --- syntax-check, format, completion, navigation ---

func TestSyntaxCheckHandler(t *testing.T) {
	rs := newTestServer(t, &fakeADTClient{})
	rec := doJSON(t, rs, http.MethodPost, "/api/v1/syntax-check", map[string]string{
		"object_type": "program",
		"object_name": "zfoo",
		"source":      "REPORT zfoo.",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestFormatSourceHandler(t *testing.T) {
	rs := newTestServer(t, &fakeADTClient{})
	rec := doJSON(t, rs, http.MethodPost, "/api/v1/format", map[string]string{
		"source": "report zfoo.",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	data := decodeSuccess[map[string]string](t, rec)
	if data["source"] != "report zfoo." {
		t.Errorf("expected passthrough formatting from fake, got %q", data["source"])
	}
}

func TestCompletionHandler(t *testing.T) {
	rs := newTestServer(t, &fakeADTClient{})
	rec := doJSON(t, rs, http.MethodPost, "/api/v1/completion", map[string]any{
		"object_type": "program",
		"object_name": "zfoo",
		"source":      "REPORT zfoo.",
		"line":        1,
		"column":      1,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNavigationHandler(t *testing.T) {
	rs := newTestServer(t, &fakeADTClient{})
	rec := doJSON(t, rs, http.MethodPost, "/api/v1/navigation", map[string]any{
		"object_type": "program",
		"object_name": "zfoo",
		"source":      "REPORT zfoo.",
		"line":        1,
		"column":      1,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- unit-tests, transports ---

func TestUnitTestsHandler(t *testing.T) {
	rs := newTestServer(t, &fakeADTClient{})
	rec := doJSON(t, rs, http.MethodPost, "/api/v1/unit-tests", map[string]string{
		"object_type": "class",
		"object_name": "zcl_demo",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	data := decodeSuccess[types.UnitTestResult](t, rec)
	if !data.AllPassed {
		t.Errorf("expected AllPassed from fake default")
	}
}

func TestTransportInfoHandler(t *testing.T) {
	rs := newTestServer(t, &fakeADTClient{})
	rec := doJSON(t, rs, http.MethodPost, "/api/v1/transports/info", map[string]string{
		"object_type": "program",
		"object_name": "zfoo",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTransportHandler(t *testing.T) {
	t.Run("creates transport", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/transports/create", map[string]string{
			"object_type": "program",
			"object_name": "zfoo",
			"description": "test transport",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing description", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/transports/create", map[string]string{
			"object_type": "program",
			"object_name": "zfoo",
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

// --- system/connect, health, version ---

func TestConnectHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{authenticated: true})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/system/connect", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("connection failure", func(t *testing.T) {
		rs := newTestServer(t, &fakeADTClient{testConnErr: context.DeadlineExceeded})
		rec := doJSON(t, rs, http.MethodPost, "/api/v1/system/connect", nil)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestHealthHandler(t *testing.T) {
	rs := newTestServer(t, &fakeADTClient{authenticated: true})
	rec := doJSON(t, rs, http.MethodGet, "/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body["adt_status"] != "connected" {
		t.Errorf("expected adt_status=connected, got %v", body["adt_status"])
	}
}

func TestVersionHandler(t *testing.T) {
	rs := newTestServer(t, &fakeADTClient{})
	rec := doJSON(t, rs, http.MethodGet, "/version", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// --- removed AI endpoints stay gone ---

func TestRemovedAIHandlersReturn410(t *testing.T) {
	rs := newTestServer(t, &fakeADTClient{})
	for _, path := range []string{
		"/api/v1/ai/analyze",
		"/api/v1/ai/review",
		"/api/v1/ai/optimize",
		"/api/v1/ai/create",
		"/generate-code",
		"/generate-code-stream",
	} {
		rec := doJSON(t, rs, http.MethodPost, path, nil)
		if rec.Code != http.StatusGone {
			t.Errorf("%s: expected 410 Gone, got %d", path, rec.Code)
		}
	}
}
