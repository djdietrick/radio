package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/djdietrick/radio/internal/auth"
	"github.com/djdietrick/radio/internal/users"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResp struct {
	Token   string `json:"token"`
	UserID  string `json:"userId"`
	IsAdmin bool   `json:"isAdmin"`
}

// Login validates credentials and returns a signed JWT. It returns 401 for both
// unknown users and bad passwords (no account enumeration).
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}

	u, err := h.d.Users.ByUsername(r.Context(), req.Username)
	if err != nil || !auth.CheckPassword(u.PasswordHash, req.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := h.d.Auth.Issue(u.ID, u.IsAdmin)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	writeJSON(w, http.StatusOK, loginResp{Token: token, UserID: u.ID, IsAdmin: u.IsAdmin})
}

// Me returns the current authenticated user.
func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	u, err := h.d.Users.ByID(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

type createUserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"isAdmin"`
}

// CreateUser provisions a new account. Admin-only (enforced by route middleware).
func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeErr(w, http.StatusBadRequest, "username and password are required")
		return
	}

	u, err := h.d.Users.Create(r.Context(), req.Username, req.Password, req.IsAdmin)
	if err == users.ErrDuplicate {
		writeErr(w, http.StatusConflict, "username already exists")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

// ListUsers returns all accounts. Admin-only.
func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	us, err := h.d.Users.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, us)
}
