package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TuyaOAuthHandler struct {
	db *sql.DB
	cipher *crypto.EnvelopeCipher
	client *tuya_cloud.OAuthClient
	clientID string
	authorizeURL string
	callbackURL string
	refreshMu sync.Mutex
}

func NewTuyaOAuthHandler(db *sql.DB, cipher *crypto.EnvelopeCipher, client *tuya_cloud.OAuthClient, clientID, authorizeURL, callbackURL string) (*TuyaOAuthHandler,error) {
	if db==nil || cipher==nil || client==nil || strings.TrimSpace(clientID)=="" { return nil,errors.New("tuya oauth: database, encryption, client and client ID are required") }
	authURL,err:=url.Parse(authorizeURL)
	if err!=nil || authURL.Scheme!="https" || authURL.Host=="" || authURL.User!=nil || authURL.Fragment!="" { return nil,errors.New("tuya oauth: authorization H5 URL must be absolute HTTPS") }
	cb,err:=url.Parse(callbackURL)
	if err!=nil || cb.Scheme!="https" || cb.Host=="" || cb.User!=nil || cb.RawQuery!="" || cb.Fragment!="" { return nil,errors.New("tuya oauth: callback URL must be absolute HTTPS without query or fragment") }
	h:=&TuyaOAuthHandler{db:db,cipher:cipher,client:client,clientID:clientID,authorizeURL:authorizeURL,callbackURL:callbackURL}
	_,err=db.Exec(`CREATE TABLE IF NOT EXISTS tuya_user_oauth_states (state TEXT PRIMARY KEY, user_id TEXT NOT NULL, expires_at INTEGER NOT NULL)`)
	if err!=nil{return nil,fmt.Errorf("tuya oauth: initialize state storage: %w",err)}
	_,err=db.Exec(`CREATE TABLE IF NOT EXISTS tuya_user_oauth_connections (user_id TEXT PRIMARY KEY, tuya_uid TEXT NOT NULL, access_token_enc TEXT NOT NULL, refresh_token_enc TEXT NOT NULL, expires_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)`)
	if err!=nil{return nil,fmt.Errorf("tuya oauth: initialize token storage: %w",err)}
	return h,nil
}

func (h *TuyaOAuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/authorize",h.Start)
	r.Get("/status",h.Status)
	r.Get("/devices",h.Devices)
	r.Post("/disconnect",h.Disconnect)
}
func (h *TuyaOAuthHandler) RegisterCallback(r chi.Router) { r.Get("/integrations/tuya-oauth/callback",h.Callback) }

func (h *TuyaOAuthHandler) Start(w http.ResponseWriter,r *http.Request) {
	u:=domusmw.GetUser(r.Context())
	if u==nil || u.Role!=user.RoleAdmin { writeError(w,http.StatusForbidden,"FORBIDDEN","administrator role required");return }
	state:=uuid.NewString()+uuid.NewString()
	_,err:=h.db.ExecContext(r.Context(),`DELETE FROM tuya_user_oauth_states WHERE expires_at < ?`,time.Now().Unix())
	if err==nil { _,err=h.db.ExecContext(r.Context(),`INSERT INTO tuya_user_oauth_states(state,user_id,expires_at) VALUES(?,?,?)`,state,u.ID,time.Now().Add(10*time.Minute).Unix()) }
	if err!=nil { writeError(w,http.StatusInternalServerError,"STORAGE_ERROR","could not create authorization session");return }
	authURL,err:=url.Parse(h.authorizeURL)
	if err!=nil { writeError(w,http.StatusInternalServerError,"CONFIG_ERROR","invalid authorization URL");return }
	q:=authURL.Query()
	q.Set("client_id",h.clientID)
	q.Set("callback_url",h.callbackURL)
	q.Set("state",state)
	authURL.RawQuery=q.Encode()
	writeJSON(w,http.StatusOK,map[string]string{"authorization_url":authURL.String()})
}

func (h *TuyaOAuthHandler) Callback(w http.ResponseWriter,r *http.Request) {
	state:=r.URL.Query().Get("state")
	code:=r.URL.Query().Get("code")
	if state=="" || len(state)>128 || code=="" || len(code)>2048 {
		http.Error(w,"Autorizzazione Tuya non valida. Torna a Domus e riprova.",http.StatusBadRequest);return
	}
	var userID string
	var expires int64
	err:=h.db.QueryRowContext(r.Context(),`SELECT user_id,expires_at FROM tuya_user_oauth_states WHERE state=?`,state).Scan(&userID,&expires)
	if err!=nil || expires<time.Now().Unix() {
		_,_=h.db.ExecContext(r.Context(),`DELETE FROM tuya_user_oauth_states WHERE state=?`,state)
		http.Error(w,"Sessione OAuth scaduta o non valida. Torna a Domus e riprova.",http.StatusBadRequest);return
	}
	res,err:=h.db.ExecContext(r.Context(),`DELETE FROM tuya_user_oauth_states WHERE state=?`,state)
	if err!=nil { http.Error(w,"Impossibile completare la sessione OAuth.",http.StatusInternalServerError);return }
	n,_:=res.RowsAffected()
	if n!=1 { http.Error(w,"Sessione OAuth già utilizzata.",http.StatusBadRequest);return }
	token,err:=h.client.ExchangeCode(r.Context(),code)
	if err!=nil { http.Error(w,"Tuya non ha completato l'autorizzazione. Verifica le impostazioni OAuth e riprova.",http.StatusBadGateway);return }
	if err=h.storeToken(r.Context(),userID,token);err!=nil { http.Error(w,"Token ricevuto ma non è stato possibile salvarlo in modo sicuro.",http.StatusInternalServerError);return }
	writeOAuthSuccessPage(w)
}

func (h *TuyaOAuthHandler) Status(w http.ResponseWriter,r *http.Request) {
	u:=domusmw.GetUser(r.Context())
	if u==nil || u.Role!=user.RoleAdmin { writeError(w,http.StatusForbidden,"FORBIDDEN","administrator role required");return }
	var expires int64
	err:=h.db.QueryRowContext(r.Context(),`SELECT expires_at FROM tuya_user_oauth_connections WHERE user_id=?`,u.ID).Scan(&expires)
	if errors.Is(err,sql.ErrNoRows) { writeJSON(w,http.StatusOK,map[string]any{"connected":false});return }
	if err!=nil { writeError(w,http.StatusInternalServerError,"STORAGE_ERROR","could not read connection status");return }
	writeJSON(w,http.StatusOK,map[string]any{"connected":true,"expires_at":time.Unix(expires,0).UTC()})
}

func (h *TuyaOAuthHandler) Devices(w http.ResponseWriter,r *http.Request) {
	u:=domusmw.GetUser(r.Context())
	if u==nil || u.Role!=user.RoleAdmin { writeError(w,http.StatusForbidden,"FORBIDDEN","administrator role required");return }
	token,err:=h.validToken(r.Context(),u.ID)
	if err!=nil { writeError(w,http.StatusBadGateway,"TUYA_OAUTH_ERROR","Smart Life account is not connected or token renewal failed");return }
	devices,err:=h.client.Devices(r.Context(),token)
	if err!=nil { writeError(w,http.StatusBadGateway,"TUYA_OAUTH_ERROR","could not retrieve authorized Smart Life devices");return }
	writeJSON(w,http.StatusOK,map[string]any{"devices":devices})
}

func (h *TuyaOAuthHandler) Disconnect(w http.ResponseWriter,r *http.Request) {
	u:=domusmw.GetUser(r.Context())
	if u==nil || u.Role!=user.RoleAdmin { writeError(w,http.StatusForbidden,"FORBIDDEN","administrator role required");return }
	if _,err:=h.db.ExecContext(r.Context(),`DELETE FROM tuya_user_oauth_connections WHERE user_id=?`,u.ID);err!=nil { writeError(w,http.StatusInternalServerError,"STORAGE_ERROR","could not remove connection");return }
	writeJSON(w,http.StatusOK,map[string]bool{"success":true})
}

func (h *TuyaOAuthHandler) validToken(ctx context.Context,userID string)(tuya_cloud.OAuthToken,error) {
	h.refreshMu.Lock()
	defer h.refreshMu.Unlock()
	var uid,accessEnc,refreshEnc string
	var expires int64
	err:=h.db.QueryRowContext(ctx,`SELECT tuya_uid,access_token_enc,refresh_token_enc,expires_at FROM tuya_user_oauth_connections WHERE user_id=?`,userID).Scan(&uid,&accessEnc,&refreshEnc,&expires)
	if err!=nil{return tuya_cloud.OAuthToken{},err}
	access,err:=h.cipher.Decrypt(accessEnc);if err!=nil{return tuya_cloud.OAuthToken{},err}
	refresh,err:=h.cipher.Decrypt(refreshEnc);if err!=nil{return tuya_cloud.OAuthToken{},err}
	if time.Until(time.Unix(expires,0))>5*time.Minute {
		return tuya_cloud.OAuthToken{UID:uid,AccessToken:access,RefreshToken:refresh,ExpireTime:int64(time.Until(time.Unix(expires,0)).Seconds())},nil
	}
	next,err:=h.client.Refresh(ctx,refresh)
	if err!=nil{return tuya_cloud.OAuthToken{},err}
	if next.UID!=uid { return tuya_cloud.OAuthToken{},errors.New("tuya oauth: refreshed token UID mismatch") }
	if err=h.storeToken(ctx,userID,next);err!=nil{return tuya_cloud.OAuthToken{},err}
	return next,nil
}

func (h *TuyaOAuthHandler) storeToken(ctx context.Context,userID string,token tuya_cloud.OAuthToken) error {
	access,err:=h.cipher.Encrypt(token.AccessToken);if err!=nil{return err}
	refresh,err:=h.cipher.Encrypt(token.RefreshToken);if err!=nil{return err}
	_,err=h.db.ExecContext(ctx,`INSERT INTO tuya_user_oauth_connections(user_id,tuya_uid,access_token_enc,refresh_token_enc,expires_at,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET tuya_uid=excluded.tuya_uid,access_token_enc=excluded.access_token_enc,refresh_token_enc=excluded.refresh_token_enc,expires_at=excluded.expires_at,updated_at=excluded.updated_at`,userID,token.UID,access,refresh,time.Now().Add(time.Duration(token.ExpireTime)*time.Second).Unix(),time.Now().Unix())
	return err
}


// writeOAuthSuccessPage confirms completion without redirecting the browser to the
// public callback hostname. Domus itself remains reachable only over the user's
// private network (for example, Tailscale).
func writeOAuthSuccessPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="it">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Smart Life collegato — Domus</title>
</head>
<body>
<main>
<h1>Autorizzazione Smart Life completata</h1>
<p>L'account Smart Life è stato collegato a Domus e i token sono stati salvati in modo sicuro.</p>
<p>Puoi chiudere questa scheda e tornare all'interfaccia Domus tramite la tua connessione Tailscale.</p>
</main>
</body>
</html>`))
}
