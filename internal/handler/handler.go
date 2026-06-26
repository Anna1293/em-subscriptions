package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Anna1293/em-subscriptions/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	repo *repository.Repo
}

func New(repo *repository.Repo) *Handler {
	return &Handler{repo: repo}
}

type createRequest struct {
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      uuid.UUID `json:"user_id"`
	StartDate   string    `json:"start_date"`
	EndDate     *string   `json:"end_date"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validate(req.ServiceName, req.Price, req.StartDate); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	sub, err := h.repo.Create(r.Context(), repository.Subscription{
		ServiceName: strings.TrimSpace(req.ServiceName),
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	})
	if err != nil {
		log.Printf("create error: %v", err)
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}

	log.Printf("created subscription id=%d user=%s", sub.ID, sub.UserID)
	writeJSON(w, http.StatusCreated, sub)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	sub, err := h.repo.Get(r.Context(), id)
	if err == sql.ErrNoRows {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		log.Printf("get error: %v", err)
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	f := repository.Filter{}

	if v := r.URL.Query().Get("user_id"); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		f.UserID = &uid
	}
	if v := r.URL.Query().Get("service_name"); v != "" {
		f.ServiceName = &v
	}

	list, err := h.repo.List(r.Context(), f)
	if err != nil {
		log.Printf("list error: %v", err)
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}
	if list == nil {
		list = []repository.Subscription{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validate(req.ServiceName, req.Price, req.StartDate); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.repo.Update(r.Context(), id, repository.Subscription{
		ServiceName: strings.TrimSpace(req.ServiceName),
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	})
	if err == sql.ErrNoRows {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		log.Printf("update error: %v", err)
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}

	sub, _ := h.repo.Get(r.Context(), id)
	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err == sql.ErrNoRows {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		log.Printf("delete error: %v", err)
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Total(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		writeErr(w, http.StatusBadRequest, "from and to are required (MM-YYYY)")
		return
	}

	fromT, err := repository.ParseMonth(from)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	toT, err := repository.ParseMonth(to)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	pf := repository.PeriodFilter{From: fromT, To: toT}

	if v := r.URL.Query().Get("user_id"); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		pf.UserID = &uid
	}
	if v := r.URL.Query().Get("service_name"); v != "" {
		pf.ServiceName = &v
	}

	total, err := h.repo.Total(r.Context(), pf)
	if err != nil {
		log.Printf("total error: %v", err)
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"total": total})
}

func validate(name string, price int, startDate string) error {
	if strings.TrimSpace(name) == "" {
		return errText("service_name is required")
	}
	if price <= 0 {
		return errText("price must be > 0")
	}
	if _, err := repository.ParseMonth(startDate); err != nil {
		return err
	}
	return nil
}

type errText string

func (e errText) Error() string { return string(e) }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
