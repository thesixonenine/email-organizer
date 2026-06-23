package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"email-organizer/internal/models"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) ListByFolder(mailboxID int64, folder string, page, pageSize int) ([]*models.Message, int, error) {
	countQuery := `SELECT COUNT(*) FROM messages WHERE mailbox_id=? AND folder=? AND is_deleted=0`
	var total int
	if err := r.db.QueryRow(countQuery, mailboxID, folder).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count messages: %w", err)
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(`SELECT id, mailbox_id, uid, folder,
		COALESCE(subject,''), COALESCE(body_preview,''), COALESCE(body,''),
		COALESCE(from_name,''), COALESCE(from_email,''),
		COALESCE(to_list,''), COALESCE(cc_list,''),
		received_at, COALESCE(flags,''), is_deleted, created_at
		FROM messages WHERE mailbox_id=? AND folder=? AND is_deleted=0
		ORDER BY received_at DESC LIMIT ? OFFSET ?`, mailboxID, folder, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var result []*models.Message
	for rows.Next() {
		m := &models.Message{}
		var toListStr, ccListStr, flagsStr string
		var receivedAtStr, createdAtStr string
		if err := rows.Scan(&m.ID, &m.MailboxID, &m.UID, &m.Folder,
			&m.Subject, &m.BodyPreview, &m.Body,
			&m.FromName, &m.FromEmail,
			&toListStr, &ccListStr,
			&receivedAtStr, &flagsStr, &m.IsDeleted, &createdAtStr); err != nil {
			return nil, 0, fmt.Errorf("scan message: %w", err)
		}
		json.Unmarshal([]byte(toListStr), &m.ToList)
		json.Unmarshal([]byte(ccListStr), &m.CcList)
		json.Unmarshal([]byte(flagsStr), &m.Flags)
		m.ReceivedAt, _ = time.Parse(time.RFC3339, receivedAtStr)
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		result = append(result, m)
	}
	return result, total, nil
}

func (r *MessageRepository) GetByID(id int64) (*models.Message, error) {
	row := r.db.QueryRow(`SELECT id, mailbox_id, uid, folder,
		COALESCE(subject,''), COALESCE(body_preview,''), COALESCE(body,''),
		COALESCE(from_name,''), COALESCE(from_email,''),
		COALESCE(to_list,''), COALESCE(cc_list,''),
		received_at, COALESCE(flags,''), is_deleted, created_at
		FROM messages WHERE id=?`, id)
	m := &models.Message{}
	var toListStr, ccListStr, flagsStr string
	var receivedAtStr, createdAtStr string
	if err := row.Scan(&m.ID, &m.MailboxID, &m.UID, &m.Folder,
		&m.Subject, &m.BodyPreview, &m.Body,
		&m.FromName, &m.FromEmail,
		&toListStr, &ccListStr,
		&receivedAtStr, &flagsStr, &m.IsDeleted, &createdAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message %d not found", id)
		}
		return nil, fmt.Errorf("get message %d: %w", id, err)
	}
	json.Unmarshal([]byte(toListStr), &m.ToList)
	json.Unmarshal([]byte(ccListStr), &m.CcList)
	json.Unmarshal([]byte(flagsStr), &m.Flags)
	m.ReceivedAt, _ = time.Parse(time.RFC3339, receivedAtStr)
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	return m, nil
}

func (r *MessageRepository) Insert(msg *models.Message) (int64, error) {
	toListJSON, _ := json.Marshal(msg.ToList)
	ccListJSON, _ := json.Marshal(msg.CcList)
	flagsJSON, _ := json.Marshal(msg.Flags)
	now := time.Now().Format(time.RFC3339)

	result, err := r.db.Exec(`INSERT OR IGNORE INTO messages
		(mailbox_id, uid, folder, subject, body_preview, body,
		 from_name, from_email, to_list, cc_list,
		 received_at, flags, is_deleted, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		msg.MailboxID, msg.UID, msg.Folder,
		msg.Subject, msg.BodyPreview, msg.Body,
		msg.FromName, msg.FromEmail,
		string(toListJSON), string(ccListJSON),
		msg.ReceivedAt.Format(time.RFC3339),
		string(flagsJSON), msg.IsDeleted, now)
	if err != nil {
		return 0, fmt.Errorf("insert message: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

func (r *MessageRepository) BatchInsert(msgs []*models.Message) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO messages
		(mailbox_id, uid, folder, subject, body_preview, body,
		 from_name, from_email, to_list, cc_list,
		 received_at, flags, is_deleted, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	now := time.Now().Format(time.RFC3339)
	for _, msg := range msgs {
		toListJSON, _ := json.Marshal(msg.ToList)
		ccListJSON, _ := json.Marshal(msg.CcList)
		flagsJSON, _ := json.Marshal(msg.Flags)
		if _, err := stmt.Exec(
			msg.MailboxID, msg.UID, msg.Folder,
			msg.Subject, msg.BodyPreview, msg.Body,
			msg.FromName, msg.FromEmail,
			string(toListJSON), string(ccListJSON),
			msg.ReceivedAt.Format(time.RFC3339),
			string(flagsJSON), msg.IsDeleted, now); err != nil {
			return fmt.Errorf("insert msg %s: %w", msg.UID, err)
		}
	}
	return tx.Commit()
}

func (r *MessageRepository) MarkDeleted(id int64) error {
	_, err := r.db.Exec(`UPDATE messages SET is_deleted=1 WHERE id=?`, id)
	return err
}