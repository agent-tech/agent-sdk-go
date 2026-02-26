package pay

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- NewClient validation ---

func TestNewClient_EmptyBaseURL(t *testing.T) {
	_, err := NewClient("", WithBearerAuth("id", "secret"))
	assertValidationError(t, err, "baseURL")
}

func TestNewClient_NoAuth(t *testing.T) {
	_, err := NewClient("http://localhost")
	assertValidationError(t, err, "auth option")
}

func TestNewClient_EmptyBearerCredentials(t *testing.T) {
	_, err := NewClient("http://localhost", WithBearerAuth("", "secret"))
	assertValidationError(t, err, "clientID")

	_, err = NewClient("http://localhost", WithBearerAuth("id", ""))
	assertValidationError(t, err, "clientSecret")
}

func TestNewClient_EmptyAPIKeyCredentials(t *testing.T) {
	_, err := NewClient("http://localhost", WithAPIKeyAuth("", "key"))
	assertValidationError(t, err, "clientID")

	_, err = NewClient("http://localhost", WithAPIKeyAuth("id", ""))
	assertValidationError(t, err, "apiKey")
}

func TestNewClient_TrailingSlash(t *testing.T) {
	c, err := NewClient("http://localhost/", WithBearerAuth("id", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	if c.baseURL != "http://localhost" {
		t.Errorf("baseURL = %q, want no trailing slash", c.baseURL)
	}
}

// --- Option behavior ---

func TestWithTimeout(t *testing.T) {
	c, err := NewClient("http://localhost",
		WithBearerAuth("id", "secret"),
		WithTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient.Timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", c.httpClient.Timeout)
	}
}

func TestWithTimeout_IgnoredAfterCustomClient(t *testing.T) {
	custom := &http.Client{Timeout: 90 * time.Second}
	c, err := NewClient("http://localhost",
		WithBearerAuth("id", "secret"),
		WithHTTPClient(custom),
		WithTimeout(5*time.Second), // should be ignored
	)
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient.Timeout != 90*time.Second {
		t.Errorf("timeout = %v, want 90s (custom client should be unmodified)", c.httpClient.Timeout)
	}
}

func TestWithTimeout_IgnoredBeforeCustomClient(t *testing.T) {
	custom := &http.Client{Timeout: 90 * time.Second}
	c, err := NewClient("http://localhost",
		WithBearerAuth("id", "secret"),
		WithTimeout(5*time.Second),  // applied to default, then replaced
		WithHTTPClient(custom),
	)
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient != custom {
		t.Error("expected custom http client")
	}
	if c.httpClient.Timeout != 90*time.Second {
		t.Errorf("timeout = %v, want 90s", c.httpClient.Timeout)
	}
}

func TestWithHTTPClient_NilIgnored(t *testing.T) {
	c, err := NewClient("http://localhost",
		WithBearerAuth("id", "secret"),
		WithHTTPClient(nil),
	)
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

// --- Error types ---

func TestAPIError_ErrorString(t *testing.T) {
	e := &APIError{StatusCode: 400, Message: "bad request"}
	want := "api error 400: bad request"
	if e.Error() != want {
		t.Errorf("Error() = %q, want %q", e.Error(), want)
	}
}

func TestValidationError_ErrorString(t *testing.T) {
	e := &ValidationError{Message: "field required"}
	want := "validation: field required"
	if e.Error() != want {
		t.Errorf("Error() = %q, want %q", e.Error(), want)
	}
}

func TestErrorsAs_APIError(t *testing.T) {
	var target *APIError
	err := error(&APIError{StatusCode: 404, Message: "not found"})
	if !errors.As(err, &target) {
		t.Error("errors.As should match *APIError")
	}
}

func TestErrorsAs_ValidationError(t *testing.T) {
	var target *ValidationError
	err := error(&ValidationError{Message: "bad"})
	if !errors.As(err, &target) {
		t.Error("errors.As should match *ValidationError")
	}
}

// --- API methods with httptest ---

func TestCreateIntent_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v2/intents" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization header")
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateIntentResponse{
			IntentBase: IntentBase{IntentID: "intent-1", Status: StatusAwaitingPayment},
		})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, WithBearerAuth("id", "secret"))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.CreateIntent(context.Background(), &CreateIntentRequest{
		Email:      "test@example.com",
		Amount:     "10.00",
		PayerChain: "solana",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.IntentID != "intent-1" {
		t.Errorf("IntentID = %q, want %q", resp.IntentID, "intent-1")
	}
}

func TestCreateIntent_NilRequest(t *testing.T) {
	c, _ := NewClient("http://localhost", WithBearerAuth("id", "secret"))
	_, err := c.CreateIntent(context.Background(), nil)
	assertValidationError(t, err, "nil")
}

func TestCreateIntent_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "invalid amount"})
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, WithBearerAuth("id", "secret"))
	_, err := c.CreateIntent(context.Background(), &CreateIntentRequest{Amount: "0"})

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
	if apiErr.Message != "invalid amount" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "invalid amount")
	}
}

func TestExecuteIntent_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/intents/abc-123/execute" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ExecuteIntentResponse{
			IntentBase: IntentBase{IntentID: "abc-123", Status: StatusBaseSettled},
		})
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, WithBearerAuth("id", "secret"))
	resp, err := c.ExecuteIntent(context.Background(), "abc-123")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != StatusBaseSettled {
		t.Errorf("Status = %q", resp.Status)
	}
}

func TestExecuteIntent_EmptyID(t *testing.T) {
	c, _ := NewClient("http://localhost", WithBearerAuth("id", "secret"))
	_, err := c.ExecuteIntent(context.Background(), "")
	assertValidationError(t, err, "intent_id")
}

func TestGetIntent_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("intent_id") != "xyz" {
			t.Errorf("intent_id = %q", r.URL.Query().Get("intent_id"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GetIntentResponse{
			IntentBase: IntentBase{IntentID: "xyz", Status: StatusBaseSettled},
		})
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, WithBearerAuth("id", "secret"))
	resp, err := c.GetIntent(context.Background(), "xyz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.IntentID != "xyz" {
		t.Errorf("IntentID = %q", resp.IntentID)
	}
}

func TestGetIntent_EmptyID(t *testing.T) {
	c, _ := NewClient("http://localhost", WithBearerAuth("id", "secret"))
	_, err := c.GetIntent(context.Background(), "")
	assertValidationError(t, err, "intent_id")
}

func TestAPIKeyAuth_Headers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Client-ID") != "myid" {
			t.Errorf("X-Client-ID = %q", r.Header.Get("X-Client-ID"))
		}
		if r.Header.Get("X-API-Key") != "mykey" {
			t.Errorf("X-API-Key = %q", r.Header.Get("X-API-Key"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GetIntentResponse{})
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, WithAPIKeyAuth("myid", "mykey"))
	_, _ = c.GetIntent(context.Background(), "test-id")
}

func TestParseError_FallsBackToErrorField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "forbidden"})
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, WithBearerAuth("id", "secret"))
	_, err := c.GetIntent(context.Background(), "id")

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %v", err)
	}
	if apiErr.Message != "forbidden" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "forbidden")
	}
}

func TestParseError_FallsBackToStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, WithBearerAuth("id", "secret"))
	_, err := c.GetIntent(context.Background(), "id")

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %v", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

// --- helpers ---

func assertValidationError(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if substr != "" && !strings.Contains(valErr.Message, substr) {
		t.Errorf("error message %q does not contain %q", valErr.Message, substr)
	}
}
