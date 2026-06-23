package store

import (
	"database/sql"
	"fmt"
	"time"

	"email-organizer/internal/models"
)

type MailboxRepository struct {
	db *sql.DB
}

func NewMailboxRepository(db *sql.DB) *MailboxRepository {
	return &MailboxRepository{db: db}
}

func (r *MailboxRepository) List() ([]*models.Mailbox, error) {
	rows, err := r.db.Query(`SELECT id, name, protocol, email,
		COALESCE(imap_server,''), COALESCE(imap_port,993),
		COALESCE(exchange_type,''), COALESCE(ews_endpoint,''),
		COALESCE(tenant_id,''), COALESCE(client_id,''),
		fetch_count, last_sync_at, is_active, created_at, updated_at FROM mailboxes ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list mailboxes: %w", err)
	}
	defer rows.Close()

	var result []*models.Mailbox
	for rows.Next() {
		m := &models.Mailbox{}
		var lastSync sql.NullString
		if err := rows.Scan(&m.ID, &m.Name, &m.Protocol, &m.Email,
			&m.IMAPServer, &m.IMAPPort, &m.ExchangeType, &m.EWSEndpoint,
			&m.TenantID, &m.ClientID, &m.FetchCount, &lastSync, &m.IsActive,
			&m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan mailbox: %w", err)
		}
		if lastSync.Valid {
			t, _ := time.Parse(time.RFC3339, lastSync.String)
			m.LastSyncAt = &t
		}
		result = append(result, m)
	}
	return result, nil
}

func (r *MailboxRepository) GetByID(id int64) (*models.Mailbox, error) {
	row := r.db.QueryRow(`SELECT id, name, protocol, email,
		COALESCE(imap_server,''), COALESCE(imap_port,993),
		COALESCE(exchange_type,''), COALESCE(ews_endpoint,''),
		COALESCE(tenant_id,''), COALESCE(client_id,''),
		fetch_count, last_sync_at, is_active, created_at, updated_at FROM mailboxes WHERE id=?`, id)
	m := &models.Mailbox{}
	var lastSync sql.NullString
	if err := row.Scan(&m.ID, &m.Name, &m.Protocol, &m.Email,
		&m.IMAPServer, &m.IMAPPort, &m.ExchangeType, &m.EWSEndpoint,
		&m.TenantID, &m.ClientID, &m.FetchCount, &lastSync, &m.IsActive,
		&m.CreatedAt, &m.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("mailbox %d not found", id)
		}
		return nil, fmt.Errorf("get mailbox %d: %w", id, err)
	}
	return m, nil
}

func (r *MailboxRepository) Upsert(mailbox *models.Mailbox) (int64, error) {
	now := time.Now().Format(time.RFC3339)
	result, err := r.db.Exec(`INSERT INTO mailboxes
		(name, protocol, email, imap_server, imap_port, exchange_type,
		 ews_endpoint, tenant_id, client_id, fetch_count, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(email) DO UPDATE SET
			name=excluded.name, protocol=excluded.protocol,
			imap_server=excluded.imap_server, imap_port=excluded.imap_port,
			exchange_type=excluded.exchange_type, ews_endpoint=excluded.ews_endpoint,
			tenant_id=excluded.tenant_id, client_id=excluded.client_id,
			fetch_count=excluded.fetch_count, is_active=excluded.is_active,
			updated_at=excluded.updated_at`,
		mailbox.Name, mailbox.Protocol, mailbox.Email,
		mailbox.IMAPServer, mailbox.IMAPPort, mailbox.ExchangeType,
		mailbox.EWSEndpoint, mailbox.TenantID, mailbox.ClientID,
		mailbox.FetchCount, boolToInt(mailbox.IsActive),
		now, now)
	if err != nil {
		return 0, fmt.Errorf("upsert mailbox: %w", err)
	}
	return result.LastInsertId()
}

func (r *MailboxRepository) UpdateLastSync(id int64, t time.Time) error {
	_, err := r.db.Exec(`UPDATE mailboxes SET last_sync_at=?, updated_at=? WHERE id=?`,
		t.Format(time.RFC3339), time.Now().Format(time.RFC3339), id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}