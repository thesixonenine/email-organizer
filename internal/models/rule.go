package models

import (
	"encoding/json"
	"time"
)

type Rule struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Enabled       bool      `json:"enabled"`
	Trigger       string    `json:"trigger"`
	ConditionJSON string    `json:"condition_json"`
	ActionJSON    string    `json:"action_json"`
	Priority      int       `json:"priority"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type RuleCondition struct {
	SubjectContains *string `json:"subject_contains,omitempty"`
	BodyContains    *string `json:"body_contains,omitempty"`
	SenderEmail     *string `json:"sender_email,omitempty"`
	SenderDomain    *string `json:"sender_domain,omitempty"`
	TimeRangeStart  *string `json:"time_range_start,omitempty"`
	TimeRangeEnd    *string `json:"time_range_end,omitempty"`
}

type RuleAction struct {
	Type      string `json:"type"`
	Target    string `json:"target,omitempty"`
	ReplyText string `json:"reply_text,omitempty"`
}

type RuleLog struct {
	ID        int64     `json:"id"`
	RuleID    int64     `json:"rule_id"`
	MailboxID int64     `json:"mailbox_id"`
	MessageID *int64    `json:"message_id,omitempty"`
	Action    string    `json:"action"`
	Result    string    `json:"result"`
	ErrorMsg  string    `json:"error_msg,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *Rule) GetCondition() (*RuleCondition, error) {
	var c RuleCondition
	if err := json.Unmarshal([]byte(r.ConditionJSON), &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Rule) GetAction() (*RuleAction, error) {
	var a RuleAction
	if err := json.Unmarshal([]byte(r.ActionJSON), &a); err != nil {
		return nil, err
	}
	return &a, nil
}