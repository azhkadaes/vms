package reportusecaseimplementation

import (
	"slices"
	"time"
	"unicode/utf8"

	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

func reportDate(s string) (time.Time, bool) {
	d, err := time.Parse(time.DateOnly, s)
	return d, err == nil && d.Year() > 0 && d.Format(time.DateOnly) == s
}

func validateSummaryFilter(filter dto.SummaryFilter) error {
	fields := failure.Validation()
	from, validFrom := reportDate(filter.From)
	to, validTo := reportDate(filter.To)
	if !validFrom {
		fields.Add("from", "Pilih tanggal awal yang valid.")
	}
	if !validTo {
		fields.Add("to", "Pilih tanggal akhir yang valid.")
	} else if validFrom && (to.Before(from) || to.After(from.AddDate(0, 0, 365))) {
		fields.Add("to", "Rentang laporan harus berurutan dan maksimal 366 hari.")
	}
	if !validChoices(filter.Status, "all", statusChoice) {
		fields.Add("status", "Pilih status yang valid tanpa pengulangan.")
	}
	return fields.Err()
}

func validateLogFilter(filter dto.VisitLogFilter) error {
	fields := failure.Validation()
	if _, ok := reportDate(filter.From); filter.From != "" && !ok {
		fields.Add("from", "Pilih tanggal awal yang valid.")
	}
	if _, ok := reportDate(filter.To); filter.To != "" && (!ok || (filter.From != "" && filter.To < filter.From)) {
		fields.Add("to", "Pilih tanggal akhir yang valid dan tidak sebelum tanggal awal.")
	}
	if utf8.RuneCountInString(filter.Q) > 200 {
		fields.Add("q", "Pencarian maksimal 200 karakter.")
	}
	if !validChoices(filter.Status, "all", statusChoice) {
		fields.Add("status", "Pilih status yang valid tanpa pengulangan.")
	}
	if !validChoices(filter.Kind, "all", func(s string) bool { return slices.Contains([]string{"standard", "vip", "restricted"}, s) }) {
		fields.Add("kind", "Pilih jenis kunjungan yang valid tanpa pengulangan.")
	}
	if !validChoices(filter.Officer, "", idChoice) {
		fields.Add("officer", "Pilih ID pejabat yang valid tanpa pengulangan.")
	}
	if !validChoices(filter.Room, "", idChoice) {
		fields.Add("room", "Pilih ID ruangan yang valid tanpa pengulangan.")
	}
	if !slices.Contains([]string{"date", "name", "organization", "host", "status", "kind", "visitors"}, filter.Sort) {
		fields.Add("sort", "Pilih urutan yang valid.")
	}
	if filter.Dir != "asc" && filter.Dir != "desc" {
		fields.Add("dir", "Pilih arah urutan yang valid.")
	}
	return fields.Err()
}

func statusChoice(s string) bool { return slices.Contains(reportStatuses, s) }
func idChoice(s string) bool     { _, err := validate.ID(s); return err == nil }

func validChoices(values dto.ReportFilterValues, fallback string, valid func(string) bool) bool {
	if len(values) == 1 && values[0] == fallback {
		return true
	}
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if seen[value] || !valid(value) {
			return false
		}
		seen[value] = true
	}
	return true
}
