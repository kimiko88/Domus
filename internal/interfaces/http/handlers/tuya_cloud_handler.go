package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
	"github.com/go-chi/chi/v5"
)

// TuyaCloudHandler exposes a narrowly scoped server-side Tuya OpenAPI integration.
type TuyaCloudHandler struct { client *tuya_cloud.Client }

func NewTuyaCloudHandler(client *tuya_cloud.Client) *TuyaCloudHandler { return &TuyaCloudHandler{client:client} }

func (h *TuyaCloudHandler) RegisterRoutes(r chi.Router) {
	r.Get("/devices",h.ListDevices)
	r.Post("/devices/{deviceID}/commands",h.SendCommands)
}
func (h *TuyaCloudHandler) ListDevices(w http.ResponseWriter,r *http.Request) {
	devices,err:=h.client.Devices(r.Context())
	if err!=nil { writeError(w,http.StatusBadGateway,"TUYA_CLOUD_ERROR",err.Error()); return }
	writeJSON(w,http.StatusOK,map[string]any{"devices":devices})
}
func (h *TuyaCloudHandler) SendCommands(w http.ResponseWriter,r *http.Request) {
	r.Body=http.MaxBytesReader(w,r.Body,16*1024)
	var req struct { Commands []tuya_cloud.Command `json:"commands"` }
	if err:=json.NewDecoder(r.Body).Decode(&req); err!=nil { writeError(w,http.StatusBadRequest,"INVALID_JSON","invalid command payload"); return }
	if len(req.Commands)==0 || len(req.Commands)>32 { writeError(w,http.StatusBadRequest,"VALIDATION_ERROR","between 1 and 32 commands are required"); return }
	if err:=h.client.SendCommands(r.Context(),chi.URLParam(r,"deviceID"),req.Commands); err!=nil { writeError(w,http.StatusBadGateway,"TUYA_CLOUD_ERROR",err.Error()); return }
	writeJSON(w,http.StatusOK,map[string]bool{"success":true})
}
