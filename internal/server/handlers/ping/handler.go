package ping

import (
	"context"
	"net/http"
	"time"
)

// Pinger is the subset of database/sql.DB used by the health check endpoint.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Handler serves the database health check endpoint.
type Handler struct {
	db Pinger
}

// New creates a ping handler. A nil database makes the endpoint always healthy.
func New(db Pinger) *Handler {
	return &Handler{db: db}
}

// Ping writes OK when the configured database responds within one second.
func (h Handler) Ping(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
