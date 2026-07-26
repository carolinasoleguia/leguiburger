package production

import (
    "encoding/json"
    "errors"
    "leguiburger/internal/auth"
    "net/http"
    "strings"
)

type Handler struct {
    service Service
}

type createInput struct {
    ProductionDate     string  `json:"production_date"`
    MedallionsProduced int     `json:"medallions_produced"`
    BreadsPurchased    int     `json:"breads_purchased"`
    Notes              string  `json:"notes"`
}

type updateInput struct {
    ProductionDate     *string `json:"production_date,omitempty"`
    MedallionsProduced *int    `json:"medallions_produced,omitempty"`
    BreadsPurchased    *int    `json:"breads_purchased,omitempty"`
    Notes              *string `json:"notes,omitempty"`
}

func NewHandler(s Service) *Handler {
    return &Handler{service: s}
}

func (h *Handler) HandleProductionRoutes(w http.ResponseWriter, r *http.Request) {
    trimmedPath := strings.Trim(r.URL.Path, "/")
    pathParts := strings.Split(trimmedPath, "/")

    if len(pathParts) == 2 && pathParts[0] == "api" && pathParts[1] == "production" {
        switch r.Method {
        case http.MethodPost:
            h.CreateReport(w, r)
            return
        case http.MethodGet:
            h.ListReports(w, r)
            return
        default:
            h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Metodo no permitido")
            return
        }
    }

    if len(pathParts) == 3 && pathParts[0] == "api" && pathParts[1] == "production" {
        id := pathParts[2]
        if id == "" {
            h.respondWithError(w, http.StatusBadRequest, "INVALID_ID", "ID invalido")
            return
        }

        switch r.Method {
        case http.MethodGet:
            h.GetReport(w, r, id)
            return
        case http.MethodPut:
            h.UpdateReport(w, r, id)
            return
        case http.MethodDelete:
            h.DeleteReport(w, r, id)
            return
        default:
            h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Metodo no permitido")
            return
        }
    }

    h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Recurso no encontrado")
}

func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
    tenantID, err := auth.TenantIDFromRequest(r)
    if err != nil {
        h.respondAuthError(w, err)
        return
    }

    claims, ok := auth.GetClaimsFromContext(r.Context())
    if !ok {
        h.respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "No autorizado")
        return
    }

    var input createInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        h.respondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON invalido")
        return
    }

    report, err := h.service.CreateProductionReport(r.Context(), tenantID, input.ProductionDate, input.MedallionsProduced, input.BreadsPurchased, input.Notes, &claims.UserID)
    if err != nil {
        h.handleError(w, err)
        return
    }

    h.respondWithJSON(w, http.StatusCreated, report)
}

func (h *Handler) ListReports(w http.ResponseWriter, r *http.Request) {
    tenantID, err := auth.TenantIDFromRequest(r)
    if err != nil {
        h.respondAuthError(w, err)
        return
    }

    query := r.URL.Query()
    startDate := query.Get("start_date")
    endDate := query.Get("end_date")
    var startPtr, endPtr *string
    if strings.TrimSpace(startDate) != "" {
        startPtr = &startDate
    }
    if strings.TrimSpace(endDate) != "" {
        endPtr = &endDate
    }

    reports, err := h.service.ListProductionReports(r.Context(), tenantID, startPtr, endPtr)
    if err != nil {
        h.handleError(w, err)
        return
    }

    h.respondWithJSON(w, http.StatusOK, reports)
}

func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request, id string) {
    tenantID, err := auth.TenantIDFromRequest(r)
    if err != nil {
        h.respondAuthError(w, err)
        return
    }

    report, err := h.service.GetProductionReport(r.Context(), tenantID, id)
    if err != nil {
        h.handleError(w, err)
        return
    }

    h.respondWithJSON(w, http.StatusOK, report)
}

func (h *Handler) UpdateReport(w http.ResponseWriter, r *http.Request, id string) {
    tenantID, err := auth.TenantIDFromRequest(r)
    if err != nil {
        h.respondAuthError(w, err)
        return
    }

    var input updateInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        h.respondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON invalido")
        return
    }

    report, err := h.service.UpdateProductionReport(r.Context(), tenantID, id, input.ProductionDate, input.MedallionsProduced, input.BreadsPurchased, input.Notes)
    if err != nil {
        h.handleError(w, err)
        return
    }

    h.respondWithJSON(w, http.StatusOK, report)
}

func (h *Handler) DeleteReport(w http.ResponseWriter, r *http.Request, id string) {
    tenantID, err := auth.TenantIDFromRequest(r)
    if err != nil {
        h.respondAuthError(w, err)
        return
    }

    if err := h.service.DeleteProductionReport(r.Context(), tenantID, id); err != nil {
        h.handleError(w, err)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) respondAuthError(w http.ResponseWriter, err error) {
    if errors.Is(err, auth.ErrMissingTenantID) {
        h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
        return
    }
    if errors.Is(err, auth.ErrForbiddenTenant) {
        h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No puedes operar en este comercio")
        return
    }
    h.respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "No autorizado")
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, ErrInvalidProductionData):
        h.respondWithError(w, http.StatusBadRequest, "INVALID_PRODUCTION_DATA", err.Error())
    case errors.Is(err, ErrDuplicateProductionDate):
        h.respondWithError(w, http.StatusConflict, "DUPLICATE_PRODUCTION_DATE", err.Error())
    case errors.Is(err, ErrProductionNotFound):
        h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
    case errors.Is(err, ErrTenantNotFoundForProduction):
        h.respondWithError(w, http.StatusBadRequest, "INVALID_TENANT", err.Error())
    default:
        h.respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Error inesperado")
    }
}

func (h *Handler) respondWithError(w http.ResponseWriter, status int, code, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}
