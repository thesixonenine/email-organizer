package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(mbHandler *MailboxHandler, ruleHandler *RuleHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(RecoveryMiddleware)
	r.Use(LoggingMiddleware)

	r.Route("/api", func(r chi.Router) {
		r.Get("/mailboxes", mbHandler.ListMailboxes)
		r.Get("/mailboxes/{id}/folders", mbHandler.ListFolders)
		r.Get("/mailboxes/{id}/folders/{folder}/messages", mbHandler.ListMessages)
		r.Get("/mailboxes/{id}/folders/{folder}/messages/{msgId}", mbHandler.GetMessage)

		r.Get("/rules", ruleHandler.ListRules)
		r.Post("/rules", ruleHandler.CreateRule)
		r.Get("/rules/{id}", ruleHandler.GetRule)
		r.Put("/rules/{id}", ruleHandler.UpdateRule)
		r.Delete("/rules/{id}", ruleHandler.DeleteRule)
		r.Patch("/rules/{id}/toggle", ruleHandler.ToggleRule)
	})
	return r
}