package pay

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewPublicClient_EmptyBaseURL(t *testing.T) {
	_, err := NewPublicClient("")
	assertValidationError(t, err, "baseURL")
}

func TestNewPublicClient_TrailingSlash(t *testing.T) {
	c, err := NewPublicClient("http://localhost/")
	if err != nil {
		t.Fatal(err)
	}
	if c.baseURL != "http://localhost" {
		t.Errorf("baseURL = %q, want no trailing slash", c.baseURL)
	}
}

func TestPublicClient_CreateIntent_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/intents" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("public API should not send Authorization header")
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateIntentResponse{
			IntentBase: IntentBase{IntentID: "intent-public-1", Status: StatusAwaitingPayment},
			PayerChain: "solana",
		})
	}))
	defer srv.Close()

	c, err := NewPublicClient(srv.URL)
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
	if resp.IntentID != "intent-public-1" {
		t.Errorf("IntentID = %q, want %q", resp.IntentID, "intent-public-1")
	}
}

func TestPublicClient_SubmitProof_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/intents/xyz-789" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var body struct {
			SettleProof string `json:"settle_proof"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.SettleProof != "proof-base64-here" {
			t.Errorf("settle_proof = %q", body.SettleProof)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SubmitProofResponse{
			IntentBase: IntentBase{IntentID: "xyz-789", Status: StatusPending},
		})
	}))
	defer srv.Close()

	c, _ := NewPublicClient(srv.URL)
	resp, err := c.SubmitProof(context.Background(), "xyz-789", "proof-base64-here")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != StatusPending {
		t.Errorf("Status = %q", resp.Status)
	}
}

func TestPublicClient_SubmitProof_EmptyIntentID(t *testing.T) {
	c, _ := NewPublicClient("http://localhost")
	_, err := c.SubmitProof(context.Background(), "", "proof")
	assertValidationError(t, err, "intent_id")
}

func TestPublicClient_SubmitProof_EmptySettleProof(t *testing.T) {
	c, _ := NewPublicClient("http://localhost")
	_, err := c.SubmitProof(context.Background(), "intent-1", "")
	assertValidationError(t, err, "settle_proof")
}

func TestPublicClient_GetIntent_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("intent_id") != "abc" {
			t.Errorf("intent_id = %q", r.URL.Query().Get("intent_id"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GetIntentResponse{
			IntentBase: IntentBase{IntentID: "abc", Status: StatusBaseSettled},
		})
	}))
	defer srv.Close()

	c, _ := NewPublicClient(srv.URL)
	resp, err := c.GetIntent(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if resp.IntentID != "abc" {
		t.Errorf("IntentID = %q", resp.IntentID)
	}
}

func TestPublicClient_CreateIntent_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "invalid amount"})
	}))
	defer srv.Close()

	c, _ := NewPublicClient(srv.URL)
	_, err := c.CreateIntent(context.Background(), &CreateIntentRequest{Amount: "0"})

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}
