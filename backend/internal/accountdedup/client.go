package accountdedup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const defaultPageSize = 1000

type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

func NewClient(baseURL, accessToken string, httpClient *http.Client) *Client {
	client := httpClient
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{
		baseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		accessToken: strings.TrimSpace(accessToken),
		httpClient:  client,
	}
}

func (c *Client) Login(ctx context.Context, email, password string) error {
	payload := map[string]string{
		"email":    strings.TrimSpace(email),
		"password": password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL("/api/v1/auth/login"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	var envelope responseEnvelope[loginResponse]
	if err := c.doJSON(req, &envelope); err != nil {
		return err
	}
	if envelope.Code != 0 {
		return fmt.Errorf("login failed: %s", strings.TrimSpace(envelope.Message))
	}
	if envelope.Data.Requires2FA {
		return fmt.Errorf("login requires 2fa and is not supported by this command")
	}
	token := strings.TrimSpace(envelope.Data.AccessToken)
	if token == "" {
		return fmt.Errorf("login response missing access_token")
	}
	c.accessToken = token
	return nil
}

func (c *Client) ListAccounts(ctx context.Context) ([]Account, error) {
	accounts := make([]Account, 0)
	page := 1

	for {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("page_size", strconv.Itoa(defaultPageSize))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL("/api/v1/admin/accounts?"+query.Encode()), nil)
		if err != nil {
			return nil, err
		}
		c.attachAuth(req)

		var envelope responseEnvelope[paginatedAccounts]
		if err := c.doJSON(req, &envelope); err != nil {
			return nil, err
		}
		if envelope.Code != 0 {
			return nil, fmt.Errorf("list accounts failed: %s", strings.TrimSpace(envelope.Message))
		}

		accounts = append(accounts, envelope.Data.Items...)
		if envelope.Data.Pages <= page || len(envelope.Data.Items) == 0 {
			break
		}
		page++
	}

	return accounts, nil
}

func (c *Client) DeleteAccount(ctx context.Context, id int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.apiURL("/api/v1/admin/accounts/"+strconv.FormatInt(id, 10)), nil)
	if err != nil {
		return err
	}
	c.attachAuth(req)

	var envelope responseEnvelope[map[string]any]
	if err := c.doJSON(req, &envelope); err != nil {
		return err
	}
	if envelope.Code != 0 {
		return fmt.Errorf("delete account %d failed: %s", id, strings.TrimSpace(envelope.Message))
	}
	return nil
}

func (c *Client) apiURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.baseURL + path
}

func (c *Client) attachAuth(req *http.Request) {
	if strings.TrimSpace(c.accessToken) == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
}

func (c *Client) doJSON(req *http.Request, target any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return err
	}
	return nil
}

type responseEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	Requires2FA bool   `json:"requires_2fa"`
}

type paginatedAccounts struct {
	Items    []Account `json:"items"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Pages    int       `json:"pages"`
}
