package chimiddleware

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const heartbeatPattern = "/health"

func WithHeartbeat(r chi.Router) {
	r.Use(middleware.Heartbeat(heartbeatPattern))
	r.Get(heartbeatPattern, nil)
}
