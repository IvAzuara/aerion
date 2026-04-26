package calendar

import (
	"fmt"
	"time"

	"github.com/hkdb/aerion/internal/logging"
	"github.com/rs/zerolog"
)

// AccessTokenGetter is a function that retrieves a valid OAuth access token for an account
type AccessTokenGetter func(accountID string) (string, error)

// Syncer handles syncing calendars and events from Google
type Syncer struct {
	store           *Store
	googleClient    *GoogleCalendarClient
	getAccountToken AccessTokenGetter
	log             zerolog.Logger
}

// NewSyncer creates a new calendar syncer
func NewSyncer(store *Store) *Syncer {
	return &Syncer{
		store:        store,
		googleClient: NewGoogleCalendarClient(),
		log:          logging.WithComponent("calendar-sync"),
	}
}

// SetAccessTokenGetter sets the function for retrieving OAuth access tokens
func (s *Syncer) SetAccessTokenGetter(getter AccessTokenGetter) {
	s.getAccountToken = getter
}

// SyncAccount syncs all calendars and events for a given account
func (s *Syncer) SyncAccount(accountID string) error {
	s.log.Info().Str("accountID", accountID).Msg("Starting account calendar sync")

	if s.getAccountToken == nil {
		return fmt.Errorf("access token getter not configured")
	}

	accessToken, err := s.getAccountToken(accountID)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// 1. Sync Calendar List
	googleCalendars, err := s.googleClient.ListCalendars(accessToken)
	if err != nil {
		return fmt.Errorf("failed to list calendars from google: %w", err)
	}

	for _, gc := range googleCalendars {
		gc.AccountID = accountID
		if err := s.store.UpsertCalendar(gc); err != nil {
			s.log.Error().Err(err).Str("calendar", gc.Name).Msg("Failed to store calendar")
			continue
		}

		// 2. Sync Events for each calendar (if enabled)
		if gc.Enabled {
			if err := s.SyncCalendarEvents(accountID, gc.ID); err != nil {
				s.log.Error().Err(err).Str("calendar", gc.Name).Msg("Failed to sync events")
			}
		}
	}

	return nil
}

// SyncCalendarEvents syncs events for a specific calendar
func (s *Syncer) SyncCalendarEvents(accountID, calendarID string) error {
	accessToken, err := s.getAccountToken(accountID)
	if err != nil {
		return err
	}

	// Range: 30 days ago to 1 year ahead
	timeMin := time.Now().AddDate(0, 0, -30)
	timeMax := time.Now().AddDate(1, 0, 0)

	events, err := s.googleClient.ListEvents(accessToken, calendarID, timeMin, timeMax)
	if err != nil {
		return err
	}

	s.log.Info().Str("calendarID", calendarID).Int("count", len(events)).Msg("Fetched events from Google")

	for _, e := range events {
		// Try to find existing event by remote ID to avoid duplication
		if e.RemoteID != "" {
			existing, err := s.store.GetEventByRemoteID(calendarID, e.RemoteID)
			if err == nil && existing != nil {
				e.ID = existing.ID
			}
		}

		if err := s.store.UpsertEvent(e); err != nil {
			s.log.Error().Err(err).Str("event", e.Summary).Msg("Failed to store event")
		}
	}

	// Update last synced at
	cal, _ := s.store.GetCalendar(calendarID)
	if cal != nil {
		cal.LastSyncedAt = time.Now()
		s.store.UpsertCalendar(cal)
	}

	return nil
}
