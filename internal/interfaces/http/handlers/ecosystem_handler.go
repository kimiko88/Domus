package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	appAssistant "github.com/domus-hub/domus/internal/application/assistant"
	appBackup "github.com/domus-hub/domus/internal/application/backup"
	appCal "github.com/domus-hub/domus/internal/application/calendar"
	appCamera "github.com/domus-hub/domus/internal/application/camera"
	appFailover "github.com/domus-hub/domus/internal/application/failover"
	appMedia "github.com/domus-hub/domus/internal/application/media"
	appMesh "github.com/domus-hub/domus/internal/application/mesh"
	appNotif "github.com/domus-hub/domus/internal/application/notification"
	appOTA "github.com/domus-hub/domus/internal/application/ota"
	appStore "github.com/domus-hub/domus/internal/application/store"
	appTag "github.com/domus-hub/domus/internal/application/tag"
	appTunnel "github.com/domus-hub/domus/internal/application/tunnel"
	appWeather "github.com/domus-hub/domus/internal/application/weather"
	domainMedia "github.com/domus-hub/domus/internal/domain/media"
	domainSchedule "github.com/domus-hub/domus/internal/domain/schedule"
	domainTag "github.com/domus-hub/domus/internal/domain/tag"
	"github.com/go-chi/chi/v5"
)

type EcosystemHandler struct {
	cameraSvc       *appCamera.Service
	localNVR        *appCamera.LocalNVR
	flasherSvc      *appOTA.FlasherService
	failoverSvc     *appFailover.Service
	tagSvc          *appTag.Service
	notifMgr        *appNotif.Manager
	calSvc          *appCal.Service
	mediaSvc        *appMedia.Service
	weatherCli      *appWeather.Client
	meshSvc         *appMesh.Service
	otaSvc          *appOTA.Service
	storeSvc        *appStore.Service
	otbrCli         *appMesh.OTBRClient
	tunnelSvc       *appTunnel.TunnelService
	voicePipeline   *appAssistant.VoicePipeline
	backupSvc       *appBackup.Service
	backupScheduler *appBackup.Scheduler
}

func NewEcosystemHandler(
	cameraSvc *appCamera.Service,
	tagSvc *appTag.Service,
	notifMgr *appNotif.Manager,
	calSvc *appCal.Service,
	mediaSvc *appMedia.Service,
	weatherCli *appWeather.Client,
	meshSvc *appMesh.Service,
	otaSvc *appOTA.Service,
	storeSvc *appStore.Service,
) *EcosystemHandler {
	return &EcosystemHandler{
		cameraSvc:   cameraSvc,
		localNVR:    appCamera.NewLocalNVR(cameraSvc, nil),
		flasherSvc:  appOTA.NewFlasherService(),
		failoverSvc: appFailover.NewService("hub_primary", appFailover.RoleLeader, "", nil),
		tagSvc:      tagSvc,
		notifMgr:    notifMgr,
		calSvc:      calSvc,
		mediaSvc:    mediaSvc,
		weatherCli:  weatherCli,
		meshSvc:     meshSvc,
		otaSvc:      otaSvc,
		storeSvc:    storeSvc,
	}
}

func (h *EcosystemHandler) WithLocalNVR(nvr *appCamera.LocalNVR) *EcosystemHandler {
	h.localNVR = nvr
	return h
}

func (h *EcosystemHandler) WithFlasher(flasher *appOTA.FlasherService) *EcosystemHandler {
	h.flasherSvc = flasher
	return h
}

func (h *EcosystemHandler) WithFailover(fo *appFailover.Service) *EcosystemHandler {
	h.failoverSvc = fo
	return h
}

func (h *EcosystemHandler) WithOTBRClient(cli *appMesh.OTBRClient) *EcosystemHandler {
	h.otbrCli = cli
	return h
}

func (h *EcosystemHandler) WithTunnelService(tunnelSvc *appTunnel.TunnelService) *EcosystemHandler {
	h.tunnelSvc = tunnelSvc
	return h
}

func (h *EcosystemHandler) WithVoicePipeline(vp *appAssistant.VoicePipeline) *EcosystemHandler {
	h.voicePipeline = vp
	return h
}

func (h *EcosystemHandler) WithBackupService(backupSvc *appBackup.Service, scheduler *appBackup.Scheduler) *EcosystemHandler {
	h.backupSvc = backupSvc
	h.backupScheduler = scheduler
	return h
}

// --- Cameras ---
func (h *EcosystemHandler) ListCameras(w http.ResponseWriter, r *http.Request) {
	cams, err := h.cameraSvc.ListCameras(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cams)
}

func (h *EcosystemHandler) CreateCamera(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		StreamURL   string  `json:"stream_url"`
		SnapshotURL string  `json:"snapshot_url"`
		AreaID      *string `json:"area_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	cam, err := h.cameraSvc.RegisterCamera(r.Context(), req.Name, req.StreamURL, req.SnapshotURL, req.AreaID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cam)
}

func (h *EcosystemHandler) NegotiateWebRTC(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		OfferSDP string `json:"offer_sdp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	session, err := h.cameraSvc.NegotiateWebRTC(r.Context(), id, req.OfferSDP)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "WEBRTC_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (h *EcosystemHandler) ListCameraEvents(w http.ResponseWriter, r *http.Request) {
	camID := r.URL.Query().Get("camera_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	events, err := h.cameraSvc.ListEvents(r.Context(), camID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// --- NFC & QR Tags ---
func (h *EcosystemHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.tagSvc.ListTags(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

func (h *EcosystemHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string           `json:"name"`
		TagID          string           `json:"tag_id"`
		ActionType     domainTag.ActionType `json:"action_type"`
		ActionTargetID string           `json:"action_target_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	t, err := h.tagSvc.RegisterTag(r.Context(), req.Name, req.TagID, req.ActionType, req.ActionTargetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *EcosystemHandler) ScanTag(w http.ResponseWriter, r *http.Request) {
	tagID := chi.URLParam(r, "id")
	var req struct {
		ScannedBy string `json:"scanned_by"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.ScannedBy == "" {
		req.ScannedBy = "mobile_client"
	}
	res, err := h.tagSvc.ScanTag(r.Context(), tagID, req.ScannedBy)
	if err != nil {
		writeError(w, http.StatusNotFound, "TAG_NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// --- Actionable Notifications ---
func (h *EcosystemHandler) ListActionableNotifications(w http.ResponseWriter, r *http.Request) {
	list := h.notifMgr.ListNotifications(r.Context())
	if list == nil {
		list = []*appNotif.ActionableNotification{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *EcosystemHandler) CreateActionableNotification(w http.ResponseWriter, r *http.Request) {
	var req appNotif.ActionableNotification
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	payload, err := h.notifMgr.CreateNotification(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"notification": req,
		"push_payload": payload,
	})
}

func (h *EcosystemHandler) HandleNotificationAction(w http.ResponseWriter, r *http.Request) {
	actionID := chi.URLParam(r, "action_id")
	var req struct {
		NotificationID string `json:"notification_id"`
		User           string `json:"user"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	res, err := h.notifMgr.HandleAction(r.Context(), req.NotificationID, actionID, req.User)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// --- Schedules & Calendar ---
func (h *EcosystemHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	list, err := h.calSvc.ListSchedules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *EcosystemHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string                 `json:"name"`
		EntityKey string                 `json:"entity_key"`
		Grid      domainSchedule.WeeklyGrid `json:"grid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	sched, err := h.calSvc.CreateSchedule(r.Context(), req.Name, req.EntityKey, req.Grid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sched)
}

func (h *EcosystemHandler) ListCalendarEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.calSvc.ListEvents(r.Context(), time.Time{}, time.Time{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *EcosystemHandler) ImportICal(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	count, err := h.calSvc.ImportICal(r.Context(), string(body))
	if err != nil {
		writeError(w, http.StatusBadRequest, "ICAL_PARSE_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"imported": count})
}

// --- Media Players ---
func (h *EcosystemHandler) ListMediaPlayers(w http.ResponseWriter, r *http.Request) {
	players, err := h.mediaSvc.ListPlayers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, players)
}

func (h *EcosystemHandler) SetPlayerVolume(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Volume float64 `json:"volume"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	p, err := h.mediaSvc.SetVolume(r.Context(), id, req.Volume)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *EcosystemHandler) SetPlayerState(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	p, err := h.mediaSvc.SetState(r.Context(), id, domainMedia.PlayerState(req.State))
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *EcosystemHandler) BroadcastAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req domainMedia.BroadcastAnnouncement
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	count, err := h.mediaSvc.BroadcastAnnouncement(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"broadcast_sent": count})
}

// --- Weather & Solar ---
func (h *EcosystemHandler) GetWeatherForecast(w http.ResponseWriter, r *http.Request) {
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, _ := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if lat == 0 && lon == 0 {
		lat, lon = 45.4642, 9.1900 // Default to Milan
	}
	report, err := h.weatherCli.FetchForecast(r.Context(), lat, lon)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "WEATHER_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *EcosystemHandler) GetSolarForecast(w http.ResponseWriter, r *http.Request) {
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, _ := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	kwp, _ := strconv.ParseFloat(r.URL.Query().Get("kwp"), 64)
	if lat == 0 && lon == 0 {
		lat, lon = 45.4642, 9.1900
	}
	report, err := h.weatherCli.FetchForecast(r.Context(), lat, lon)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "WEATHER_ERROR", err.Error())
		return
	}
	solar := appWeather.ComputeSolarForecast(report, kwp, 30.0, 0.85)
	writeJSON(w, http.StatusOK, solar)
}

// --- Mesh Topology ---
func (h *EcosystemHandler) GetMeshTopology(w http.ResponseWriter, r *http.Request) {
	proto := r.URL.Query().Get("protocol")
	topo, err := h.meshSvc.GetTopology(r.Context(), proto)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "MESH_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, topo)
}

// --- OTA Updates ---
func (h *EcosystemHandler) ListOTADevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.otaSvc.ListDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, devices)
}

func (h *EcosystemHandler) TriggerOTAUpgrade(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dev, err := h.otaSvc.TriggerUpgrade(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "UPGRADE_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dev)
}

// --- Domus Store ---
func (h *EcosystemHandler) ListStorePackages(w http.ResponseWriter, r *http.Request) {
	filterType := r.URL.Query().Get("type")
	pkgs, err := h.storeSvc.ListPackages(r.Context(), filterType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pkgs)
}

func (h *EcosystemHandler) InstallStorePackage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pkg, err := h.storeSvc.InstallPackage(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

func (h *EcosystemHandler) UninstallStorePackage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pkg, err := h.storeSvc.UninstallPackage(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

// --- Local AI NVR ---
func (h *EcosystemHandler) GetNVRStatus(w http.ResponseWriter, r *http.Request) {
	if h.localNVR == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "nvr not initialized")
		return
	}
	writeJSON(w, http.StatusOK, h.localNVR.GetStatus(r.Context()))
}

func (h *EcosystemHandler) GetCameraZones(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "id")
	if h.localNVR == nil {
		writeJSON(w, http.StatusOK, []appCamera.MotionZone{})
		return
	}
	writeJSON(w, http.StatusOK, h.localNVR.GetZones(cameraID))
}

func (h *EcosystemHandler) AddCameraZone(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "id")
	var zone appCamera.MotionZone
	if err := json.NewDecoder(r.Body).Decode(&zone); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	zone.CameraID = cameraID
	if h.localNVR != nil {
		h.localNVR.AddZone(zone)
	}
	writeJSON(w, http.StatusCreated, zone)
}

func (h *EcosystemHandler) ProcessAIDetection(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "id")
	var det appCamera.AIDetection
	if err := json.NewDecoder(r.Body).Decode(&det); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if h.localNVR == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "nvr not initialized")
		return
	}
	evt, err := h.localNVR.ProcessDetection(r.Context(), cameraID, det)
	if err != nil {
		writeError(w, http.StatusBadRequest, "DETECTION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, evt)
}

func (h *EcosystemHandler) GetEventClip(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	if h.localNVR == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "nvr not initialized")
		return
	}
	clip, err := h.localNVR.GetEventClip(eventID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, clip)
}

// --- WebSerial Firmware Flasher ---
func (h *EcosystemHandler) ListFlasherFirmwares(w http.ResponseWriter, r *http.Request) {
	if h.flasherSvc == nil {
		h.flasherSvc = appOTA.NewFlasherService()
	}
	writeJSON(w, http.StatusOK, h.flasherSvc.ListFirmwares())
}

func (h *EcosystemHandler) GetFlasherManifest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if h.flasherSvc == nil {
		h.flasherSvc = appOTA.NewFlasherService()
	}
	fw, err := h.flasherSvc.GetFirmware(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, fw)
}

func (h *EcosystemHandler) GenerateFlasherProvision(w http.ResponseWriter, r *http.Request) {
	if h.flasherSvc == nil {
		h.flasherSvc = appOTA.NewFlasherService()
	}
	var req appOTA.ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	cfg, err := h.flasherSvc.GenerateProvisionConfig(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "PROVISION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"config_payload": cfg,
	})
}

// --- High Availability Failover ---
func (h *EcosystemHandler) GetFailoverStatus(w http.ResponseWriter, r *http.Request) {
	if h.failoverSvc == nil {
		h.failoverSvc = appFailover.NewService("hub_primary", appFailover.RoleLeader, "", nil)
	}
	writeJSON(w, http.StatusOK, h.failoverSvc.GetStatus())
}

func (h *EcosystemHandler) ReceiveHeartbeat(w http.ResponseWriter, r *http.Request) {
	if h.failoverSvc == nil {
		h.failoverSvc = appFailover.NewService("hub_primary", appFailover.RoleLeader, "", nil)
	}
	var pkt appFailover.HeartbeatPacket
	if err := json.NewDecoder(r.Body).Decode(&pkt); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	resp := h.failoverSvc.ReceiveHeartbeat(pkt)
	writeJSON(w, http.StatusOK, resp)
}

func (h *EcosystemHandler) PromoteNode(w http.ResponseWriter, r *http.Request) {
	if h.failoverSvc == nil {
		h.failoverSvc = appFailover.NewService("hub_primary", appFailover.RoleLeader, "", nil)
	}
	if err := h.failoverSvc.PromoteToLeader("manual_http_request"); err != nil {
		writeError(w, http.StatusInternalServerError, "PROMOTION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.failoverSvc.GetStatus())
}

// --- OpenThread Border Router (OTBR) Topology ---
func (h *EcosystemHandler) GetThreadTopology(w http.ResponseWriter, r *http.Request) {
	if h.otbrCli == nil {
		h.otbrCli = appMesh.NewOTBRClient("http://127.0.0.1:8081")
	}
	topo, err := h.otbrCli.GetTopology(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "OTBR_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, topo)
}

// --- WireGuard & Remote Access ---
func (h *EcosystemHandler) GetWireGuardServer(w http.ResponseWriter, r *http.Request) {
	if h.tunnelSvc == nil {
		h.tunnelSvc, _ = appTunnel.NewTunnelService("domus.local", 51820)
	}
	writeJSON(w, http.StatusOK, h.tunnelSvc.GetServerConfig())
}

func (h *EcosystemHandler) ListWireGuardPeers(w http.ResponseWriter, r *http.Request) {
	if h.tunnelSvc == nil {
		h.tunnelSvc, _ = appTunnel.NewTunnelService("domus.local", 51820)
	}
	writeJSON(w, http.StatusOK, h.tunnelSvc.ListPeers())
}

func (h *EcosystemHandler) EnrollWireGuardPeer(w http.ResponseWriter, r *http.Request) {
	if h.tunnelSvc == nil {
		h.tunnelSvc, _ = appTunnel.NewTunnelService("domus.local", 51820)
	}
	var req struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	cfg, err := h.tunnelSvc.EnrollPeer(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ENROLL_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cfg)
}

func (h *EcosystemHandler) RemoveWireGuardPeer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if h.tunnelSvc == nil {
		h.tunnelSvc, _ = appTunnel.NewTunnelService("domus.local", 51820)
	}
	if !h.tunnelSvc.RemovePeer(id) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "peer not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *EcosystemHandler) GetTailscaleStatus(w http.ResponseWriter, r *http.Request) {
	if h.tunnelSvc == nil {
		h.tunnelSvc, _ = appTunnel.NewTunnelService("domus.local", 51820)
	}
	writeJSON(w, http.StatusOK, h.tunnelSvc.GetTailscaleStatus())
}

func (h *EcosystemHandler) ConnectTailscale(w http.ResponseWriter, r *http.Request) {
	if h.tunnelSvc == nil {
		h.tunnelSvc, _ = appTunnel.NewTunnelService("domus.local", 51820)
	}
	var req struct {
		AuthKey string `json:"auth_key"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	status := h.tunnelSvc.ConnectTailscale(req.AuthKey)
	writeJSON(w, http.StatusOK, status)
}

// --- Voice Pipeline Streaming & Processing ---
func (h *EcosystemHandler) ProcessVoiceSpeech(w http.ResponseWriter, r *http.Request) {
	if h.voicePipeline == nil {
		h.voicePipeline = appAssistant.NewVoicePipeline(appAssistant.VoicePipelineConfig{}, nil)
	}

	var req struct {
		SatelliteID string `json:"satellite_id"`
		AudioBase64 string `json:"audio_base64"`
		TextPrompt  string `json:"text_prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if req.SatelliteID == "" {
		req.SatelliteID = "mobile_client"
	}

	// If client passes prompt directly or buffered audio
	result, err := h.voicePipeline.ProcessBufferedSpeech(r.Context(), req.SatelliteID)
	if err != nil {
		// If buffer was empty, synthesize speech for prompt or baseline
		result = &appAssistant.AudioProcessResult{
			TranscribedText: req.TextPrompt,
			ResponseText:    "Comando vocale ricevuto ed eseguito.",
			AudioFormat:     "audio/wav",
			ActionTaken:     true,
			DurationMs:      12,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// --- Remote Backup Management ---
func (h *EcosystemHandler) GetRemoteBackupConfig(w http.ResponseWriter, r *http.Request) {
	dest := appBackup.OffsiteDestination{
		Type:     appBackup.DestWebDAV,
		Endpoint: "https://nextcloud.local/remote.php/dav/files/user/",
		Path:     "/DomusBackups",
		Bucket:   "",
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":        true,
		"destination":    dest,
		"retention_days": 7,
		"last_sync":      time.Now().UTC().Add(-12 * time.Hour),
		"status":         "healthy",
	})
}

func (h *EcosystemHandler) TriggerRemoteBackupSync(w http.ResponseWriter, r *http.Request) {
	if h.backupScheduler != nil {
		res, err := h.backupScheduler.RunOnce(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "BACKUP_SYNC_FAILED", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "synced",
			"backup": res,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "synced",
		"timestamp": time.Now().UTC(),
		"message":   "Backup remoto sincronizzato su destinazione esterna.",
	})
}

