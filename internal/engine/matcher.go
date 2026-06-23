package engine

import (
	"strings"
	"time"

	"email-organizer/internal/models"
)

func MatchMessage(cond *models.RuleCondition, msg *models.Message) bool {
	if cond.SubjectContains != nil {
		if !strings.Contains(msg.Subject, *cond.SubjectContains) {
			return false
		}
	}
	if cond.BodyContains != nil {
		if !strings.Contains(msg.Body, *cond.BodyContains) {
			return false
		}
	}
	if cond.SenderEmail != nil {
		if msg.FromEmail != *cond.SenderEmail {
			return false
		}
	}
	if cond.SenderDomain != nil {
		parts := strings.Split(msg.FromEmail, "@")
		if len(parts) != 2 || parts[1] != *cond.SenderDomain {
			return false
		}
	}
	if cond.TimeRangeStart != nil || cond.TimeRangeEnd != nil {
		msgTime := msg.ReceivedAt.Format("15:04")
		if cond.TimeRangeStart != nil && msgTime < *cond.TimeRangeStart {
			return false
		}
		if cond.TimeRangeEnd != nil && msgTime > *cond.TimeRangeEnd {
			return false
		}
	}
	return true
}

func parseTime(s string) (time.Time, error) {
	return time.Parse("15:04", s)
}