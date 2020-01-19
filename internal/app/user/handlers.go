package user

import (
	"context"
	"encoding/json"
	"net/http"

	"praslar.com/gotasma/internal/app/types"
	"praslar.com/gotasma/internal/pkg/http/respond"
)

type (
	service interface {
		Register(ctx context.Context, req *types.RegisterRequest) (*types.User, error)
	}
	Handler struct {
		srv service
	}
)

func NewHandler(srv service) *Handler {
	return &Handler{
		srv: srv,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req types.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	user, err := h.srv.Register(r.Context(), &req)
	if err != nil {
		respond.Error(w, err, http.StatusInternalServerError)
		return
	}
	respond.JSON(w, http.StatusOK, types.BaseResponse{
		Data: user,
	})
}
