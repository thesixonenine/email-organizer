package store

import (
	"database/sql"
	"fmt"
	"time"

	"email-organizer/internal/models"
)

type RuleRepository struct {
	db *sql.DB
}

func NewRuleRepository(db *sql.DB) *RuleRepository {
	return &RuleRepository{db: db}
}

func (r *RuleRepository) List() ([]*models.Rule, error) {
	rows, err := r.db.Query(`SELECT id, name, enabled, trigger,
		condition_json, action_json, priority, created_at, updated_at
		FROM rules ORDER BY priority ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	var result []*models.Rule
	for rows.Next() {
		rule := &models.Rule{}
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Enabled, &rule.Trigger,
			&rule.ConditionJSON, &rule.ActionJSON, &rule.Priority,
			&rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		result = append(result, rule)
	}
	return result, nil
}

func (r *RuleRepository) GetByID(id int64) (*models.Rule, error) {
	row := r.db.QueryRow(`SELECT id, name, enabled, trigger,
		condition_json, action_json, priority, created_at, updated_at
		FROM rules WHERE id=?`, id)
	rule := &models.Rule{}
	if err := row.Scan(&rule.ID, &rule.Name, &rule.Enabled, &rule.Trigger,
		&rule.ConditionJSON, &rule.ActionJSON, &rule.Priority,
		&rule.CreatedAt, &rule.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rule %d not found", id)
		}
		return nil, fmt.Errorf("get rule %d: %w", id, err)
	}
	return rule, nil
}

func (r *RuleRepository) Create(rule *models.Rule) (int64, error) {
	now := time.Now().Format(time.RFC3339)
	result, err := r.db.Exec(`INSERT INTO rules
		(name, enabled, trigger, condition_json, action_json, priority, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		rule.Name, rule.Enabled, rule.Trigger,
		rule.ConditionJSON, rule.ActionJSON, rule.Priority, now, now)
	if err != nil {
		return 0, fmt.Errorf("create rule: %w", err)
	}
	return result.LastInsertId()
}

func (r *RuleRepository) Update(rule *models.Rule) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec(`UPDATE rules SET
		name=?, enabled=?, trigger=?, condition_json=?, action_json=?,
		priority=?, updated_at=?
		WHERE id=?`,
		rule.Name, rule.Enabled, rule.Trigger,
		rule.ConditionJSON, rule.ActionJSON,
		rule.Priority, now, rule.ID)
	return err
}

func (r *RuleRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM rules WHERE id=?`, id)
	return err
}

func (r *RuleRepository) SetEnabled(id int64, enabled bool) error {
	_, err := r.db.Exec(`UPDATE rules SET enabled=?, updated_at=? WHERE id=?`,
		enabled, time.Now().Format(time.RFC3339), id)
	return err
}

func (r *RuleRepository) ListByTrigger(trigger string) ([]*models.Rule, error) {
	rows, err := r.db.Query(`SELECT id, name, enabled, trigger,
		condition_json, action_json, priority, created_at, updated_at
		FROM rules WHERE trigger=? AND enabled=1
		ORDER BY priority ASC, id ASC`, trigger)
	if err != nil {
		return nil, fmt.Errorf("list rules by trigger: %w", err)
	}
	defer rows.Close()

	var result []*models.Rule
	for rows.Next() {
		rule := &models.Rule{}
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Enabled, &rule.Trigger,
			&rule.ConditionJSON, &rule.ActionJSON, &rule.Priority,
			&rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		result = append(result, rule)
	}
	return result, nil
}

func (r *RuleRepository) CreateLog(log *models.RuleLog) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec(`INSERT INTO rule_logs
		(rule_id, mailbox_id, message_id, action, result, error_msg, created_at)
		VALUES (?,?,?,?,?,?,?)`,
		log.RuleID, log.MailboxID, log.MessageID,
		log.Action, log.Result, log.ErrorMsg, now)
	return err
}