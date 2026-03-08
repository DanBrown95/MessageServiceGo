package models

// DecryptedMessageRequest is the payload encrypted and sent to ManagementPanelAPI.
// Mirrors the C# DecryptedMessageRequest model.
type DecryptedMessageRequest struct {
	Message    MessageRecord `json:"Message"`
	RequestKey string        `json:"RequestKey"`
}

// EncryptedMessageRequest is the outer envelope POSTed to /api/monitor/message.
type EncryptedMessageRequest struct {
	EncodedRequest string `json:"EncodedRequest"`
}
