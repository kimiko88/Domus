// Package xiaomi_home implements Xiaomi Open Platform OAuth and the documented
// MIoT cloud control API. It does not use Home Assistant's private integration API.
package xiaomi_home

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	xiaomiOAuthBase = "https://account.xiaomi.com"
	xiaomiProfileURL = "https://open.account.xiaomi.com/user/profile"
	xiaomiSpecBase = "https://miot-spec.org/miot-spec-v2"
	xiaomiMaxResponse = 2 << 20
)

var (
	xiaomiClientIDPattern = regexp.MustCompile(`^[0-9]{10,24}$`)
	xiaomiRegionHosts = map[string]string{
		"eu": "https://de.api.home.mi.com",
		"de": "https://de.api.home.mi.com",
		"us": "https://us.api.home.mi.com",
		"sg": "https://sg.api.home.mi.com",
	}
)

type CloudConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Region       string
}

type CloudClient struct {
	config CloudConfig
	apiBase string
	http *http.Client
}

type CloudToken struct {
	AccessToken string    `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	MacKey string         `json:"mac_key,omitempty"`
	MacAlgorithm string   `json:"mac_algorithm,omitempty"`
	ExpiresAt time.Time   `json:"expires_at,omitempty"`
	ExpiresIn int64       `json:"expires_in,omitempty"`
}

type CloudUser struct {
	UserID string `json:"user_id"`
	Nickname string `json:"nickname,omitempty"`
}

type CloudDevice struct {
	DID string `json:"did"`
	Name string `json:"name"`
	Online bool `json:"online"`
	Type string `json:"type"`
	Category string `json:"category,omitempty"`
	RoomID string `json:"rid,omitempty"`
	CloudID string `json:"cloud_id,omitempty"`
	LastUpdate int64 `json:"last_update_timestamp,omitempty"`
	AliasName map[string][]string `json:"alias_name,omitempty"`
}

type CloudHome struct {
	ID int64 `json:"id"`
	Name string `json:"name"`
	Rooms []CloudRoom `json:"rooms,omitempty"`
}

type CloudRoom struct {
	ID string `json:"id"`
	Name string `json:"name"`
}

type PropertyValue struct {
	PID string `json:"pid"`
	Value any `json:"value,omitempty"`
	Status int `json:"status,omitempty"`
	Description string `json:"description,omitempty"`
}

type PropertyResponse struct {
	Properties []PropertyValue `json:"properties"`
}

type PropertyUpdate struct {
	PID string `json:"pid"`
	Value any `json:"value"`
}

type ActionResponse struct {
	OID string `json:"oid,omitempty"`
	Out []any `json:"out,omitempty"`
}

type subscriptionItem struct {
	Topic string `json:"topic"`
	Properties []string `json:"properties,omitempty"`
	Events []string `json:"events,omitempty"`
	Devices []string `json:"devices,omitempty"`
	Type string `json:"type"`
	ReceiverURL string `json:"receiver-url"`
	CustomData map[string]any `json:"custom-data,omitempty"`
}

type xiaomiAPIError struct {
	Code any `json:"code"`
	Description string `json:"description"`
	Result string `json:"result"`
}

func NewCloudClient(cfg CloudConfig, client *http.Client) (*CloudClient, error) {
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.ClientSecret = strings.TrimSpace(cfg.ClientSecret)
	cfg.RedirectURL = strings.TrimSpace(cfg.RedirectURL)
	cfg.Region = strings.ToLower(strings.TrimSpace(cfg.Region))
	host, ok := xiaomiRegionHosts[cfg.Region]
	if !ok {
		return nil, errors.New("xiaomi: unsupported Xiaomi Open Platform region")
	}
	if !xiaomiClientIDPattern.MatchString(cfg.ClientID) || cfg.ClientSecret == "" || len(cfg.ClientSecret) > 512 {
		return nil, errors.New("xiaomi: invalid Open Platform application credentials")
	}
	if err := validateHTTPSURL(cfg.RedirectURL); err != nil {
		return nil, fmt.Errorf("xiaomi: invalid OAuth redirect URL: %w", err)
	}
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}
	safe := *client
	safe.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	if safe.Timeout <= 0 || safe.Timeout > 20*time.Second {
		safe.Timeout = 12 * time.Second
	}
	return &CloudClient{config: cfg, apiBase: host, http: &safe}, nil
}

func (c *CloudClient) AuthorizationURL(state string, scopes []string) (string, error) {
	if len(state) < 32 || len(state) > 128 || strings.ContainsAny(state, "\r\n\t ") {
		return "", errors.New("xiaomi: invalid OAuth state")
	}
	query := url.Values{
		"client_id": {c.config.ClientID},
		"redirect_uri": {c.config.RedirectURL},
		"response_type": {"code"},
		"state": {state},
	}
	if len(scopes) > 0 {
		for _, scope := range scopes {
			if scope == "" || len(scope) > 64 || strings.ContainsAny(scope, "\r\n\t ") {
				return "", errors.New("xiaomi: invalid OAuth scope")
			}
		}
		query.Set("scope", strings.Join(scopes, " "))
	} else {
		query.Set("scope", "1")
	}
	return xiaomiOAuthBase + "/oauth2/authorize?" + query.Encode(), nil
}

func (c *CloudClient) ExchangeCode(ctx context.Context, code string) (*CloudToken, error) {
	code = strings.TrimSpace(code)
	if code == "" || len(code) > 2048 || strings.ContainsAny(code, "\r\n") {
		return nil, errors.New("xiaomi: invalid OAuth authorization code")
	}
	return c.tokenRequest(ctx, url.Values{
		"pt": {"0"},
		"client_id": {c.config.ClientID},
		"client_secret": {c.config.ClientSecret},
		"redirect_uri": {c.config.RedirectURL},
		"grant_type": {"authorization_code"},
		"code": {code},
	})
}

func (c *CloudClient) RefreshToken(ctx context.Context, refreshToken string) (*CloudToken, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" || len(refreshToken) > 8192 {
		return nil, errors.New("xiaomi: invalid OAuth refresh token")
	}
	return c.tokenRequest(ctx, url.Values{
		"pt": {"0"},
		"client_id": {c.config.ClientID},
		"client_secret": {c.config.ClientSecret},
		"redirect_uri": {c.config.RedirectURL},
		"grant_type": {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

func (c *CloudClient) tokenRequest(ctx context.Context, form url.Values) (*CloudToken, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xiaomiOAuthBase+"/oauth2/token?"+form.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("xiaomi: create token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xiaomi: OAuth token request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, errors.New("xiaomi: could not read OAuth token response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("xiaomi: OAuth token endpoint returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		AccessToken string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		MacKey string `json:"mac_key"`
		MacAlgorithm string `json:"mac_algorithm"`
		ExpiresIn int64 `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.New("xiaomi: invalid OAuth token response")
	}
	if result.AccessToken == "" || len(result.AccessToken) > 8192 || result.RefreshToken == "" || len(result.RefreshToken) > 8192 {
		return nil, errors.New("xiaomi: OAuth response omitted required tokens")
	}
	if result.MacAlgorithm != "" && !strings.EqualFold(result.MacAlgorithm, "HmacSHA1") {
		return nil, errors.New("xiaomi: unsupported Xiaomi OAuth MAC algorithm")
	}
	token := &CloudToken{
		AccessToken: result.AccessToken, RefreshToken: result.RefreshToken,
		MacKey: result.MacKey, MacAlgorithm: result.MacAlgorithm,
		ExpiresIn: result.ExpiresIn,
	}
	if result.ExpiresIn > 0 && result.ExpiresIn <= 31536000 {
		token.ExpiresAt = time.Now().UTC().Add(time.Duration(result.ExpiresIn) * time.Second)
	}
	return token, nil
}

func (c *CloudClient) GetUserProfile(ctx context.Context, accessToken string) (*CloudUser, error) {
	values := url.Values{"clientId": {c.config.ClientID}, "token": {accessToken}}
	var result struct {
		Result string `json:"result"`
		Code any `json:"code"`
		Data struct {
			UserID any `json:"userId"`
			Nick string `json:"miliaoNick"`
		} `json:"data"`
	}
	if err := c.oauthGET(ctx, xiaomiProfileURL, values, &result); err != nil {
		return nil, err
	}
	if result.Result != "ok" {
		return nil, errors.New("xiaomi: Xiaomi profile API rejected the access token")
	}
	userID := strings.TrimSpace(fmt.Sprint(result.Data.UserID))
	if userID == "" || userID == "<nil>" {
		return nil, errors.New("xiaomi: Xiaomi profile response did not include a user ID")
	}
	return &CloudUser{UserID: userID, Nickname: result.Data.Nick}, nil
}

func (c *CloudClient) oauthGET(ctx context.Context, endpoint string, query url.Values, target any) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Host != "open.account.xiaomi.com" {
		return errors.New("xiaomi: invalid Xiaomi OAuth endpoint")
	}
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil { return err }
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil { return fmt.Errorf("xiaomi: account profile request failed: %w", err) }
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil { return errors.New("xiaomi: could not read account profile") }
	if resp.StatusCode < 200 || resp.StatusCode >= 300 { return fmt.Errorf("xiaomi: account profile returned HTTP %d", resp.StatusCode) }
	if err := json.Unmarshal(body, target); err != nil { return errors.New("xiaomi: invalid account profile response") }
	return nil
}

func (c *CloudClient) ListDevices(ctx context.Context, accessToken, userID string) ([]CloudDevice, error) {
	var response struct { Devices []CloudDevice `json:"devices"` }
	if err := c.apiRequest(ctx, http.MethodGet, "/api/v1/devices?compact=true", accessToken, userID, nil, &response); err != nil {
		return nil, err
	}
	if response.Devices == nil { response.Devices = []CloudDevice{} }
	return response.Devices, nil
}

func (c *CloudClient) ListHomes(ctx context.Context, accessToken, userID string) ([]CloudHome, error) {
	var response struct { Homes []CloudHome `json:"homes"` }
	if err := c.apiRequest(ctx, http.MethodGet, "/api/v1/homes", accessToken, userID, nil, &response); err != nil {
		return nil, err
	}
	if response.Homes == nil { response.Homes = []CloudHome{} }
	return response.Homes, nil
}

func (c *CloudClient) GetProperties(ctx context.Context, accessToken, userID string, pids []string) ([]PropertyValue, error) {
	if len(pids) == 0 || len(pids) > 100 {
		return nil, errors.New("xiaomi: property query must contain 1..100 property IDs")
	}
	for _, pid := range pids { if !validMIoTPID(pid) { return nil, errors.New("xiaomi: invalid MIoT property ID") } }
	query := url.Values{"pid": {strings.Join(pids, ",")}}
	var response PropertyResponse
	if err := c.apiRequest(ctx, http.MethodGet, "/api/v1/properties?"+query.Encode(), accessToken, userID, nil, &response); err != nil {
		return nil, err
	}
	return response.Properties, nil
}

func (c *CloudClient) SetProperties(ctx context.Context, accessToken, userID string, values []PropertyUpdate) ([]PropertyValue, error) {
	if len(values) == 0 || len(values) > 100 { return nil, errors.New("xiaomi: property write must contain 1..100 values") }
	for _, item := range values { if !validMIoTPID(item.PID) { return nil, errors.New("xiaomi: invalid MIoT property ID") } }
	payload := map[string]any{"properties": values}
	var response PropertyResponse
	if err := c.apiRequest(ctx, http.MethodPut, "/api/v1/properties", accessToken, userID, payload, &response); err != nil {
		return nil, err
	}
	return response.Properties, nil
}

func (c *CloudClient) InvokeAction(ctx context.Context, accessToken, userID, aid string, inputs []any) (*ActionResponse, error) {
	if !validMIoTAID(aid) || len(inputs) > 32 {
		return nil, errors.New("xiaomi: invalid MIoT action ID or inputs")
	}
	payload := map[string]any{"aid": aid, "in": inputs}
	var response ActionResponse
	if err := c.apiRequest(ctx, http.MethodPut, "/api/v1/action", accessToken, userID, payload, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *CloudClient) Subscribe(ctx context.Context, accessToken, userID, callbackURL, topic string, ids []string, custom map[string]any) (json.RawMessage, error) {
	if err := validateHTTPSURL(callbackURL); err != nil { return nil, errors.New("xiaomi: subscription callback must be HTTPS") }
	payload := map[string]any{"topic": topic, "type": "http", "receiver-url": callbackURL, "custom-data": custom}
	switch topic {
	case "properties-changed":
		if len(ids) == 0 || len(ids) > 100 { return nil, errors.New("xiaomi: property subscription must include 1..100 IDs") }
		for _, id := range ids { if !validMIoTPID(id) { return nil, errors.New("xiaomi: invalid property subscription ID") } }
		payload["properties"] = ids
	case "events-occured":
		if len(ids) == 0 || len(ids) > 100 { return nil, errors.New("xiaomi: event subscription must include 1..100 IDs") }
		for _, id := range ids { if !validMIoTEID(id) { return nil, errors.New("xiaomi: invalid event subscription ID") } }
		payload["events"] = ids
	case "devices-status-changed":
		if len(ids) == 0 || len(ids) > 100 { return nil, errors.New("xiaomi: device subscription must include 1..100 IDs") }
		for _, id := range ids { if !validMIoTDID(id) { return nil, errors.New("xiaomi: invalid device subscription ID") } }
		payload["devices"] = ids
	default:
		return nil, errors.New("xiaomi: unsupported subscription topic")
	}
	var result json.RawMessage
	if err := c.apiRequest(ctx, http.MethodPost, "/api/v1/subscriptions", accessToken, userID, payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *CloudClient) apiRequest(ctx context.Context, method, path, accessToken, userID string, payload any, target any) error {
	if accessToken == "" || len(accessToken) > 8192 || userID == "" || len(userID) > 128 {
		return errors.New("xiaomi: cloud access token or user ID is invalid")
	}
	if !strings.HasPrefix(path, "/api/v1/") || strings.ContainsAny(path, "\r\n") || strings.Contains(path, "..") {
		return errors.New("xiaomi: invalid cloud API path")
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil || len(encoded) > 128*1024 { return errors.New("xiaomi: invalid or oversized cloud API request") }
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.apiBase+path, body)
	if err != nil { return fmt.Errorf("xiaomi: create cloud API request: %w", err) }
	req.Header.Set("Accept", "application/json")
	req.Header.Set("App-Id", c.config.ClientID)
	req.Header.Set("Access-Token", accessToken)
	req.Header.Set("User-ID", userID)
	req.Header.Set("Spec-NS", "miot-spec-v2")
	if payload != nil { req.Header.Set("Content-Type", "application/json") }
	resp, err := c.http.Do(req)
	if err != nil { return fmt.Errorf("xiaomi: cloud API request failed: %w", err) }
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, xiaomiMaxResponse))
	if err != nil { return errors.New("xiaomi: could not read cloud API response") }
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("xiaomi: cloud API returned HTTP %d", resp.StatusCode)
	}
	if target == nil { return nil }
	if err := json.Unmarshal(data, target); err != nil {
		return errors.New("xiaomi: invalid cloud API response")
	}
	return nil
}

func (c *CloudClient) FetchDeviceSpec(ctx context.Context, deviceType string) (*MIoTDeviceSpec, error) {
	if !validMIoTTypeURN(deviceType) { return nil, errors.New("xiaomi: invalid MIoT device type URN") }
	query := url.Values{"type": {deviceType}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xiaomiSpecBase+"/instance?"+query.Encode(), nil)
	if err != nil { return nil, err }
	resp, err := c.http.Do(req)
	if err != nil { return nil, fmt.Errorf("xiaomi: MIoT-Spec request failed: %w", err) }
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, xiaomiMaxResponse))
	if err != nil { return nil, errors.New("xiaomi: could not read MIoT-Spec response") }
	if resp.StatusCode < 200 || resp.StatusCode >= 300 { return nil, fmt.Errorf("xiaomi: MIoT-Spec returned HTTP %d", resp.StatusCode) }
	var spec MIoTDeviceSpec
	if err := json.Unmarshal(data, &spec); err != nil { return nil, errors.New("xiaomi: invalid MIoT-Spec response") }
	if spec.Type == "" { spec.Type = deviceType }
	if spec.Type != deviceType { return nil, errors.New("xiaomi: MIoT-Spec response type did not match request") }
	return &spec, nil
}

func validMIoTTypeURN(value string) bool {
	return strings.HasPrefix(value, "urn:miot-spec-v2:device:") && len(value) <= 256 &&
		!strings.ContainsAny(value, "\r\n\t /?#")
}
func validMIoTDID(value string) bool {
	if value == "" || len(value) > 128 { return false }
	for _, ch := range value { if !(ch >= 'a' && ch <= 'z') && !(ch >= 'A' && ch <= 'Z') && !(ch >= '0' && ch <= '9') && ch != '-' && ch != '_' { return false } }
	return true
}
func validMIoTPID(value string) bool { return validMIoTTripleID(value) }
func validMIoTAID(value string) bool { return validMIoTTripleID(value) }
func validMIoTEID(value string) bool { return validMIoTTripleID(value) }
func validMIoTTripleID(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 3 || !validMIoTDID(parts[0]) { return false }
	for _, part := range parts[1:] { if n, err := strconv.Atoi(part); err != nil || n < 1 || n > 65535 { return false } }
	return true
}
func validateHTTPSURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return errors.New("must be an absolute HTTPS URL without credentials or fragment")
	}
	return nil
}

// GenerateCloudState returns a cryptographically random OAuth state parameter.
func GenerateCloudState() (string, error) {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil { return "", err }
	return hex.EncodeToString(token[:]), nil
}

// CompareCloudState performs constant-time comparison of OAuth state values.
func CompareCloudState(expected, got string) bool {
	return len(expected) == len(got) && len(expected) >= 32 &&
		subtle.ConstantTimeCompare([]byte(expected), []byte(got)) == 1
}
