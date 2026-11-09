package lectionary

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// CalendarDay is the selected observance from a versioned calendar provider,
// not the old seasonal estimate. The maintenance adapter handles precedence.
type CalendarDay struct {
	Date         string   `json:"date"`
	Observance   string   `json:"observance"`
	Rank         string   `json:"rank"`
	Temporal     string   `json:"temporal"`
	SundayCycle  string   `json:"sunday_cycle"`
	WeekdayCycle string   `json:"weekday_cycle"`
	Alternatives []string `json:"alternatives,omitempty"`
}

type EntryDefinition struct {
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
	e := EntryDefinition{}
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
		e.Key = "temporal:" + d.Observance + ":" + d.WeekdayCycle + ":day"
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
	for id, p := range invariantPropers {
		if entry == "proper:"+id+":day" {
			return p.ref
		}
	}
	return ""
}

// CalendarSupported describes adapter profiles, not ecclesiastical approval.
func CalendarSupported(id string) bool { return id == "roman-general" || id == "vietnam" }

func ValidRulesVersion(version string) bool {
	return len(version) > 0 && len(version) <= 128 && !strings.ContainsAny(version, "\n\r\t")
}
