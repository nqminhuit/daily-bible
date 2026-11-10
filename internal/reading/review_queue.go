package reading

import (
	"context"
	"database/sql"
	"sort"

	"github.com/minh/daily-bible/internal/scripture"
)

// ReferenceCandidate is review material, not an approved mapping. Even available
// verse rows cannot establish calendar applicability or word-level completeness.
type ReferenceCandidate struct {
	Ref        string   `json:"ref"`
	Dates      []string `json:"dates"`
	TextStatus string   `json:"text_status"`
}
type EvidenceCandidate struct {
	Ref  string `json:"ref"`
	URL  string `json:"source_url"`
	Date string `json:"source_date,omitempty"`
}
type EntryAudit struct {
	EntryKey             string               `json:"entry_key"`
	RulesVersion         string               `json:"rules_version"`
	MassOption           string               `json:"mass_option"`
	Dates                []string             `json:"dates"`
	Statuses             map[string]int       `json:"date_statuses"`
	LegacyCandidates     []ReferenceCandidate `json:"legacy_candidates"`
	Observations         []EvidenceCandidate  `json:"source_observations"`
	AcceptedEvidence     []EvidenceCandidate  `json:"accepted_evidence"`
	HasReferenceVariants bool                 `json:"has_reference_variants"`
}

// FinalizeAudit groups dates by actual resolved identity, never the old key.
// Call after merging year reports so one entry has one review queue item.
// It performs local read-only queries; it neither approves candidates nor crawls.
func FinalizeAudit(ctx context.Context, db *sql.DB, report *AuditReport) error {
	type identity struct{ version, key string }
	groups := map[identity]*EntryAudit{}
	for _, row := range report.Rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if row.EntryKey == "" {
			continue
		} // source-only/date decisions are not reusable entries
		id := identity{row.RulesVersion, row.EntryKey}
		e := groups[id]
		if e == nil {
			e = &EntryAudit{EntryKey: id.key, RulesVersion: id.version, MassOption: DayMass, Statuses: map[string]int{}, LegacyCandidates: []ReferenceCandidate{}, Observations: []EvidenceCandidate{}, AcceptedEvidence: []EvidenceCandidate{}}
			groups[id] = e
		}
		e.Dates = append(e.Dates, row.Date)
		e.Statuses[row.Status]++
		if row.LegacyRef != "" {
			found := false
			for i := range e.LegacyCandidates {
				c := &e.LegacyCandidates[i]
				if c.Ref == row.LegacyRef || scripture.Equivalent(c.Ref, row.LegacyRef) {
					c.Dates = append(c.Dates, row.Date)
					found = true
					break
				}
			}
			if !found {
				ref := row.LegacyRef
				if normalized, err := scripture.NormalizeGospelRef(ref); err == nil {
					ref = normalized
				}
				e.LegacyCandidates = append(e.LegacyCandidates, ReferenceCandidate{Ref: ref, Dates: []string{row.Date}})
			}
		}
		addEvidence := func(items *[]EvidenceCandidate, ref, url, date string) {
			if ref == "" || url == "" {
				return
			}
			c := EvidenceCandidate{Ref: ref, URL: url, Date: date}
			for _, old := range *items {
				if old == c {
					return
				}
			}
			*items = append(*items, c)
		}
		addEvidence(&e.Observations, row.ObservedRef, row.ObservationURL, row.ObservationDate)
		addEvidence(&e.AcceptedEvidence, row.ResolvedRef, row.EvidenceURL, row.EvidenceDate)
	}
	// Cache coverage work across groups: a verse passage may be shared without
	// conflating the separate calendar/entry approvals.
	textStatus := map[string]string{}
	entries := make([]EntryAudit, 0, len(groups))
	for _, e := range groups {
		sort.Strings(e.Dates)
		var refs []string
		addVariant := func(ref string) {
			if _, err := scripture.NormalizeGospelRef(ref); err != nil {
				return
			}
			for _, old := range refs {
				if scripture.Equivalent(old, ref) {
					return
				}
			}
			refs = append(refs, ref)
		}
		for i := range e.LegacyCandidates {
			c := &e.LegacyCandidates[i]
			sort.Strings(c.Dates)
			addVariant(c.Ref)
			status, ok := textStatus[c.Ref]
			if !ok {
				status = "invalid_reference"
				if _, err := scripture.NormalizeGospelRef(c.Ref); err == nil {
					available, err := scripture.HasText(ctx, db, c.Ref)
					if err != nil {
						return err
					}
					status = "missing_text"
					if available {
						status = "structurally_available"
					}
				}
				textStatus[c.Ref] = status
			}
			c.TextStatus = status
		}
		for _, c := range e.Observations {
			addVariant(c.Ref)
		}
		for _, c := range e.AcceptedEvidence {
			addVariant(c.Ref)
		}
		e.HasReferenceVariants = len(refs) > 1
		sort.Slice(e.LegacyCandidates, func(i, j int) bool { return e.LegacyCandidates[i].Ref < e.LegacyCandidates[j].Ref })
		less := func(items []EvidenceCandidate, i, j int) bool {
			a, b := items[i], items[j]
			if a.Date != b.Date {
				return a.Date < b.Date
			}
			if a.URL != b.URL {
				return a.URL < b.URL
			}
			return a.Ref < b.Ref
		}
		sort.Slice(e.Observations, func(i, j int) bool { return less(e.Observations, i, j) })
		sort.Slice(e.AcceptedEvidence, func(i, j int) bool { return less(e.AcceptedEvidence, i, j) })
		entries = append(entries, *e)
	}
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.RulesVersion != b.RulesVersion {
			return a.RulesVersion < b.RulesVersion
		}
		return a.EntryKey < b.EntryKey
	})
	report.Entries = entries
	return ctx.Err()
}
