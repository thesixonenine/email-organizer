package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"email-organizer/internal/mail/types"
	"email-organizer/internal/store"
)

type MailboxHandler struct {
	mailboxRepo *store.MailboxRepository
	msgRepo     *store.MessageRepository
	mailFactory func(id int64) (types.MailClient, error)
}

func NewMailboxHandler(mbRepo *store.MailboxRepository, msgRepo *store.MessageRepository,
	mf func(int64) (types.MailClient, error)) *MailboxHandler {
	return &MailboxHandler{
		mailboxRepo: mbRepo,
		msgRepo:     msgRepo,
		mailFactory: mf,
	}
}

func (h *MailboxHandler) ListMailboxes(w http.ResponseWriter, r *http.Request) {
	mailboxes, err := h.mailboxRepo.List()
	if err != nil {
		Error(w, 500, err.Error())
		return
	}
	// Mask auth codes in response
	for _, mb := range mailboxes {
		// Do not expose auth_code — it's not in the Mailbox model
		_ = mb
	}
	JSON(w, 200, mailboxes)
}

func (h *MailboxHandler) ListFolders(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		Error(w, 400, "invalid mailbox id")
		return
	}
	client, err := h.mailFactory(id)
	if err != nil {
		Error(w, 500, err.Error())
		return
	}
	folders, err := client.ListFolders()
	if err != nil {
		Error(w, 500, err.Error())
		return
	}
	JSON(w, 200, folders)
}

func (h *MailboxHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		Error(w, 400, "invalid mailbox id")
		return
	}
	folder := chi.URLParam(r, "folder")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	messages, total, err := h.msgRepo.ListByFolder(id, folder, page, pageSize)
	if err != nil {
		Error(w, 500, err.Error())
		return
	}
	JSON(w, 200, map[string]interface{}{
		"messages": messages,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *MailboxHandler) GetMessage(w http.ResponseWriter, r *http.Request) {
	msgID, err := strconv.ParseInt(chi.URLParam(r, "msgId"), 10, 64)
	if err != nil {
		Error(w, 400, "invalid message id")
		return
	}
	msg, err := h.msgRepo.GetByID(msgID)
	if err != nil {
		Error(w, 404, "message not found")
		return
	}
	JSON(w, 200, msg)
}