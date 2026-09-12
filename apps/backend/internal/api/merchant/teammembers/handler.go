package teammembers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/Seizmann/RexiO-Pay/backend/internal/api/merchant/common"
	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	"github.com/Seizmann/RexiO-Pay/backend/internal/billing"
	dbpkg "github.com/Seizmann/RexiO-Pay/backend/internal/db"
	db "github.com/Seizmann/RexiO-Pay/backend/internal/db/sqlc"
	"github.com/Seizmann/RexiO-Pay/backend/internal/middleware"
	"github.com/Seizmann/RexiO-Pay/backend/internal/roles"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	queries *db.Queries
	limits  *billing.Limiter
}

func NewHandler(pool *dbpkg.Pool) *Handler {
	return &Handler{queries: db.New(pool.SqlDB), limits: billing.New(pool)}
}
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Invite)
	r.Patch("/{id}", h.UpdateRole)
	r.Delete("/{id}", h.Remove)
	return r
}

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}
type roleRequest struct {
	Role string `json:"role"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	items, err := h.queries.ListTeamMembers(r.Context(), merchantID)
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}
func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	if !h.limits.Check(r.Context(), w, merchantID, billing.ResourceTeamMembers) {
		return
	}
	var req inviteRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if req.Role == "" {
		req.Role = roles.RoleStaff
	}
	if req.Email == "" || !validRole(req.Role) {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "email and a valid role are required")
		return
	}
	user, err := h.queries.GetUserByEmail(r.Context(), req.Email)
	if errors.Is(err, sql.ErrNoRows) {
		apierr.Render(w, http.StatusNotFound, apierr.CodeUserNotFound, "user not found")
		return
	}
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	actor := middleware.GetMerchant(r.Context())
	invitedBy := sql.NullString{}
	if actor != nil {
		invitedBy.Valid = false
	}
	if err := h.queries.InviteTeamMember(r.Context(), db.InviteTeamMemberParams{MerchantID: merchantID, UserID: user.ID, Role: req.Role, InvitedBy: invitedBy}); err != nil {
		apierr.Internal(w, err)
		return
	}
	member, err := h.queries.GetTeamMember(r.Context(), db.GetTeamMemberParams{MerchantID: merchantID, UserID: user.ID})
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusCreated, member)
}
func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	var req roleRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if !validRole(req.Role) {
		apierr.Render(w, http.StatusBadRequest, apierr.CodeInvalidRequest, "invalid role")
		return
	}
	id := common.PathID(r, "id")
	if err := h.queries.UpdateTeamMemberRole(r.Context(), db.UpdateTeamMemberRoleParams{MerchantID: merchantID, UserID: id, Role: req.Role}); err != nil {
		apierr.Internal(w, err)
		return
	}
	member, err := h.queries.GetTeamMember(r.Context(), db.GetTeamMemberParams{MerchantID: merchantID, UserID: id})
	if errors.Is(err, sql.ErrNoRows) {
		apierr.NotFound(w)
		return
	}
	if err != nil {
		apierr.Internal(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, member)
}
func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := common.MerchantID(r)
	if !ok {
		apierr.Unauthorized(w)
		return
	}
	id := common.PathID(r, "id")
	if err := h.queries.RemoveTeamMember(r.Context(), db.RemoveTeamMemberParams{MerchantID: merchantID, UserID: id}); err != nil {
		apierr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func validRole(role string) bool {
	return role == roles.RoleOwner || role == roles.RoleAdmin || role == roles.RoleStaff
}
