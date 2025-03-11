package pihole

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

type EnableAdBlockResponse struct {
	Status string
}

// ToEnableAdBlock turns am EnableAdBlockResponse into an EnableAdBlock object
func (eb EnableAdBlockResponse) ToEnableAdBlock() *EnableAdBlock {
	return &EnableAdBlock{
		Enabled: eb.Status == "enabled",
	}
}

type EnableAdBlock struct {
	Enabled bool
}

// GetAdBlockerStatus returns whether pihole ad blocking is enabled or not
func (c Client) GetAdBlockerStatus(ctx context.Context) (*EnableAdBlock, error) {
	if c.tokenClient != nil {
		return nil, fmt.Errorf("%w: set ad blocker status", ErrNotImplementedTokenClient)
	}

	req, err := c.RequestWithSession2(ctx, "GET", "/api/dns/blocking", map[string]any{})
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to retrieve current status, got status code %d", res.StatusCode)
	}

	defer res.Body.Close()
	type Response struct {
		Blocking string `json:"blocking"`
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var response Response
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, err
	}

	var enabled bool
	switch response.Blocking {
	case "disabled":
		enabled = false
	case "enabled":
		enabled = true
	default:
		return nil, fmt.Errorf("got unexpected value, blocking=%s", response.Blocking)
	}
	if response.Blocking == "disabled" {
		enabled = false
	}

	return &EnableAdBlock{Enabled: enabled}, nil
}

// SetAdBlockEnabled sets whether pihole ad blocking is enabled or not
func (c Client) SetAdBlockEnabled(ctx context.Context, enable bool) (*EnableAdBlock, error) {
	if c.tokenClient != nil {
		return nil, fmt.Errorf("%w: set ad blocker status", ErrNotImplementedTokenClient)
	}

	req, err := c.RequestWithSession2(ctx, "POST", "/api/dns/blocking", map[string]any{
		"blocking": enable,
	})
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to enable/disable blocking, got status code %d", res.StatusCode)
	}

	defer res.Body.Close()
	type Response struct {
		Blocking string `json:"blocking"`
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var response Response
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, err
	}

	var enabled bool
	switch response.Blocking {
	case "disabled":
		enabled = false
	case "enabled":
		enabled = true
	default:
		return nil, fmt.Errorf("got unexpected value, blocking=%s", response.Blocking)
	}

	return &EnableAdBlock{Enabled: enabled}, nil
}
