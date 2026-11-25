package handlers

import (
    "encoding/base64"
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "strings"

    "github.com/domus-hub/domus/internal/plugins/open_display"
    "github.com/go-chi/chi/v5"
)

const maxOpenDisplayUploadBytes = 8 << 20

type OpenDisplayHandler struct {
    adapter *open_display.Adapter
}

func NewOpenDisplayHandler(adapter *open_display.Adapter) *OpenDisplayHandler {
    return &OpenDisplayHandler{adapter: adapter}
}

func (h *OpenDisplayHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
    if h.adapter == nil {
        writeError(w, http.StatusServiceUnavailable, "OPENDISPLAY_DISABLED", "OpenDisplay integration is not enabled")
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"devices": h.adapter.ListDevices()})
}

func (h *OpenDisplayHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
    if h.adapter == nil {
        writeError(w, http.StatusServiceUnavailable, "OPENDISPLAY_DISABLED", "OpenDisplay integration is not enabled")
        return
    }
    r.Body = http.MaxBytesReader(w, r.Body, 12<<20)
    var req struct {
        ImageBase64 string `json:"image_base64"`
        Fit string `json:"fit"`
        RefreshMode string `json:"refresh_mode"`
    }
    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "INVALID_JSON", "provide image_base64, fit and refresh_mode")
        return
    }
    var trailing any
    if err := decoder.Decode(&trailing); err != io.EOF {
        writeError(w, http.StatusBadRequest, "INVALID_JSON", "unexpected trailing JSON content")
        return
    }
    if len(req.ImageBase64) == 0 || len(req.ImageBase64) > ((maxOpenDisplayUploadBytes+2)/3)*4 {
        writeError(w, http.StatusRequestEntityTooLarge, "IMAGE_TOO_LARGE", "image exceeds the 8 MiB limit")
        return
    }
    image, err := base64.StdEncoding.DecodeString(req.ImageBase64)
    if err != nil || len(image) == 0 || len(image) > maxOpenDisplayUploadBytes {
        writeError(w, http.StatusBadRequest, "INVALID_IMAGE", "image_base64 must contain raster image data up to 8 MiB")
        return
    }
    address := strings.TrimSpace(chi.URLParam(r, "address"))
    if address == "" {
        writeError(w, http.StatusBadRequest, "INVALID_DEVICE", "device address is required")
        return
    }
    if err := h.adapter.UploadImage(r.Context(), address, image, req.Fit, req.RefreshMode); err != nil {
        if errors.Is(err, context.Canceled) {
            writeError(w, http.StatusRequestTimeout, "REQUEST_CANCELLED", "image upload was cancelled")
            return
        }
        if strings.Contains(err.Error(), "not currently discovered") {
            writeError(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "OpenDisplay device is not currently discovered")
            return
        }
        writeError(w, http.StatusBadGateway, "UPLOAD_FAILED", "OpenDisplay image upload failed")
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"ok": true, "address": address})
}
