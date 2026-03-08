package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/rerolldrinks/messageservice/models"
)

type SQLService interface {
	GetActiveMessages(ctx context.Context) ([]models.MessageRecord, error)
	UpdateLastRunUTC(ctx context.Context, id int, lastRunUTC time.Time) error
}

type sqlService struct {
	db *sql.DB
}

func NewSQLService(db *sql.DB) SQLService {
	return &sqlService{db: db}
}

// GetActiveMessages returns all messages that are active, have started, and have not expired.
// Interval/LastRunUTC filtering is done in the caller so the cache stays simple.
const queryActiveMessages = `
SELECT Id, ClientId, Message, IsActive, StartUTC, ExpiresUTC, LastRunUTC, IntervalMinutes
FROM Messages
WHERE IsActive = 1
  AND StartUTC <= GETUTCDATE()
  AND (ExpiresUTC IS NULL OR ExpiresUTC > GETUTCDATE())
`

func (s *sqlService) GetActiveMessages(ctx context.Context) ([]models.MessageRecord, error) {
	rows, err := s.db.QueryContext(ctx, queryActiveMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to query active messages: %w", err)
	}
	defer rows.Close()

	var messages []models.MessageRecord
	for rows.Next() {
		var msg models.MessageRecord
		var clientId sql.NullString
		var expiresUTC sql.NullTime
		var lastRunUTC sql.NullTime

		err := rows.Scan(
			&msg.Id,
			&clientId,
			&msg.Message,
			&msg.IsActive,
			&msg.StartUTC,
			&expiresUTC,
			&lastRunUTC,
			&msg.IntervalMinutes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}

		if clientId.Valid {
			msg.ClientId = &clientId.String
		}
		if expiresUTC.Valid {
			msg.ExpiresUTC = &expiresUTC.Time
		}
		if lastRunUTC.Valid {
			msg.LastRunUTC = &lastRunUTC.Time
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func (s *sqlService) UpdateLastRunUTC(ctx context.Context, id int, lastRunUTC time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE Messages SET LastRunUTC = @p1 WHERE Id = @p2",
		lastRunUTC, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update LastRunUTC for message %d: %w", id, err)
	}
	return nil
}
