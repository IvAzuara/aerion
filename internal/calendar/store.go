package calendar

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hkdb/aerion/internal/database"
)

// Store handles database operations for calendars and events
type Store struct {
	db *database.DB
}

// NewStore creates a new calendar store
func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

// ListCalendars returns all calendars for an account
func (s *Store) ListCalendars(accountID string) ([]*Calendar, error) {
	rows, err := s.db.Query(`
		SELECT id, account_id, name, color, type, access_role, enabled, sync_token, last_synced_at, created_at, updated_at
		FROM calendars
		WHERE account_id = ?
		ORDER BY name ASC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calendars []*Calendar
	for rows.Next() {
		c := &Calendar{}
		var lastSyncedAt sql.NullTime
		if err := rows.Scan(
			&c.ID, &c.AccountID, &c.Name, &c.Color, &c.Type, &c.AccessRole, &c.Enabled,
			&c.SyncToken, &lastSyncedAt, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		c.LastSyncedAt = lastSyncedAt.Time
		calendars = append(calendars, c)
	}
	return calendars, nil
}

// GetCalendar returns a calendar by ID
func (s *Store) GetCalendar(id string) (*Calendar, error) {
	c := &Calendar{}
	var lastSyncedAt sql.NullTime
	err := s.db.QueryRow(`
		SELECT id, account_id, name, color, type, access_role, enabled, sync_token, last_synced_at, created_at, updated_at
		FROM calendars
		WHERE id = ?
	`, id).Scan(
		&c.ID, &c.AccountID, &c.Name, &c.Color, &c.Type, &c.AccessRole, &c.Enabled,
		&c.SyncToken, &lastSyncedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.LastSyncedAt = lastSyncedAt.Time
	return c, nil
}

// UpsertCalendar creates or updates a calendar
func (s *Store) UpsertCalendar(c *Calendar) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT INTO calendars (id, account_id, name, color, type, access_role, enabled, sync_token, last_synced_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			color = excluded.color,
			access_role = excluded.access_role,
			enabled = excluded.enabled,
			sync_token = excluded.sync_token,
			last_synced_at = excluded.last_synced_at,
			updated_at = excluded.updated_at
	`, c.ID, c.AccountID, c.Name, c.Color, c.Type, c.AccessRole, c.Enabled, c.SyncToken, c.LastSyncedAt, c.CreatedAt, c.UpdatedAt)
	return err
}

// ListEvents returns events for a calendar within a time range
func (s *Store) ListEvents(calendarID string, start, end time.Time) ([]*Event, error) {
	rows, err := s.db.Query(`
		SELECT id, calendar_id, remote_id, summary, description, location, start_time, end_time, is_all_day, recurrence, meet_link, status, created_at, updated_at
		FROM calendar_events
		WHERE calendar_id = ? AND ((start_time >= ? AND start_time <= ?) OR (end_time >= ? AND end_time <= ?))
		ORDER BY start_time ASC
	`, calendarID, start, end, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		e := &Event{}
		if err := rows.Scan(
			&e.ID, &e.CalendarID, &e.RemoteID, &e.Summary, &e.Description, &e.Location,
			&e.StartTime, &e.EndTime, &e.IsAllDay, &e.Recurrence, &e.MeetLink, &e.Status,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		// Load attendees
		attendees, err := s.ListAttendees(e.ID)
		if err != nil {
			return nil, err
		}
		e.Attendees = attendees
		events = append(events, e)
	}
	return events, nil
}

// UpsertEvent creates or updates an event and its attendees
func (s *Store) UpsertEvent(e *Event) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now

	_, err = tx.Exec(`
		INSERT INTO calendar_events (id, calendar_id, remote_id, summary, description, location, start_time, end_time, is_all_day, recurrence, meet_link, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			summary = excluded.summary,
			description = excluded.description,
			location = excluded.location,
			start_time = excluded.start_time,
			end_time = excluded.end_time,
			is_all_day = excluded.is_all_day,
			recurrence = excluded.recurrence,
			meet_link = excluded.meet_link,
			status = excluded.status,
			updated_at = excluded.updated_at
	`, e.ID, e.CalendarID, e.RemoteID, e.Summary, e.Description, e.Location, e.StartTime, e.EndTime, e.IsAllDay, e.Recurrence, e.MeetLink, e.Status, e.CreatedAt, e.UpdatedAt)
	if err != nil {
		return err
	}

	// Update attendees: delete existing and insert new
	_, err = tx.Exec("DELETE FROM calendar_attendees WHERE event_id = ?", e.ID)
	if err != nil {
		return err
	}

	for _, a := range e.Attendees {
		if a.ID == "" {
			a.ID = uuid.New().String()
		}
		_, err = tx.Exec(`
			INSERT INTO calendar_attendees (id, event_id, email, name, status, is_organizer)
			VALUES (?, ?, ?, ?, ?, ?)
		`, a.ID, e.ID, a.Email, a.Name, a.Status, a.IsOrganizer)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListAttendees returns all attendees for an event
func (s *Store) ListAttendees(eventID string) ([]Attendee, error) {
	rows, err := s.db.Query(`
		SELECT id, event_id, email, name, status, is_organizer
		FROM calendar_attendees
		WHERE event_id = ?
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attendees []Attendee
	for rows.Next() {
		a := Attendee{}
		if err := rows.Scan(&a.ID, &a.EventID, &a.Email, &a.Name, &a.Status, &a.IsOrganizer); err != nil {
			return nil, err
		}
		attendees = append(attendees, a)
	}
	return attendees, nil
}

// DeleteEvent removes an event and its attendees
func (s *Store) DeleteEvent(id string) error {
	_, err := s.db.Exec("DELETE FROM calendar_events WHERE id = ?", id)
	return err
}

// DeleteCalendar removes a calendar and all its events
func (s *Store) DeleteCalendar(id string) error {
	_, err := s.db.Exec("DELETE FROM calendars WHERE id = ?", id)
	return err
}

// GetEvent returns a single event by ID
func (s *Store) GetEvent(id string) (*Event, error) {
	e := &Event{}
	err := s.db.QueryRow(`
		SELECT id, calendar_id, remote_id, summary, description, location, start_time, end_time, is_all_day, recurrence, meet_link, status, created_at, updated_at
		FROM calendar_events
		WHERE id = ?
	`, id).Scan(
		&e.ID, &e.CalendarID, &e.RemoteID, &e.Summary, &e.Description, &e.Location,
		&e.StartTime, &e.EndTime, &e.IsAllDay, &e.Recurrence, &e.MeetLink, &e.Status,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Load attendees
	attendees, err := s.ListAttendees(e.ID)
	if err != nil {
		return nil, err
	}
	e.Attendees = attendees
	return e, nil
}

// GetEventByRemoteID returns a single event by its remote ID and calendar ID
func (s *Store) GetEventByRemoteID(calendarID, remoteID string) (*Event, error) {
	e := &Event{}
	err := s.db.QueryRow(`
		SELECT id, calendar_id, remote_id, summary, description, location, start_time, end_time, is_all_day, recurrence, meet_link, status, created_at, updated_at
		FROM calendar_events
		WHERE calendar_id = ? AND remote_id = ?
	`, calendarID, remoteID).Scan(
		&e.ID, &e.CalendarID, &e.RemoteID, &e.Summary, &e.Description, &e.Location,
		&e.StartTime, &e.EndTime, &e.IsAllDay, &e.Recurrence, &e.MeetLink, &e.Status,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}
