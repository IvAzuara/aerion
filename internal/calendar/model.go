package calendar

import (
	"time"
)

// Calendar represents a calendar associated with an account
type Calendar struct {
	ID           string    `json:"id"`
	AccountID    string    `json:"accountId"`
	Name         string    `json:"name"`
	Color        string    `json:"color"`
	Type         string    `json:"type"` // 'google', 'local'
	Enabled      bool      `json:"enabled"`
	SyncToken    string    `json:"syncToken,omitempty"`
	LastSyncedAt time.Time `json:"lastSyncedAt,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Event represents a calendar event
type Event struct {
	ID          string     `json:"id"`
	CalendarID  string     `json:"calendarId"`
	RemoteID    string     `json:"remoteId,omitempty"`
	Summary     string     `json:"summary"`
	Description string     `json:"description,omitempty"`
	Location    string     `json:"location,omitempty"`
	StartTime   time.Time  `json:"startTime"`
	EndTime     time.Time  `json:"endTime"`
	IsAllDay    bool       `json:"isAllDay"`
	Recurrence  string     `json:"recurrence,omitempty"`
	MeetLink    string     `json:"meetLink,omitempty"`
	Status      string     `json:"status"` // 'confirmed', 'tentative', 'cancelled'
	Attendees   []Attendee `json:"attendees,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// Attendee represents an event participant
type Attendee struct {
	ID          string `json:"id"`
	EventID     string `json:"eventId"`
	Email       string `json:"email"`
	Name        string `json:"name,omitempty"`
	Status      string `json:"status"` // 'needsAction', 'declined', 'tentative', 'accepted'
	IsOrganizer bool   `json:"isOrganizer"`
}
