package lectionary

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// CalendarDay separates the underlying calendar observance/season from an
// explicitly assigned Mass. A transferred Mass must not move the season or Hours.
// The maintenance adapter handles precedence; unsupported days have no entry.
type CalendarDay struct {
	Date              string          `json:"date"`
	Observance        string          `json:"observance"`
	Rank              string          `json:"rank"`
	Temporal          string          `json:"temporal"`
	SundayCycle       string          `json:"sunday_cycle"`
	WeekdayCycle      string          `json:"weekday_cycle"`
	Alternatives      []string        `json:"alternatives,omitempty"`
	Seasons           []string        `json:"seasons,omitempty"`
	Mass              *MassAssignment `json:"mass,omitempty"`
	Celebration       string          `json:"celebration,omitempty"`
	SourceURL         string          `json:"source_url,omitempty"`
	ReviewNote        string          `json:"review_note,omitempty"`
	UnsupportedReason string          `json:"unsupported_reason,omitempty"`
}

// MassAssignment is an explicit, sourced formulary selection, not a fictitious
// liturgical rank. Keep cycle/context distinctions until the lectionary permits
// broader reuse. It does not select a Gospel or implement the Hours.
type MassAssignment struct {
	Formulary   string `json:"formulary"`
	Celebration string `json:"celebration"`
}

// GospelEntrySchemaVersion changes whenever reusable Gospel identities change.
// Maintenance manifests must explicitly agree; supported caches are not rewritten.
const GospelEntrySchemaVersion = 3

type EntryDefinition struct {
	Observance  string
	Key         string
	Celebration string
	Displaced   string
}

// These three proper Gospels do not vary by Sunday/weekday cycle. Other
// celebrations retain conservative cycle/context keys until individually reviewed.
var invariantPropers = map[string]struct{ ref, label string }{
	"dedication_of_the_lateran_basilica": {"Ga 2,13-22", "Cung hiến Thánh đường La-tê-ra-nô"},
	"all_saints":                         {"Mt 5,1-12a", "Các Thánh Nam Nữ"},
	"annunciation_of_the_lord":           {"Lc 1,26-38", "Lễ Truyền Tin"},
}

// Only explicitly implemented daytime formularies. In particular, an arbitrary
// vigil/night/ritual identity cannot be smuggled into the default day slot.
var assignedDayFormularies = map[string]bool{
	"tet_day_1_new_year":                 true,
	"tet_day_2_ancestors":                true,
	"tet_day_3_labor":                    true,
	"ash_wednesday":                      true,
	"most_holy_body_and_blood_of_christ": true,
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
	if d.Mass != nil && (!assignedDayFormularies[d.Mass.Formulary] || strings.TrimSpace(d.Mass.Celebration) == "") {
		return EntryDefinition{}, fmt.Errorf("assigned Mass requires a formulary identity and celebration")
	}
	switch d.Rank {
	case "WEEKDAY", "SUNDAY", "MEMORIAL", "OPTIONAL_MEMORIAL", "COMMEMORATION", "FEAST", "SOLEMNITY", "TRIDUUM":
	default:
		return EntryDefinition{}, fmt.Errorf("unsupported rank %q", d.Rank)
	}
	e := EntryDefinition{Observance: d.Observance, Celebration: d.Celebration}
	if d.UnsupportedReason != "" {
		if d.Mass != nil {
			return e, fmt.Errorf("unsupported day cannot assign a Mass")
		}
		return e, nil
	}
	if d.Mass != nil {
		e.Observance = d.Mass.Formulary
		e.Displaced = d.Observance
		e.Key = "formulary:" + d.Mass.Formulary + ":" + d.SundayCycle + ":" + d.WeekdayCycle + ":day"
		e.Celebration = d.Mass.Celebration
		return e, nil
	}
	if d.Temporal != d.Observance {
		e.Displaced = d.Temporal
	}
	if d.Observance == "holy_saturday" || d.Observance == "holy_thursday" {
		return e, nil
	} // a generic daytime Mass is not defined
	if p, ok := invariantPropers[d.Observance]; ok && (d.Rank == "FEAST" || d.Rank == "SOLEMNITY") {
		e.Key = "proper:" + d.Observance + ":day"
		e.Celebration = p.label
		return e, nil
	}
	switch d.Rank {
	case "WEEKDAY":
		// OLM 69: weekday Gospels repeat annually. I/II distinguishes the
		// first reading, not the Gospel; proper observances stay separate.
		e.Key = "temporal:" + d.Observance + ":day"
	case "SUNDAY":
		e.Key = "temporal:" + d.Observance + ":" + d.SundayCycle + ":day"
	case "MEMORIAL", "OPTIONAL_MEMORIAL", "COMMEMORATION":
		e.Key = "observance:" + d.Observance + ":" + d.Temporal + ":" + d.SundayCycle + ":" + d.WeekdayCycle + ":day"
	case "FEAST", "SOLEMNITY", "TRIDUUM":
		e.Key = "observance:" + d.Observance + ":" + d.SundayCycle + ":" + d.WeekdayCycle + ":day"
	default:
		return e, fmt.Errorf("unsupported rank %q", d.Rank)
	}
	return e, nil
}

func InvariantGospel(entry string) string {
	// The named Ash Mass has this Gospel, not the Friday-after-Ash reading.
	// Keep the conservative formulary cycle keys; this validates a review and
	// does not automatically approve a mapping or generalize national dates.
	if strings.HasPrefix(entry, "formulary:ash_wednesday:") {
		return "Mt 6,1-6.16-18"
	}
	for id, p := range invariantPropers {
		if entry == "proper:"+id+":day" {
			return p.ref
		}
	}
	return ""
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
