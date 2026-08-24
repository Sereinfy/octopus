package handlers

import (
	"net/http"

	"github.com/bestruirui/octopus/internal/relay"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/router"
)

func init() {
	router.NewGroupRouter("/v1").
		Use(middleware.APIKeyAuth(), middleware.ImageBodyLimit()).
		AddRoute(
			router.NewRoute("/images/generations", http.MethodPost).
				Handle(relay.HandleImageGenerations),
		).
		AddRoute(
			router.NewRoute("/images/edits", http.MethodPost).
				Handle(relay.HandleImageEdits),
		).
		AddRoute(
			router.NewRoute("/images/variations", http.MethodPost).
				Handle(relay.HandleImageVariations),
		)
}
