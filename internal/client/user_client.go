package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type UserClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewUserClient(baseURL string) *UserClient {
	return &UserClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *UserClient) UserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	url := fmt.Sprintf("%s/users/%s", c.baseURL, userID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("unexpected status %d from user service", resp.StatusCode)
}

func (c *UserClient) ValidateUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error) {
	var invalidIDs []uuid.UUID
	for _, id := range userIDs {
		exists, err := c.UserExists(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to validate user %s: %w", id, err)
		}
		if !exists {
			invalidIDs = append(invalidIDs, id)
		}
	}
	return invalidIDs, nil
}
