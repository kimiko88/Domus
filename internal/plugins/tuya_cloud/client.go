// Package tuya_cloud implements the official Tuya IoT Cloud OpenAPI.
package tuya_cloud

import (
	"bytes"
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
	"unicode"
	"sync"
	"time"
)

const maxResponseBytes = 2 << 20

type Config struct {
	ClientID string
	Secret string
	BaseURL string
	Schema string
	HTTPClient *http.Client
}
type Client struct {
	cfg Config
	http *http.Client
	mu sync.Mutex
	token string
	tokenExpiry time.Time
}
type Device struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	LocalKey    string   `json:"local_key"`
	IP          string   `json:"ip"`
	ProductID   string   `json:"product_id"`
	ProductName string   `json:"product_name"`
	Category    string   `json:"category"`
	Online      bool     `json:"online"`
	Status      []Status `json:"status"`
}
type Status struct { Code string `json:"code"`; Value any `json:"value"` }
type Command struct { Code string `json:"code"`; Value any `json:"value"` }
type Function struct { Code string `json:"code"`; Name string `json:"name"`; Type string `json:"type"`; Values string `json:"values"` }
type envelope struct {
	Success bool `json:"success"`
	Code any `json:"code"`
	Msg string `json:"msg"`
	Result json.RawMessage `json:"result"`
}
type tokenResult struct { AccessToken string `json:"access_token"`; ExpireTime int64 `json:"expire_time"` }

func New(cfg Config) (*Client,error) {
	if strings.TrimSpace(cfg.ClientID)=="" || strings.TrimSpace(cfg.Secret)=="" { return nil,errors.New("tuya cloud: client ID and secret are required") }
	u,err:=url.Parse(cfg.BaseURL)
	if err!=nil || u.Scheme!="https" || u.Host=="" || u.User!=nil || (u.Port()!="" && u.Port()!="443") || (u.Path!="" && u.Path!="/") || u.RawQuery!="" || u.Fragment!="" { return nil,errors.New("tuya cloud: base URL must be an HTTPS regional endpoint") }
	allowed:=map[string]bool{"openapi.tuyaeu.com":true,"openapi.tuyaus.com":true,"openapi.tuyacn.com":true,"openapi.tuyain.com":true,"openapi-ueaz.tuyaus.com":true,"openapi-weaz.tuyaeu.com":true}
	if !allowed[strings.ToLower(u.Hostname())] { return nil,errors.New("tuya cloud: unsupported regional API host") }
	if cfg.HTTPClient==nil { cfg.HTTPClient=&http.Client{Timeout:12*time.Second} }
	// Never forward Tuya signing headers or access tokens to redirect targets.
	httpClient:=*cfg.HTTPClient
	httpClient.CheckRedirect=func(_ *http.Request,_ []*http.Request) error { return http.ErrUseLastResponse }
	cfg.BaseURL=strings.TrimRight(cfg.BaseURL,"/")
	return &Client{cfg:cfg,http:&httpClient},nil
}

func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	const pageSize = 100
	const maxPages = 20
	devices := make([]Device, 0)
	var total int64
	for page := 1; page <= maxPages; page++ {
		q := url.Values{"page_no": []string{strconv.Itoa(page)}, "page_size": []string{strconv.Itoa(pageSize)}}
		if c.cfg.Schema != "" { q.Set("schema", c.cfg.Schema) }
		var result struct {
			Devices []Device `json:"devices"`
			Total int64 `json:"total"`
		}
		if err := c.do(ctx, http.MethodGet, "/v1.0/devices", q, nil, &result); err != nil { return nil, err }
		devices = append(devices, result.Devices...)
		if result.Total > 0 { total = result.Total }
		if len(result.Devices) < pageSize || (total > 0 && int64(len(devices)) >= total) { return devices, nil }
	}
	if total > int64(len(devices)) { return nil, errors.New("tuya cloud: device count exceeds the safe 2,000-device listing limit") }
	return devices, nil
}

// GetDevice retrieves full details for a single device, including its LAN local_key.
func (c *Client) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	if deviceID == "" || len(deviceID) > 128 || strings.ContainsAny(deviceID, "/?#") {
		return nil, errors.New("tuya cloud: invalid device ID")
	}
	var dev Device
	path := "/v1.0/devices/" + url.PathEscape(deviceID)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &dev); err != nil {
		return nil, err
	}
	return &dev, nil
}
func (c *Client) Functions(ctx context.Context, deviceID string) ([]Function, error) {
	if deviceID == "" || len(deviceID) > 128 || strings.ContainsAny(deviceID, "/?#") {
		return nil, errors.New("tuya cloud: invalid device ID")
	}
	var result struct { Functions []Function `json:"functions"` }
	path := "/v1.0/iot-03/devices/" + url.PathEscape(deviceID) + "/functions"
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &result); err != nil { return nil, err }
	if result.Functions == nil { result.Functions = []Function{} }
	return result.Functions, nil
}

func (c *Client) SendCommands(ctx context.Context,deviceID string,commands []Command) error {
	if deviceID=="" || len(deviceID)>128 || strings.ContainsAny(deviceID,"/?#") { return errors.New("tuya cloud: invalid device ID") }
	if len(commands)==0 || len(commands)>32 { return errors.New("tuya cloud: between 1 and 32 commands are required") }
	for _,cmd:=range commands { if cmd.Code=="" || len(cmd.Code)>64 || strings.ContainsAny(cmd.Code,"/?#") { return errors.New("tuya cloud: invalid command code") } }
	functions,err:=c.Functions(ctx,deviceID); if err!=nil{return err}
	allowed:=make(map[string]struct{},len(functions)); for _,fn:=range functions { allowed[fn.Code]=struct{}{} }
	for _,cmd:=range commands { if _,ok:=allowed[cmd.Code]; !ok { return fmt.Errorf("tuya cloud: datapoint %q is not a writable device function",cmd.Code) } }
	body:=struct { Commands []Command `json:"commands"` }{commands}
	return c.do(ctx,http.MethodPost,"/v1.0/iot-03/devices/"+url.PathEscape(deviceID)+"/commands",nil,body,nil)
}
func (c *Client) do(ctx context.Context,method,path string,q url.Values,body,out any) error {
	if q==nil { q=url.Values{} }
	var raw []byte
	var err error
	if body!=nil { raw,err=json.Marshal(body); if err!=nil{return err} }
	token,err:=c.accessToken(ctx); if err!=nil{return err}
	return c.request(ctx,method,path,q,raw,token,out)
}
func (c *Client) accessToken(ctx context.Context)(string,error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token!="" && time.Until(c.tokenExpiry)>time.Minute { return c.token,nil }
	q:=url.Values{"grant_type":[]string{"1"}}
	requestPath:="/v1.0/token?"+q.Encode()
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,c.cfg.BaseURL+requestPath,nil); if err!=nil{return "",err}
	ts:=strconv.FormatInt(time.Now().UnixMilli(),10)
	req.Header.Set("client_id",c.cfg.ClientID); req.Header.Set("t",ts); req.Header.Set("sign_method","HMAC-SHA256")
	req.Header.Set("sign",c.signature(http.MethodGet,requestPath,nil,ts,"",true))
	resp,err:=c.http.Do(req); if err!=nil{return "",fmt.Errorf("tuya cloud token request: %w",err)}
	defer resp.Body.Close()
	raw,err:=io.ReadAll(io.LimitReader(resp.Body,maxResponseBytes+1)); if err!=nil{return "",err}
	if len(raw)>maxResponseBytes{return "",errors.New("tuya cloud: response too large")}
	if resp.StatusCode!=http.StatusOK{return "",fmt.Errorf("tuya cloud token endpoint returned HTTP %d",resp.StatusCode)}
	var env struct { Success bool `json:"success"`; Msg string `json:"msg"`; Result tokenResult `json:"result"` }
	if err=json.Unmarshal(raw,&env); err!=nil{return "",fmt.Errorf("tuya cloud: invalid token response: %w",err)}
	if !env.Success || env.Result.AccessToken=="" { return "",fmt.Errorf("tuya cloud: token request rejected: %s",safeMessage(env.Msg)) }
	ttl:=time.Duration(env.Result.ExpireTime)*time.Second
	if ttl<=0 || ttl>24*time.Hour { ttl=2*time.Hour }
	c.token=env.Result.AccessToken; c.tokenExpiry=time.Now().Add(ttl)
	return c.token,nil
}
func (c *Client) request(ctx context.Context,method,path string,q url.Values,body []byte,token string,out any) error {
	requestPath:=path
	if encoded:=q.Encode(); encoded!="" { requestPath+="?"+encoded }
	req,err:=http.NewRequestWithContext(ctx,method,c.cfg.BaseURL+requestPath,bytes.NewReader(body)); if err!=nil{return err}
	ts:=strconv.FormatInt(time.Now().UnixMilli(),10)
	req.Header.Set("client_id",c.cfg.ClientID); req.Header.Set("access_token",token); req.Header.Set("t",ts); req.Header.Set("sign_method","HMAC-SHA256")
	req.Header.Set("sign",c.signature(method,requestPath,body,ts,token,false)); req.Header.Set("Content-Type","application/json")
	resp,err:=c.http.Do(req); if err!=nil{return fmt.Errorf("tuya cloud request: %w",err)}
	defer resp.Body.Close()
	raw,err:=io.ReadAll(io.LimitReader(resp.Body,maxResponseBytes+1)); if err!=nil{return err}
	if len(raw)>maxResponseBytes{return errors.New("tuya cloud: response too large")}
	if resp.StatusCode<200 || resp.StatusCode>=300{return fmt.Errorf("tuya cloud endpoint returned HTTP %d",resp.StatusCode)}
	var env envelope
	if err=json.Unmarshal(raw,&env); err!=nil{return fmt.Errorf("tuya cloud: invalid response: %w",err)}
	if !env.Success {
		if env.Code=="1010" || env.Code=="1011" || env.Code==float64(1010) || env.Code==float64(1011) { c.mu.Lock(); c.token=""; c.tokenExpiry=time.Time{}; c.mu.Unlock() }
		return fmt.Errorf("tuya cloud API error %v: %s",env.Code,safeMessage(env.Msg))
	}
	if out!=nil && len(env.Result)>0 && string(env.Result)!="null" { if err=json.Unmarshal(env.Result,out); err!=nil{return fmt.Errorf("tuya cloud: invalid result: %w",err)} }
	return nil
}
func (c *Client) signature(method,path string,body []byte,ts,token string,isToken bool) string {
	sum:=sha256.Sum256(body)
	toSign:=strings.ToUpper(method)+"\n"+hex.EncodeToString(sum[:])+"\n\n"+path
	prefix:=c.cfg.ClientID
	if !isToken { prefix+=token }
	mac:=hmac.New(sha256.New,[]byte(c.cfg.Secret)); _,_=mac.Write([]byte(prefix+ts+toSign))
	return strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
}
func safeMessage(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) {
			r = ' '
		}
		b.WriteRune(r)
	}
	normalized := strings.Join(strings.Fields(b.String()), " ")
	runes := []rune(normalized)
	if len(runes) > 180 {
		normalized = string(runes[:180])
	}
	return normalized
}
