package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// NewHTTPBroadcaster returns a function that POSTs an encrypted message payload
// to ManagementPanelAPI's /api/monitor/message endpoint.
func NewHTTPBroadcaster(monitorAPIAddress string) func(ctx context.Context, payload string) error {
	client := &http.Client{}
	url := monitorAPIAddress + "/api/monitor/message"

	return func(ctx context.Context, payload string) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(payload))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send message broadcast: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("message broadcast returned status %d: %s", resp.StatusCode, string(body))
		}
		return nil
	}
}
