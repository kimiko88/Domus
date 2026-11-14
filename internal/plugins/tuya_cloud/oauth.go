// Package tuya_cloud implements Tuya Cloud and end-user authorization flows.
package tuya_cloud

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OAuthConfig struct {
	ClientID, Secret, Identifier, BaseURL string
	HTTPClient                            *http.Client
}
type OAuthToken struct {
	UID          string `json:"uid"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpireTime   int64  `json:"expire_time"`
}

// OAuthAPIError contains provider diagnostics safe to expose to server logs. It never
// includes request headers, signatures, authorization codes, or token values.
type OAuthAPIError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *OAuthAPIError) Error() string {
	if e == nil {
		return "tuya oauth: provider request failed"
	}
	return fmt.Sprintf("tuya oauth: provider request failed (http=%d code=%s): %s", e.HTTPStatus, e.Code, safeMessage(e.Message))
}
func providerCode(code any) string {
	if code == nil {
		return ""
	}
	return safeMessage(fmt.Sprint(code))
}

type OAuthClient struct {
	cfg  OAuthConfig
	http *http.Client
}

func NewOAuthClient(cfg OAuthConfig) (*OAuthClient, error) {
	if strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.Secret) == "" || strings.TrimSpace(cfg.Identifier) == "" {
		return nil, errors.New("tuya oauth: app authorization client ID, secret and identifier are required")
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || (u.Port() != "" && u.Port() != "443") || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("tuya oauth: base URL must be an HTTPS regional endpoint")
	}
	allowed := map[string]bool{"openapi.tuyaeu.com": true, "openapi.tuyaus.com": true, "openapi.tuyacn.com": true, "openapi.tuyain.com": true, "openapi-ueaz.tuyaus.com": true, "openapi-weaz.tuyaeu.com": true}
	if !allowed[strings.ToLower(u.Hostname())] {
		return nil, errors.New("tuya oauth: unsupported regional API host")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 12 * time.Second}
	}
	client := *cfg.HTTPClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &OAuthClient{cfg: cfg, http: &client}, nil
}
func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (OAuthToken, error) {
	if code == "" || len(code) > 2048 {
		return OAuthToken{}, errors.New("tuya oauth: invalid authorization code")
	}
	return c.tokenRequest(ctx, "/v1.0/token", url.Values{"grant_type": []string{"2"}, "code": []string{code}})
}
func (c *OAuthClient) Refresh(ctx context.Context, refresh string) (OAuthToken, error) {
	if refresh == "" || len(refresh) > 4096 || strings.ContainsAny(refresh, "/?#") {
		return OAuthToken{}, errors.New("tuya oauth: invalid refresh token")
	}
	return c.tokenRequest(ctx, "/v1.0/token/"+url.PathEscape(refresh), nil)
}
func (c *OAuthClient) tokenRequest(ctx context.Context, path string, q url.Values) (OAuthToken, error) {
	requestPath := path
	if len(q) > 0 {
		requestPath += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+requestPath, nil)
	if err != nil {
		return OAuthToken{}, err
	}
	c.signHeaders(req, requestPath, strconv.FormatInt(time.Now().UnixMilli(), 10), uuid.NewString(), "")
	resp, err := c.http.Do(req)
	if err != nil {
		return OAuthToken{}, fmt.Errorf("tuya oauth token request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if err != nil {
		return OAuthToken{}, err
	}
	if len(raw) > 2<<20 {
		return OAuthToken{}, errors.New("tuya oauth: response too large")
	}
	var env struct {
		Success bool       `json:"success"`
		Code    any        `json:"code"`
		Msg     string     `json:"msg"`
		Result  OAuthToken `json:"result"`
	}
	if err = json.Unmarshal(raw, &env); err != nil {
		if resp.StatusCode != http.StatusOK {
			return OAuthToken{}, &OAuthAPIError{HTTPStatus: resp.StatusCode, Message: "non-JSON provider error response"}
		}
		return OAuthToken{}, fmt.Errorf("tuya oauth: invalid token response")
	}
	if resp.StatusCode != http.StatusOK || !env.Success {
		return OAuthToken{}, &OAuthAPIError{HTTPStatus: resp.StatusCode, Code: providerCode(env.Code), Message: safeMessage(env.Msg)}
	}
	if env.Result.AccessToken == "" || env.Result.RefreshToken == "" || env.Result.UID == "" {
		return OAuthToken{}, &OAuthAPIError{HTTPStatus: resp.StatusCode, Code: providerCode(env.Code), Message: "provider returned an incomplete token"}
	}
	if env.Result.ExpireTime <= 0 || env.Result.ExpireTime > 86400 {
		env.Result.ExpireTime = 7200
	}
	return env.Result, nil
}
func (c *OAuthClient) Devices(ctx context.Context, token OAuthToken) ([]Device, error) {
	if token.UID == "" || token.AccessToken == "" {
		return nil, errors.New("tuya oauth: user token is incomplete")
	}
	path := "/v1.0/users/" + url.PathEscape(token.UID) + "/devices"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	c.signHeaders(req, path, strconv.FormatInt(time.Now().UnixMilli(), 10), uuid.NewString(), token.AccessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tuya oauth devices request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 2<<20 {
		return nil, errors.New("tuya oauth: response too large")
	}
	var env struct {
		Success bool     `json:"success"`
		Code    any      `json:"code"`
		Msg     string   `json:"msg"`
		Result  []Device `json:"result"`
	}
	if err = json.Unmarshal(raw, &env); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, &OAuthAPIError{HTTPStatus: resp.StatusCode, Message: "non-JSON provider error response"}
		}
		return nil, fmt.Errorf("tuya oauth: invalid devices response")
	}
	if resp.StatusCode != http.StatusOK || !env.Success {
		return nil, &OAuthAPIError{HTTPStatus: resp.StatusCode, Code: providerCode(env.Code), Message: safeMessage(env.Msg)}
	}
	if env.Result == nil {
		env.Result = []Device{}
	}
	return env.Result, nil
}

// GetDevice fetches detailed device metadata including local_key for a single device via user token.
func (c *OAuthClient) GetDevice(ctx context.Context, token OAuthToken, deviceID string) (*Device, error) {
	if token.UID == "" || token.AccessToken == "" {
		return nil, errors.New("tuya oauth: user token is incomplete")
	}
	if deviceID == "" || len(deviceID) > 128 || strings.ContainsAny(deviceID, "/?#") {
		return nil, errors.New("tuya oauth: invalid device ID")
	}
	path := "/v1.0/devices/" + url.PathEscape(deviceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	c.signHeaders(req, path, strconv.FormatInt(time.Now().UnixMilli(), 10), uuid.NewString(), token.AccessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tuya oauth device request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 2<<20 {
		return nil, errors.New("tuya oauth: response too large")
	}
	var env struct {
		Success bool    `json:"success"`
		Code    any     `json:"code"`
		Msg     string  `json:"msg"`
		Result  *Device `json:"result"`
	}
	if err = json.Unmarshal(raw, &env); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, &OAuthAPIError{HTTPStatus: resp.StatusCode, Message: "non-JSON provider error response"}
		}
		return nil, fmt.Errorf("tuya oauth: invalid device response")
	}
	if resp.StatusCode != http.StatusOK || !env.Success {
		return nil, &OAuthAPIError{HTTPStatus: resp.StatusCode, Code: providerCode(env.Code), Message: safeMessage(env.Msg)}
	}
	return env.Result, nil
}
func (c *OAuthClient) signHeaders(req *http.Request, path, ts, nonce, access string) {
	emptyHash := sha256.Sum256(nil)
	stringToSign := req.Method + "\n" + hex.EncodeToString(emptyHash[:]) + "\n\n" + path
	prefix := c.cfg.ClientID + ts + nonce + c.cfg.Identifier
	if access != "" {
		prefix = c.cfg.ClientID + access + ts + nonce + c.cfg.Identifier
		req.Header.Set("access_token", access)
	}
	mac := hmac.New(sha256.New, []byte(c.cfg.Secret))
	_, _ = mac.Write([]byte(prefix + stringToSign))
	req.Header.Set("client_id", c.cfg.ClientID)
	req.Header.Set("identifier", c.cfg.Identifier)
	req.Header.Set("t", ts)
	req.Header.Set("nonce", nonce)
	req.Header.Set("sign_method", "HMAC-SHA256")
	req.Header.Set("sign", strings.ToUpper(hex.EncodeToString(mac.Sum(nil))))
	req.Header.Set("Content-Type", "application/json")
}
