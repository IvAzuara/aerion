package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/hkdb/aerion/internal/account"
	"github.com/hkdb/aerion/internal/calendar"
	"github.com/hkdb/aerion/internal/logging"
)

// GetCalendars returns all configured calendars across all accounts
func (a *App) GetCalendars() ([]*calendar.Calendar, error) {
	accounts, err := a.accountStore.List()
	if err != nil {
		return nil, err
	}

	var allCalendars []*calendar.Calendar

	for _, acc := range accounts {
		calendars, err := a.calendarStore.ListCalendars(acc.ID)
		if err != nil {
			continue
		}
		
		for _, cal := range calendars {
			// Mark global Google calendars
			if cal.Type == "google" && strings.HasSuffix(cal.ID, "@group.v.calendar.google.com") {
				cal.IsGlobal = true
				
				// Extract canonical ID
				parts := strings.Split(cal.ID, "@")
				canonicalID := parts[0]
				if dotIdx := strings.LastIndex(canonicalID, "."); dotIdx != -1 {
					canonicalID = canonicalID[dotIdx+1:]
				}
				cal.CanonicalID = canonicalID
			}
			
			allCalendars = append(allCalendars, cal)
		}
	}
	return allCalendars, nil
}

// GetCalendarEvents returns events for a specific calendar within a time range
// start and end are expected in ISO8601 format (e.g., "2023-01-01T00:00:00Z")
func (a *App) GetCalendarEvents(calendarID string, startStr, endStr string) ([]*calendar.Event, error) {
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}

	return a.calendarStore.ListEvents(calendarID, start, end)
}

// UpsertCalendarEvent creates or updates a calendar event
func (a *App) UpsertCalendarEvent(event calendar.Event) (*calendar.Event, error) {
	log := logging.WithComponent("app")

	if event.RemoteID != "" {
		return a.UpdateCalendarEvent(event)
	}

	// If it's a Google event, we should ideally sync it to Google too
	cal, err := a.calendarStore.GetCalendar(event.CalendarID)
	if err != nil {
		return nil, err
	}

	if cal != nil && cal.Type == "google" {
		accessToken, err := a.getValidOAuthToken(cal.AccountID)
		if err != nil {
			return nil, fmt.Errorf("failed to get oauth token: %w", err)
		}

		created, err := a.calendarClient.CreateEvent(accessToken.AccessToken, cal.ID, &event)
		if err != nil {
			return nil, fmt.Errorf("google api error: %w", err)
		}
		event = *created
	}

	if err := a.calendarStore.UpsertEvent(&event); err != nil {
		return nil, err
	}

	log.Info().Str("id", event.ID).Str("summary", event.Summary).Msg("Calendar event saved")
	return &event, nil
}

// UpdateCalendarEvent updates an existing calendar event
func (a *App) UpdateCalendarEvent(event calendar.Event) (*calendar.Event, error) {
	log := logging.WithComponent("app")

	cal, err := a.calendarStore.GetCalendar(event.CalendarID)
	if err != nil {
		return nil, err
	}

	if cal != nil && cal.Type == "google" && event.RemoteID != "" {
		accessToken, err := a.getValidOAuthToken(cal.AccountID)
		if err != nil {
			return nil, fmt.Errorf("failed to get oauth token: %w", err)
		}

		updated, err := a.calendarClient.UpdateEvent(accessToken.AccessToken, cal.ID, event.RemoteID, &event)
		if err != nil {
			return nil, fmt.Errorf("google api error: %w", err)
		}
		event = *updated
	}

	if err := a.calendarStore.UpsertEvent(&event); err != nil {
		return nil, err
	}

	log.Info().Str("id", event.ID).Str("summary", event.Summary).Msg("Calendar event updated")
	return &event, nil
}

// DeleteCalendarEvent removes an event
func (a *App) DeleteCalendarEvent(calendarID, eventID string) error {
	log := logging.WithComponent("app")

	cal, err := a.calendarStore.GetCalendar(calendarID)
	if err != nil {
		return err
	}

	if cal != nil && cal.Type == "google" {
		event, err := a.calendarStore.GetEvent(eventID)
		if err == nil && event != nil && event.RemoteID != "" {
			accessToken, err := a.getValidOAuthToken(cal.AccountID)
			if err == nil {
				a.calendarClient.DeleteEvent(accessToken.AccessToken, cal.ID, event.RemoteID)
			}
		}
	}

	if err := a.calendarStore.DeleteEvent(eventID); err != nil {
		return err
	}

	log.Info().Str("id", eventID).Msg("Calendar event deleted")
	return nil
}

// SyncCalendars manually triggers a sync for all accounts
func (a *App) SyncCalendars() error {
	log := logging.WithComponent("app")
	
	accounts, err := a.accountStore.List()
	if err != nil {
		return err
	}

	found := false
	for _, acc := range accounts {
		if acc.AuthType == account.AuthOAuth2 {
			go a.calendarSyncer.SyncAccount(acc.ID)
			found = true
		}
	}

	if !found {
		log.Warn().Msg("No accounts found for calendar sync")
	}

	return nil
}

// SetCalendarEnabled updates the enabled state of a calendar
func (a *App) SetCalendarEnabled(calendarID string, enabled bool) error {
	log := logging.WithComponent("app")
	cal, err := a.calendarStore.GetCalendar(calendarID)
	if err != nil {
		return err
	}
	if cal == nil {
		return fmt.Errorf("calendar not found")
	}

	cal.Enabled = enabled
	if err := a.calendarStore.UpsertCalendar(cal); err != nil {
		return err
	}

	log.Info().Str("id", calendarID).Bool("enabled", enabled).Msg("Calendar enabled state updated")
	
	// If enabled, trigger a sync for this calendar
	if enabled {
		go a.calendarSyncer.SyncCalendarEvents(cal.AccountID, cal.ID)
	}
	
	return nil
}

// SetCalendarsEnabled updates the enabled state for multiple calendars
func (a *App) SetCalendarsEnabled(calendarIDs []string, enabled bool) error {
	log := logging.WithComponent("app")
	for _, id := range calendarIDs {
		cal, err := a.calendarStore.GetCalendar(id)
		if err != nil {
			log.Error().Err(err).Str("id", id).Msg("Failed to get calendar for status update")
			continue
		}
		if cal != nil {
			cal.Enabled = enabled
			if err := a.calendarStore.UpsertCalendar(cal); err != nil {
				log.Error().Err(err).Str("id", id).Msg("Failed to update calendar enabled state")
				continue
			}
			
			// If enabled, trigger a sync for this calendar
			if enabled {
				go a.calendarSyncer.SyncCalendarEvents(cal.AccountID, cal.ID)
			}
		}
	}
	return nil
}
