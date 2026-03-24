package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Tier struct {
	ID        string  `json:"id"`
	CreatorID string  `json:"creator_id"`
	Price     float64 `json:"price"`
	IsActive  bool    `json:"is_active"`
}

type SubscriptionCheck struct {
	HasAccess bool    `json:"has_access"`
	TierID    *string `json:"tier_id"`
}

type Client struct {
	baseURL        string
	internalSecret string
	http           *http.Client
}

func NewClient(baseURL, internalSecret string) *Client {
	return &Client{
		baseURL:        baseURL,
		internalSecret: internalSecret,
		http:           &http.Client{},
	}
}

func (c *Client) GetTier(ctx context.Context, tierID string) (*Tier, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/internal/tiers/%s", c.baseURL, tierID), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Internal-Secret", c.internalSecret)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get tier: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile service returned %d", resp.StatusCode)
	}

	var tier Tier
	if err := json.NewDecoder(resp.Body).Decode(&tier); err != nil {
		return nil, fmt.Errorf("decode tier: %w", err)
	}
	return &tier, nil
}

func (c *Client) CheckSubscription(ctx context.Context, fanID, creatorID string) (*SubscriptionCheck, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/internal/subscriptions/check?fan_id=%s&creator_id=%s", c.baseURL, fanID, creatorID), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Internal-Secret", c.internalSecret)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("check subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile service returned %d", resp.StatusCode)
	}

	var check SubscriptionCheck
	if err := json.NewDecoder(resp.Body).Decode(&check); err != nil {
		return nil, fmt.Errorf("decode check: %w", err)
	}
	return &check, nil
}

var ErrNotFound = fmt.Errorf("not found")
