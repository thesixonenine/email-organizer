package engine

import (
	"fmt"

	"email-organizer/internal/mail/types"
	"email-organizer/internal/models"
)

func ExecuteAction(client types.MailClient, action *models.RuleAction, msg *models.Message) error {
	switch action.Type {
	case "move":
		return client.MoveMessage(msg.Folder, msg.UID, action.Target)
	case "mark_read":
		return client.MarkRead(msg.Folder, msg.UID)
	case "forward":
		return client.ForwardMessage(msg, action.Target)
	case "reply":
		return client.ReplyMessage(msg, action.ReplyText)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}