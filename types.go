package pay

// Intent status constants returned by the API.
const (
	StatusAwaitingPayment    = "AWAITING_PAYMENT"
	StatusPending            = "PENDING"
	StatusVerificationFailed = "VERIFICATION_FAILED"
	StatusSourceSettled      = "SOURCE_SETTLED"
	StatusBaseSettling       = "BASE_SETTLING"
	StatusBaseSettled        = "BASE_SETTLED"
	StatusExpired            = "EXPIRED"
)

// CreateIntentRequest is the body for POST /v2/intents.
// Exactly one of Email or Recipient must be set.
type CreateIntentRequest struct {
	Email      string `json:"email,omitempty"`
	Recipient  string `json:"recipient,omitempty"`
	Amount     string `json:"amount"`
	PayerChain string `json:"payer_chain"`
}

// FeeBreakdown holds fee details from the API.
type FeeBreakdown struct {
	SourceChain           string `json:"source_chain"`
	SourceChainFee        string `json:"source_chain_fee"`
	TargetChain           string `json:"target_chain"`
	TargetChainFee        string `json:"target_chain_fee"`
	PlatformFee           string `json:"platform_fee"`
	PlatformFeePercentage string `json:"platform_fee_percentage"`
	TotalFee              string `json:"total_fee"`
}

// PaymentRequirements is used by the client to sign X402 authorization.
type PaymentRequirements struct {
	Scheme            string         `json:"scheme"`
	Network           string         `json:"network"`
	Amount            string         `json:"amount"`
	PayTo             string         `json:"payTo"`
	MaxTimeoutSeconds int            `json:"maxTimeoutSeconds"`
	Asset             string         `json:"asset"`
	Extra             map[string]any `json:"extra,omitempty"`
}

// IntentBase contains the fields shared across all intent response types.
type IntentBase struct {
	IntentID          string       `json:"intent_id"`
	MerchantRecipient string       `json:"merchant_recipient"`
	SendingAmount     string       `json:"sending_amount"`
	ReceivingAmount   string       `json:"receiving_amount"`
	EstimatedFee      string       `json:"estimated_fee"`
	FeeBreakdown      FeeBreakdown `json:"fee_breakdown"`
	Status            string       `json:"status"`
	CreatedAt         string       `json:"created_at"`
	ExpiresAt         string       `json:"expires_at"`
}

// CreateIntentResponse is the response for POST /v2/intents (201).
type CreateIntentResponse struct {
	IntentBase
	Email               string              `json:"email,omitempty"`
	SourceRecipient     string              `json:"source_recipient,omitempty"`
	PayerChain          string              `json:"payer_chain"`
	PaymentRequirements PaymentRequirements `json:"payment_requirements"`
}

// ExecuteIntentResponse is the response for POST /v2/intents/{intent_id}/execute (200).
// Backend signs with the Agent wallet and transfers USDC on Base; no settle_proof required.
type ExecuteIntentResponse struct {
	IntentBase
}

// SubmitProofResponse is the response for POST /api/intents/{intent_id} (200).
// Same structure as ExecuteIntentResponse; returned when settle_proof is submitted.
type SubmitProofResponse = ExecuteIntentResponse

// SourcePayment holds source-chain payment details from GetIntent.
type SourcePayment struct {
	Chain       string `json:"chain"`
	TxHash      string `json:"tx_hash"`
	SettleProof string `json:"settle_proof"`
	SettledAt   string `json:"settled_at"`
	ExplorerURL string `json:"explorer_url"`
}

// BasePayment holds Base-chain payment details from GetIntent.
type BasePayment struct {
	TxHash      string `json:"tx_hash"`
	SettleProof string `json:"settle_proof"`
	SettledAt   string `json:"settled_at"`
	ExplorerURL string `json:"explorer_url"`
}

// GetIntentResponse is the response for GET /v2/intents?intent_id=... (200).
type GetIntentResponse struct {
	IntentBase
	PayerChain    string         `json:"payer_chain"`
	ReceiverEmail string         `json:"receiver_email,omitempty"`
	PayerWallet   string         `json:"payer_wallet,omitempty"`
	ErrorMessage  string         `json:"error_message,omitempty"`
	CompletedAt   string         `json:"completed_at,omitempty"`
	SourcePayment *SourcePayment `json:"source_payment,omitempty"`
	BasePayment   *BasePayment   `json:"base_payment,omitempty"`
}

// ErrorResponse is the common error body from the API.
type ErrorResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}
