package handlers

import (
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "strings"

    "github.com/domus-hub/domus/internal/application/marketplace"
    "github.com/go-chi/chi/v5"
)

type MarketplaceHandler struct {
    service *marketplace.Service
}

func NewMarketplaceHandler(service *marketplace.Service) *MarketplaceHandler {
    return &MarketplaceHandler{service: service}
}

func (h *MarketplaceHandler) ListRepositories(w http.ResponseWriter, r *http.Request) {
    sources, err := h.service.ListRepositories(r.Context())
    if err != nil {
        writeError(w, http.StatusInternalServerError, "MARKETPLACE_ERROR", "could not list plugin repositories")
        return
    }
    writeJSON(w, http.StatusOK, sources)
}

func (h *MarketplaceHandler) AddRepository(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
    var request struct {
        URL string `json:"url"`
    }
    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(&request); err != nil {
        writeError(w, http.StatusBadRequest, "INVALID_JSON", "repository request must contain a URL")
        return
    }
    var trailing any
    if err := decoder.Decode(&trailing); err != io.EOF {
        writeError(w, http.StatusBadRequest, "INVALID_JSON", "unexpected trailing JSON content")
        return
    }
    source, err := h.service.AddRepository(r.Context(), request.URL)
    if err != nil {
        writeError(w, http.StatusBadRequest, "INVALID_REPOSITORY", err.Error())
        return
    }
    writeJSON(w, http.StatusCreated, source)
}

func (h *MarketplaceHandler) RemoveRepository(w http.ResponseWriter, r *http.Request) {
    id := strings.TrimSpace(chi.URLParam(r, "id"))
    if err := h.service.RemoveRepository(r.Context(), id); err != nil {
        status := http.StatusInternalServerError
        if errors.Is(err, marketplace.ErrNotFound) { status = http.StatusNotFound }
        writeError(w, status, "REMOVE_REPOSITORY_FAILED", "repository could not be removed")
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"removed": true, "id": id})
}

func (h *MarketplaceHandler) SyncRepository(w http.ResponseWriter, r *http.Request) {
    source, err := h.service.SyncRepository(r.Context(), strings.TrimSpace(chi.URLParam(r, "id")))
    if err != nil {
        status := http.StatusBadGateway
        if errors.Is(err, marketplace.ErrNotFound) { status = http.StatusNotFound }
        if errors.Is(err, marketplace.ErrUntrusted) { status = http.StatusForbidden }
        writeError(w, status, "SYNC_REPOSITORY_FAILED", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, source)
}

func (h *MarketplaceHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
    packages, err := h.service.ListPackages(r.Context(), strings.TrimSpace(r.URL.Query().Get("repository_id")))
    if err != nil {
        writeError(w, http.StatusInternalServerError, "MARKETPLACE_ERROR", "could not list repository packages")
        return
    }
    writeJSON(w, http.StatusOK, packages)
}

func (h *MarketplaceHandler) InstallPackage(w http.ResponseWriter, r *http.Request) {
    pkg, err := h.service.InstallArtifact(r.Context(), strings.TrimSpace(chi.URLParam(r, "id")))
    if err != nil {
        status := http.StatusBadGateway
        if errors.Is(err, marketplace.ErrNotFound) { status = http.StatusNotFound }
        if errors.Is(err, marketplace.ErrUntrusted) { status = http.StatusForbidden }
        if errors.Is(err, marketplace.ErrArtifactIntegrity) { status = http.StatusUnprocessableEntity }
        writeError(w, status, "PLUGIN_INSTALL_FAILED", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, pkg)
}

func (h *MarketplaceHandler) UninstallPackage(w http.ResponseWriter, r *http.Request) {
    pkg, err := h.service.UninstallArtifact(r.Context(), strings.TrimSpace(chi.URLParam(r, "id")))
    if err != nil {
        status := http.StatusInternalServerError
        if errors.Is(err, marketplace.ErrNotFound) { status = http.StatusNotFound }
        writeError(w, status, "PLUGIN_UNINSTALL_FAILED", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, pkg)
}
