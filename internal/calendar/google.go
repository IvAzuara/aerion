package calendar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hkdb/aerion/internal/logging"
	"github.com/rs/zerolog"
)

// GoogleCalendarClient interacts with the Google Calendar API v3
type GoogleCalendarClient struct {
	httpClient *http.Client
	log        zerolog.Logger
}

// NewGoogleCalendarClient creates a new Google Calendar API client
func NewGoogleCalendarClient() *GoogleCalendarClient {
	return &GoogleCalendarClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		log:        logging.WithComponent("google-calendar"),
	}
}

// ListCalendars fetches the user's calendar list
func (c *GoogleCalendarClient) ListCalendars(accessToken string) ([]*Calendar, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/calendar/v3/users/me/calendarList", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google api error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []struct {
			ID              string `json:"id"`
			Summary         string `json:"summary"`
			BackgroundColor string `json:"backgroundColor"`
			Primary         bool   `json:"primary"`
			AccessRole      string `json:"accessRole"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	calendars := make([]*Calendar, len(result.Items))
	for i, item := range result.Items {
		name := item.Summary
		if item.Primary {
			name += " (Primary)"
		}
		calendars[i] = &Calendar{
			ID:         item.ID,
			Name:       name,
			Color:      item.BackgroundColor,
			Type:       "google",
			AccessRole: item.AccessRole,
			Enabled:    true,
		}
	}

	return calendars, nil
}

// ListEvents fetches events from a specific calendar
func (c *GoogleCalendarClient) ListEvents(accessToken, calendarID string, timeMin, timeMax time.Time) ([]*Event, error) {
	u, _ := url.Parse(fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events", url.PathEscape(calendarID)))
	q := u.Query()
	q.Set("timeMin", timeMin.Format(time.RFC3339))
	q.Set("timeMax", timeMax.Format(time.RFC3339))
	q.Set("singleEvents", "true")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google api error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []googleEvent `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	events := make([]*Event, len(result.Items))
	for i, ge := range result.Items {
		events[i] = ge.toModel(calendarID)
	}

	return events, nil
}

// CreateEvent creates a new event in Google Calendar
func (c *GoogleCalendarClient) CreateEvent(accessToken, calendarID string, event *Event) (*Event, error) {
	ge := fromModel(event)
	body, _ := json.Marshal(ge)

	u := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events", url.PathEscape(calendarID))
	req, err := http.NewRequest("POST", u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google api error %d: %s", resp.StatusCode, string(body))
	}

	var createdGE googleEvent
	if err := json.NewDecoder(resp.Body).Decode(&createdGE); err != nil {
		return nil, err
	}

	return createdGE.toModel(calendarID), nil
}

// UpdateEvent updates an existing event in Google Calendar using PATCH
func (c *GoogleCalendarClient) UpdateEvent(accessToken, calendarID, remoteID string, event *Event) (*Event, error) {
	ge := fromModel(event)
	body, _ := json.Marshal(ge)

	u := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events/%s", url.PathEscape(calendarID), url.PathEscape(remoteID))
	req, err := http.NewRequest("PATCH", u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google api error %d: %s", resp.StatusCode, string(body))
	}

	var updatedGE googleEvent
	if err := json.NewDecoder(resp.Body).Decode(&updatedGE); err != nil {
		return nil, err
	}

	return updatedGE.toModel(calendarID), nil
}

// DeleteEvent removes an event from Google Calendar
func (c *GoogleCalendarClient) DeleteEvent(accessToken, calendarID, remoteID string) error {
	u := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events/%s", url.PathEscape(calendarID), url.PathEscape(remoteID))
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("google api error: %d", resp.StatusCode)
	}

	return nil
}

// googleEvent internal structures for API mapping

type googleEvent struct {
	ID          string `json:"id,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	Start       struct {
		DateTime string `json:"dateTime,omitempty"`
		Date     string `json:"date,omitempty"`
	} `json:"start"`
	End struct {
		DateTime string `json:"dateTime,omitempty"`
		Date     string `json:"date,omitempty"`
	} `json:"end"`
	HangoutLink string `json:"hangoutLink,omitempty"`
	Status      string `json:"status,omitempty"`
	Attendees   []struct {
		Email       string `json:"email"`
		DisplayName string `json:"displayName,omitempty"`
		Response    string `json:"responseStatus,omitempty"`
		Organizer   bool   `json:"organizer,omitempty"`
	} `json:"attendees,omitempty"`
}

func (ge googleEvent) toModel(calendarID string) *Event {
	e := &Event{
		RemoteID:    ge.ID,
		CalendarID:  calendarID,
		Summary:     ge.Summary,
		Description: ge.Description,
		Location:    ge.Location,
		MeetLink:    ge.HangoutLink,
		Status:      ge.Status,
	}

	if ge.Start.DateTime != "" {
		e.StartTime, _ = time.Parse(time.RFC3339, ge.Start.DateTime)
		e.EndTime, _ = time.Parse(time.RFC3339, ge.End.DateTime)
		e.IsAllDay = false
	} else {
		e.StartTime, _ = time.Parse("2006-01-02", ge.Start.Date)
		e.EndTime, _ = time.Parse("2006-01-02", ge.End.Date)
		e.IsAllDay = true
	}

	for _, a := range ge.Attendees {
		e.Attendees = append(e.Attendees, Attendee{
			Email:       a.Email,
			Name:        a.DisplayName,
			Status:      a.Response,
			IsOrganizer: a.Organizer,
		})
	}

	return e
}

func fromModel(e *Event) googleEvent {
	ge := googleEvent{
		Summary:     e.Summary,
		Description: e.Description,
		Location:    e.Location,
		Status:      e.Status,
	}

	if e.IsAllDay {
		ge.Start.Date = e.StartTime.Format("2006-01-02")
		ge.End.Date = e.EndTime.Format("2006-01-02")
	} else {
		ge.Start.DateTime = e.StartTime.Format(time.RFC3339)
		ge.End.DateTime = e.EndTime.Format(time.RFC3339)
	}

	for _, a := range e.Attendees {
		ge.Attendees = append(ge.Attendees, struct {
			Email       string `json:"email"`
			DisplayName string `json:"displayName,omitempty"`
			Response    string `json:"responseStatus,omitempty"`
			Organizer   bool   `json:"organizer,omitempty"`
		}{
			Email:       a.Email,
			DisplayName: a.Name,
			Response:    a.Status,
			Organizer:   a.IsOrganizer,
		})
	}

	return ge
}
