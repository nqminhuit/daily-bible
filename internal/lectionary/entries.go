package lectionary

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// CalendarDay is one resolved calendar day. The maintenance adapter handles
// precedence and names the year-independent GospelEntry; unsupported days have none.
type CalendarDay struct {
	Date              string          `json:"date"`
	Observance        string          `json:"observance"`
	Rank              string          `json:"rank"`
	Temporal          string          `json:"temporal"`
	SundayCycle       string          `json:"sunday_cycle"`
	WeekdayCycle      string          `json:"weekday_cycle"`
	Alternatives      []string        `json:"alternatives,omitempty"`
	Seasons           []string        `json:"seasons,omitempty"`
	GospelEntry       string          `json:"gospel_entry,omitempty"`
	Mass              *MassAssignment `json:"mass,omitempty"`
	Celebration       string          `json:"celebration,omitempty"`
	SourceURL         string          `json:"source_url,omitempty"`
	ReviewNote        string          `json:"review_note,omitempty"`
	UnsupportedReason string          `json:"unsupported_reason,omitempty"`
}

// MassAssignment is retained only so stored schema-3 definitions still decode;
// schema 4 rejects it.
type MassAssignment struct {
	Formulary   string `json:"formulary"`
	Celebration string `json:"celebration"`
}

// GospelEntrySchemaVersion changes whenever reusable Gospel identities change.
// Maintenance manifests must explicitly agree; supported caches are not rewritten.
const GospelEntrySchemaVersion = 4

type EntryDefinition struct {
	Observance  string
	Key         string
	Celebration string
	Displaced   string
}

var calendarIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,150}$`)

func (d CalendarDay) Entry() (EntryDefinition, error) {
	if _, err := time.Parse("2006-01-02", d.Date); err != nil {
		return EntryDefinition{}, err
	}
	if !calendarIdentifier.MatchString(d.Observance) || !calendarIdentifier.MatchString(d.Temporal) {
		return EntryDefinition{}, fmt.Errorf("invalid observance/temporal identity")
	}
	if d.SundayCycle != "YEAR_A" && d.SundayCycle != "YEAR_B" && d.SundayCycle != "YEAR_C" {
		return EntryDefinition{}, fmt.Errorf("invalid Sunday cycle")
	}
	if d.WeekdayCycle != "YEAR_1" && d.WeekdayCycle != "YEAR_2" {
		return EntryDefinition{}, fmt.Errorf("invalid weekday cycle")
	}
	for _, id := range d.Alternatives {
		if !calendarIdentifier.MatchString(id) {
			return EntryDefinition{}, fmt.Errorf("invalid alternative")
		}
	}
	for _, season := range d.Seasons {
		if !calendarIdentifier.MatchString(season) {
			return EntryDefinition{}, fmt.Errorf("invalid season")
		}
	}
	if d.Mass != nil {
		return EntryDefinition{}, fmt.Errorf("assigned Masses are not supported by Gospel entry schema %d", GospelEntrySchemaVersion)
	}
	switch d.Rank {
	case "WEEKDAY", "SUNDAY", "MEMORIAL", "OPTIONAL_MEMORIAL", "COMMEMORATION", "FEAST", "SOLEMNITY", "TRIDUUM":
	default:
		return EntryDefinition{}, fmt.Errorf("unsupported rank %q", d.Rank)
	}
	e := EntryDefinition{Observance: d.Observance, Celebration: d.Celebration}
	if d.UnsupportedReason != "" {
		if d.GospelEntry != "" {
			return e, fmt.Errorf("unsupported day cannot name a Gospel entry")
		}
		return e, nil
	}
	if d.Temporal != d.Observance {
		e.Displaced = d.Temporal
	}
	base := d.GospelEntry
	switch base {
	case "temporal:" + d.Temporal, "temporal:" + d.Temporal + ":" + d.SundayCycle,
		"observance:" + d.Observance, "observance:" + d.Observance + ":" + d.SundayCycle:
		e.Key = base + ":day"
	default:
		return e, fmt.Errorf("gospel entry %q does not match the day's observance/temporal identity", base)
	}
	return e, nil
}

// CalendarSupported describes implemented namespaces, not ecclesiastical approval.
func CalendarSupported(id string) bool { return id == "roman-general" || id == "vietnam" }

// CalendarRulesSupported blocks the unreviewed national Romcal profile: its
// pinned Vietnam plugin puts the Vietnamese Martyrs on November 13, not 24.
// Reviewed national definitions need their own explicit rules version;
// merely renaming that provider's output does not constitute calendar review.
func CalendarRulesSupported(id, version string) bool {
	return CalendarSupported(id) && ValidRulesVersion(version) && !(id == "vietnam" && strings.HasPrefix(version, "romcal-"))
}

func ValidRulesVersion(version string) bool {
	return len(version) > 0 && len(version) <= 128 && !strings.ContainsAny(version, "\n\r\t")
}
