package main

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"os"
	"sort"
	"time"
)

const (
	RedEmoji    = "<:statusred:816435667001147482>"
	YellowEmoji = "<:statusyellow:816435667017400350>"
	GreenEmoji  = "<:statusgreen:816435666988171314>"
)

type InstatusWebhook struct {
	PageMeta

	Incident    *Incident    `json:"incident"`
	Maintenance *Maintenance `json:"maintenance"`
}

type PageMeta struct {
	Meta struct {
		Unsubscribe   string `json:"unsubscribe"`
		Documentation string `json:"documentation"`
	} `json:"meta"`

	Page struct {
		ID                string              `json:"id"`
		URL               string              `json:"url"`
		StatusIndicator   PageStatusIndicator `json:"status_indicator"`
		StatusDescription string              `json:"status_description"`
	} `json:"page"`
}

type PageStatusIndicator string

const (
	StatusIndicatorUp               PageStatusIndicator = "UP"
	StatusIndicatorHasIssues        PageStatusIndicator = "HASISSUES"
	StatusIndicatorUnderMaintenance PageStatusIndicator = "UNDERMAINTENANCE"
)

type AffectedComponent struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Incident struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`

	Status     IncidentStatus `json:"status"`
	Impact     string         `json:"impact"`
	Backfilled bool           `json:"backfilled"`

	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
	ResolvedAt *string `json:"resolved_at"`

	Components []AffectedComponent `json:"affected_components"`

	Updates []IncidentUpdate `json:"incident_updates"`
}

func (u *Incident) Emoji() string {
	switch u.Status {
	case IncidentStatusInvestigating:
		return RedEmoji
	case IncidentStatusIdentified, IncidentStatusMonitoring:
		return YellowEmoji
	case IncidentStatusResolved:
		return GreenEmoji
	default:
		return ""
	}
}

type IncidentStatus string

const (
	IncidentStatusInvestigating IncidentStatus = "Investigating"
	IncidentStatusIdentified    IncidentStatus = "Identified"
	IncidentStatusMonitoring    IncidentStatus = "Monitoring"
	IncidentStatusResolved      IncidentStatus = "Resolved"
)

type IncidentUpdate struct {
	ID         string `json:"id"`
	IncidentID string `json:"incident_id"`

	Status   IncidentStatus `json:"status"`
	Body     string         `json:"body"`
	Markdown string         `json:"markdown"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (u *IncidentUpdate) HumanizedTime() string {
	return humanizedTime(u.CreatedAt)
}

func (u *IncidentUpdate) Emoji() string {
	switch u.Status {
	case IncidentStatusInvestigating:
		return "<:statusred:816435667001147482>"
	case IncidentStatusIdentified, IncidentStatusMonitoring:
		return "<:statusyellow:816435667017400350>"
	case IncidentStatusResolved:
		return "<:statusgreen:816435666988171314>"
	default:
		return ""
	}
}

type Maintenance struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`

	Duration   float64           `json:"duration"`
	Status     MaintenanceStatus `json:"status"`
	Impact     string            `json:"impact"`
	Backfilled bool              `json:"backfilled"`

	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
	ResolvedAt *string `json:"resolved_at"`

	Components []AffectedComponent `json:"affected_components"`

	Updates []MaintenanceUpdate `json:"maintenance_updates"`
}

func (u *Maintenance) Emoji() string {
	switch u.Status {
	case MaintenanceStatusPlanned:
		return YellowEmoji
	case MaintenanceStatusInProgress:
		return RedEmoji
	case MaintenanceStatusCompleted:
		return GreenEmoji
	default:
		return ""
	}
}

type MaintenanceStatus string

const (
	MaintenanceStatusPlanned    MaintenanceStatus = "Planned"
	MaintenanceStatusInProgress MaintenanceStatus = "In progress"
	MaintenanceStatusCompleted  MaintenanceStatus = "Completed"
)

type MaintenanceUpdate struct {
	ID            string `json:"id"`
	MaintenanceID string `json:"maintenance_id"`

	Body     string `json:"body"`
	Markdown string `json:"markdown"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (u *MaintenanceUpdate) HumanizedTime() string {
	return humanizedTime(u.CreatedAt)
}

func Main(ctx context.Context, event InstatusWebhook) {
	webhookID := os.Getenv("DISCORD_WEBHOOK_ID")
	webhookToken := os.Getenv("DISCORD_WEBHOOK_TOKEN")
	statusRoleID := os.Getenv("DISCORD_STATUS_ROLE_ID")

	s, err := discordgo.New("")
	if err != nil {
		return
	}

	var e *discordgo.MessageEmbed
	if event.Incident != nil {
		e = makeIncidentEmbed(event.Incident)
	} else if event.Maintenance != nil {
		e = makeMaintenanceEmbed(event.Maintenance)
	} else {
		return
	}

	p := &discordgo.WebhookParams{
		Content: fmt.Sprintf("<@&%s>", statusRoleID),
		Embeds:  []*discordgo.MessageEmbed{e},
	}

	_, err = s.WebhookExecute(webhookID, webhookToken, false, p)

	return
}

func makeIncidentEmbed(inc *Incident) *discordgo.MessageEmbed {
	var fields []*discordgo.MessageEmbedField

	sort.Slice(inc.Updates, func(i, j int) bool {
		return inc.Updates[i].CreatedAt < inc.Updates[j].CreatedAt
	})

	for _, u := range inc.Updates {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s %s (%s)", u.Emoji(), u.Status, u.HumanizedTime()),
			Value:  u.Markdown,
			Inline: false,
		})
	}

	return &discordgo.MessageEmbed{
		URL:    inc.URL,
		Title:  fmt.Sprintf("%s Incident: %s", inc.Emoji(), inc.Name),
		Color:  0x2483C5,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Started at",
		},
		Timestamp: inc.CreatedAt,
	}
}

func makeMaintenanceEmbed(m *Maintenance) *discordgo.MessageEmbed {
	var fields []*discordgo.MessageEmbedField

	sort.Slice(m.Updates, func(i, j int) bool {
		return m.Updates[i].CreatedAt < m.Updates[j].CreatedAt
	})

	for _, u := range m.Updates {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("Update (%s)", u.HumanizedTime()),
			Value:  u.Markdown,
			Inline: false,
		})
	}

	return &discordgo.MessageEmbed{
		URL:    m.URL,
		Title:  fmt.Sprintf("%s Maintenance: %s", m.Emoji(), m.Name),
		Color:  0x2483C5,
		Fields: fields,
	}
}

func humanizedTime(t string) string {
	parsed, err := time.Parse(time.RFC3339Nano, t)
	if err != nil {
		return t
	}

	return parsed.Format("15:04:05z")
}
