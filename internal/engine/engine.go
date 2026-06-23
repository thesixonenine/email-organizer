package engine

import (
	"log/slog"

	"email-organizer/internal/mail/types"
	"email-organizer/internal/models"
	"email-organizer/internal/store"
)

type RuleEngine struct {
	ruleRepo *store.RuleRepository
}

func NewRuleEngine(ruleRepo *store.RuleRepository) *RuleEngine {
	return &RuleEngine{ruleRepo: ruleRepo}
}

func (e *RuleEngine) ProcessMessage(client types.MailClient, mailboxID int64, msg *models.Message) error {
	rules, err := e.ruleRepo.ListByTrigger("on_arrival")
	if err != nil {
		return err
	}

	for _, rule := range rules {
		cond, err := rule.GetCondition()
		if err != nil {
			slog.Warn("parse condition", "rule", rule.ID, "error", err)
			continue
		}

		if !MatchMessage(cond, msg) {
			continue
		}

		action, err := rule.GetAction()
		if err != nil {
			slog.Warn("parse action", "rule", rule.ID, "error", err)
			continue
		}

		if err := ExecuteAction(client, action, msg); err != nil {
			slog.Error("execute action failed", "rule", rule.ID, "error", err)
			e.ruleRepo.CreateLog(&models.RuleLog{
				RuleID:    rule.ID,
				MailboxID: mailboxID,
				MessageID: &msg.ID,
				Action:    action.Type,
				Result:    "failed",
				ErrorMsg:  err.Error(),
			})
		} else {
			slog.Info("action executed", "rule", rule.ID, "action", action.Type, "msg", msg.UID)
			e.ruleRepo.CreateLog(&models.RuleLog{
				RuleID:    rule.ID,
				MailboxID: mailboxID,
				MessageID: &msg.ID,
				Action:    action.Type,
				Result:    "success",
			})
		}
	}
	return nil
}