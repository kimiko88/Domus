package vesync

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	apiBaseUS = "https://smartapi.vesync.com"
	apiBaseEU = "https://smartapi.vesync.eu"
	apiUserAgent = "okhttp/3.12.1"
	appVersion = "5.6.60"
	appID = "eldodkfj"
	phoneBrand = "pyvesync"
	phoneOS = "Android"
	clientType = "vesyncApp"
)

var nonEUCountryCodes = map[string]struct{}{
	"US": {}, "CA": {}, "MX": {}, "JP": {},
}

// Session is the small credential set retained by Domus. It must only be
// serialized inside the existing encrypted integration configuration.
type Session struct {
	AccountID string `json:"account_id"`
	Token string `json:"token"`
	CountryCode string `json:"country_code"`
	Region string `json:"region"`
	TimeZone string `json:"time_zone"`
}

// DeviceInfo is a sanitized representation of a VeSync device. Raw account
// credentials and provider response payloads are never exposed to the UI.
type DeviceInfo struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Model string `json:"model"`
	DeviceType string `json:"device_type"`
	ConfigModule string `json:"config_module,omitempty"`
	ProductType string `json:"product_type"`
	Status string `json:"status,omitempty"`
	ConnectionStatus string `json:"connection_status,omitempty"`
	Firmware string `json:"firmware,omitempty"`
	SupportedModes []string `json:"supported_modes,omitempty"`
	FanLevels []int `json:"fan_levels,omitempty"`
}

type APIError struct {
	Code string
	Message string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return "VeSync cloud API returned an error (code " + e.Code + ")"
	}
	return "VeSync cloud API error (code " + e.Code + "): " + e.Message
}

// Client implements the supported VeSync cloud API subset. VeSync devices
// currently require the provider cloud; this client does not claim LAN control.
type Client struct {
	httpClient *http.Client
	baseURL string
}

func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &Client{httpClient: &http.Client{Timeout: timeout}}
}

func newClientForTest(client *http.Client, baseURL string) *Client {
	return &Client{httpClient: client, baseURL: baseURL}
}

func RegionForCountry(countryCode string) string {
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	if _, ok := nonEUCountryCodes[countryCode]; ok {
		return "US"
	}
	return "EU"
}

func (c *Client) endpoint(region string) (string, error) {
	if c.baseURL != "" {
		return strings.TrimRight(c.baseURL, "/"), nil
	}
	switch strings.ToUpper(strings.TrimSpace(region)) {
	case "EU":
		return apiBaseEU, nil
	case "US":
		return apiBaseUS, nil
	default:
		return "", errors.New("vesync: unsupported cloud region")
	}
}

func traceID() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func terminalID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "2" + strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return "2" + hex.EncodeToString(raw[:])
}

func passwordDigest(password string) string {
	sum := md5.Sum([]byte(password))
	return hex.EncodeToString(sum[:])
}

func (c *Client) post(ctx context.Context, region, path string, payload map[string]any, bypass bool) (map[string]any, error) {
	base, err := c.endpoint(region)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("vesync: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("vesync: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Accept", "application/json")
	if bypass {
		req.Header.Set("User-Agent", apiUserAgent)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vesync: cloud request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("vesync: read cloud response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{Code: strconv.Itoa(resp.StatusCode), Message: "HTTP request rejected"}
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, errors.New("vesync: invalid cloud response")
	}
	if responseCode(decoded["code"]) != "0" {
		message, _ := decoded["msg"].(string)
		return nil, &APIError{Code: responseCode(decoded["code"]), Message: safeAPIMessage(message)}
	}
	return decoded, nil
}

func responseCode(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	default:
		return "unknown"
	}
}

func safeAPIMessage(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 180 {
		message = message[:180]
	}
	lower := strings.ToLower(message)
	if strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "accountid") {
		return "The cloud service rejected the request"
	}
	return message
}

func (c *Client) Authenticate(ctx context.Context, email, password, countryCode, timeZone string) (Session, error) {
	email = strings.TrimSpace(email)
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	timeZone = strings.TrimSpace(timeZone)
	if email == "" || password == "" || countryCode == "" || timeZone == "" {
		return Session{}, errors.New("vesync: email, password, country and time zone are required")
	}
	region := RegionForCountry(countryCode)
	terminal := terminalID()

	first := map[string]any{
		"email": email,
		"method": "authByPWDOrOTM",
		"password": passwordDigest(password),
		"acceptLanguage": "en",
		"accountID": "",
		"authProtocolType": "generic",
		"clientInfo": phoneBrand,
		"clientType": clientType,
		"clientVersion": "VeSync " + appVersion,
		"debugMode": false,
		"osInfo": phoneOS,
		"terminalId": terminal,
		"timeZone": timeZone,
		"token": "",
		"userCountryCode": countryCode,
		"appID": appID,
		"sourceAppID": appID,
		"traceId": traceID(),
	}
	firstResponse, err := c.post(ctx, region, "/globalPlatform/api/accountAuth/v1/authByPWDOrOTM", first, false)
	if err != nil {
		return Session{}, err
	}
	firstResult, ok := firstResponse["result"].(map[string]any)
	if !ok {
		return Session{}, errors.New("vesync: authentication response is missing its result")
	}
	accountID, _ := firstResult["accountID"].(string)
	authorizeCode, _ := firstResult["authorizeCode"].(string)
	if accountID == "" || authorizeCode == "" {
		return Session{}, errors.New("vesync: authentication response is incomplete")
	}

	second := map[string]any{
		"method": "loginByAuthorizeCode4Vesync",
		"authorizeCode": authorizeCode,
		"acceptLanguage": "en",
		"accountID": accountID,
		"clientInfo": phoneBrand,
		"clientType": clientType,
		"clientVersion": "VeSync " + appVersion,
		"debugMode": false,
		"emailSubscriptions": false,
		"osInfo": phoneOS,
		"terminalId": terminal,
		"timeZone": timeZone,
		"token": "",
		"userCountryCode": countryCode,
		"traceId": traceID(),
	}
	secondResponse, err := c.post(ctx, region, "/user/api/accountManage/v1/loginByAuthorizeCode4Vesync", second, false)
	if err != nil {
		return Session{}, err
	}
	secondResult, ok := secondResponse["result"].(map[string]any)
	if !ok {
		return Session{}, errors.New("vesync: login response is missing its result")
	}
	token, _ := secondResult["token"].(string)
	if token == "" {
		return Session{}, errors.New("vesync: login response did not contain a cloud token")
	}
	if returnedAccount, ok := secondResult["accountID"].(string); ok && returnedAccount != "" {
		accountID = returnedAccount
	}
	if returnedCountry, ok := secondResult["countryCode"].(string); ok && returnedCountry != "" {
		countryCode = strings.ToUpper(returnedCountry)
	}
	if returnedRegion, ok := secondResult["currentRegion"].(string); ok && returnedRegion != "" {
		region = strings.ToUpper(returnedRegion)
	}
	if region != "EU" && region != "US" {
		return Session{}, errors.New("vesync: login returned an unsupported region")
	}
	return Session{AccountID: accountID, Token: token, CountryCode: countryCode, Region: region, TimeZone: timeZone}, nil
}

func DeviceListRequest(session Session) map[string]any {
	return map[string]any{
		"token": session.Token,
		"accountID": session.AccountID,
		"timeZone": session.TimeZone,
		"method": "devices",
		"pageNo": 1,
		"pageSize": 100,
		"appVersion": appVersion,
		"phoneBrand": phoneBrand,
		"phoneOS": phoneOS,
		"acceptLanguage": "en",
		"traceId": traceID(),
	}
}

func (c *Client) ListDevices(ctx context.Context, session Session) ([]DeviceInfo, error) {
	if session.Token == "" || session.AccountID == "" {
		return nil, errors.New("vesync: cloud session is not configured")
	}
	response, err := c.post(ctx, session.Region, "/cloud/v1/deviceManaged/devices", DeviceListRequest(session), true)
	if err != nil {
		return nil, err
	}
	result, ok := response["result"].(map[string]any)
	if !ok {
		return nil, errors.New("vesync: device list response is invalid")
	}
	items, ok := result["list"].([]any)
	if !ok {
		return nil, errors.New("vesync: device list is missing")
	}
	devices := make([]DeviceInfo, 0, len(items))
	for _, item := range items {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		cid := stringValue(raw["cid"])
		if cid == "" {
			continue
		}
		deviceType := stringValue(raw["deviceType"])
		configModule := stringValue(raw["configModule"])
		model := knownPurifierModel(deviceType)
		if model == "" {
			model = knownPurifierModel(configModule)
		}
		if model == "" {
			model = deviceType
		}
		productType, modes, levels := modelCapabilities(model)
		if productType == "unknown" {
			productType = classifyProduct(deviceType)
		}
		if productType == "unknown" {
			productType = classifyProduct(configModule)
		}
		devices = append(devices, DeviceInfo{
			ID: cid,
			Name: fallback(stringValue(raw["deviceName"]), model),
			Model: model,
			DeviceType: deviceType,
			ConfigModule: configModule,
			ProductType: productType,
			Status: stringValue(raw["deviceStatus"]),
			ConnectionStatus: stringValue(raw["connectionStatus"]),
			Firmware: stringValue(raw["currentFirmVersion"]),
			SupportedModes: modes,
			FanLevels: levels,
		})
	}
	return devices, nil
}

func stringValue(value any) string {
	v, _ := value.(string)
	return strings.TrimSpace(v)
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func classifyProduct(model string) string {
	v := strings.ToLower(model)
	switch {
	case strings.HasPrefix(v, "esw"):
		return "outlet"
	case strings.HasPrefix(v, "esl"):
		return "bulb"
	case strings.HasPrefix(v, "caf-"), strings.HasPrefix(v, "cs"):
		return "air_fryer"
	case strings.HasPrefix(v, "luh-"), strings.Contains(v, "humidifier"):
		return "humidifier"
	case strings.Contains(v, "fan"):
		return "fan"
	case strings.HasPrefix(v, "ltm-"):
		return "thermostat"
	default:
		return "unknown"
	}
}

var purifierCapabilities = map[string]struct {
	modes []string
	levels []int
}{
	"Core200S": {modes: []string{"sleep", "manual"}, levels: []int{1, 2, 3}},
	"LAP-C201S-AUSR": {modes: []string{"sleep", "manual"}, levels: []int{1, 2, 3}},
	"LAP-C202S-WUSR": {modes: []string{"sleep", "manual"}, levels: []int{1, 2, 3}},
	"Core300S": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3}},
	"LAP-C301S-WJP": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3}},
	"LAP-C302S-WUSB": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3}},
	"LAP-C301S-WAAA": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3}},
	"LAP-C302S-WGC": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3}},
	"Core400S": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3, 4}},
	"LAP-C401S-WJP": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3, 4}},
	"LAP-C401S-WUSR": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3, 4}},
	"LAP-C401S-WAAA": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3, 4}},
	"Core600S": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3, 4}},
	"LAP-C601S-WUS": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3, 4}},
	"LAP-C601S-WUSR": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3, 4}},
	"LAP-C601S-WEU": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3, 4}},
	"LAP-V102S-AASR": {modes: []string{"sleep", "manual", "auto", "pet"}, levels: []int{1, 2, 3, 4}},
	"LAP-V102S-WUS": {modes: []string{"sleep", "manual", "auto", "pet"}, levels: []int{1, 2, 3, 4}},
	"LAP-V102S-WEU": {modes: []string{"sleep", "manual", "auto", "pet"}, levels: []int{1, 2, 3, 4}},
	"LAP-V201S-AASR": {modes: []string{"sleep", "manual", "auto", "pet"}, levels: []int{1, 2, 3, 4}},
	"LAP-V201S-WJP": {modes: []string{"sleep", "manual", "auto", "pet"}, levels: []int{1, 2, 3, 4}},
	"LAP-V201S-WEU": {modes: []string{"sleep", "manual", "auto", "pet"}, levels: []int{1, 2, 3, 4}},
	"LAP-V201S-WUS": {modes: []string{"sleep", "manual", "auto", "pet"}, levels: []int{1, 2, 3, 4}},
	"LAP-EL551S-WEU": {modes: []string{"sleep", "manual", "auto", "turbo"}, levels: []int{1, 2, 3}},
	"LAP-EL551S-WUS": {modes: []string{"sleep", "manual", "auto", "turbo"}, levels: []int{1, 2, 3}},
	"LAP-B851S-WEU": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3}},
	"LAP-B851S-WUS": {modes: []string{"sleep", "manual", "auto"}, levels: []int{1, 2, 3}},
}

func knownPurifierModel(rawModel string) string {
	rawModel = strings.TrimSpace(rawModel)
	if _, ok := purifierCapabilities[rawModel]; ok {
		return rawModel
	}
	for model := range purifierCapabilities {
		if strings.Contains(rawModel, model) {
			return model
		}
	}
	return ""
}

func modelCapabilities(model string) (string, []string, []int) {
	caps, ok := purifierCapabilities[strings.TrimSpace(model)]
	if !ok {
		return "unknown", nil, nil
	}
	return "purifier", append([]string(nil), caps.modes...), append([]int(nil), caps.levels...)
}

func BypassV2Request(session Session, device DeviceInfo, method string, data map[string]any) map[string]any {
	return map[string]any{
		"acceptLanguage": "en",
		"appVersion": appVersion,
		"phoneBrand": phoneBrand,
		"phoneOS": phoneOS,
		"accountID": session.AccountID,
		"cid": device.ID,
		"configModule": device.ConfigModule,
		"debugMode": false,
		"traceId": traceID(),
		"timeZone": session.TimeZone,
		"token": session.Token,
		"userCountryCode": session.CountryCode,
		"configModel": device.ConfigModule,
		"deviceId": device.ID,
		"method": "bypassV2",
		"payload": map[string]any{
			"method": method,
			"source": "APP",
			"data": data,
		},
	}
}

func (c *Client) callPurifier(ctx context.Context, session Session, device DeviceInfo, method string, data map[string]any) (map[string]any, error) {
	if device.ProductType != "purifier" {
		return nil, errors.New("vesync: controls are only enabled for verified purifier models")
	}
	allowed := map[string]struct{}{
		"getPurifierStatus": {},
		"setSwitch": {},
		"setPurifierMode": {},
		"setLevel": {},
	}
	if _, ok := allowed[method]; !ok {
		return nil, errors.New("vesync: unsupported purifier command")
	}
	response, err := c.post(ctx, session.Region, "/cloud/v2/deviceManaged/bypassV2", BypassV2Request(session, device, method, data), true)
	if err != nil {
		return nil, err
	}
	if code, message, failed := nestedProviderError(response); failed {
		return nil, &APIError{Code: code, Message: safeAPIMessage(message)}
	}
	return response, nil
}

func providerErrorCode(value any) (string, bool) {
	switch code := value.(type) {
	case float64:
		if code != 0 {
			return strconv.FormatInt(int64(code), 10), true
		}
	case int:
		if code != 0 {
			return strconv.Itoa(code), true
		}
	case json.Number:
		if parsed, err := code.Float64(); err == nil && parsed != 0 {
			return code.String(), true
		}
	}
	return "", false
}

func nestedProviderError(value any) (string, string, bool) {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			lower := strings.ToLower(key)
			if lower == "code" || lower == "error" || lower == "error_code" || lower == "errorcode" || lower == "device_error_code" {
				if code, failed := providerErrorCode(child); failed {
					message, _ := current["msg"].(string)
					return code, message, true
				}
			}
			if code, message, failed := nestedProviderError(child); failed {
				return code, message, true
			}
		}
	case []any:
		for _, child := range current {
			if code, message, failed := nestedProviderError(child); failed {
				return code, message, true
			}
		}
	}
	return "", "", false
}

func (c *Client) GetPurifierStatus(ctx context.Context, session Session, device DeviceInfo) (map[string]any, error) {
	return c.callPurifier(ctx, session, device, "getPurifierStatus", map[string]any{})
}

func (c *Client) SetPurifierPower(ctx context.Context, session Session, device DeviceInfo, enabled bool) (map[string]any, error) {
	return c.callPurifier(ctx, session, device, "setSwitch", map[string]any{"enabled": enabled, "id": 0})
}

func (c *Client) SetPurifierMode(ctx context.Context, session Session, device DeviceInfo, mode string) (map[string]any, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if !containsString(device.SupportedModes, mode) {
		return nil, errors.New("vesync: unsupported mode for this purifier model")
	}
	if mode == "manual" {
		if len(device.FanLevels) == 0 {
			return nil, errors.New("vesync: manual mode requires a known fan-level profile")
		}
		return c.SetPurifierFanLevel(ctx, session, device, device.FanLevels[0])
	}
	return c.callPurifier(ctx, session, device, "setPurifierMode", map[string]any{"mode": mode})
}

func (c *Client) SetPurifierFanLevel(ctx context.Context, session Session, device DeviceInfo, level int) (map[string]any, error) {
	if !containsInt(device.FanLevels, level) {
		return nil, errors.New("vesync: unsupported fan level for this purifier model")
	}
	return c.callPurifier(ctx, session, device, "setLevel", map[string]any{"id": 0, "level": level, "type": "wind"})
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target { return true }
	}
	return false
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target { return true }
	}
	return false
}
