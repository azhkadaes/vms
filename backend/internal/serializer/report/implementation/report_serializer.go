package reportserializerimplementation

import (
	"strconv"
	"strings"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	reportserializer "github.com/azhkadaes/vms/backend/internal/serializer/report"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type serializer struct{}

func NewReportSerializer() reportserializer.ReportSerializer { return serializer{} }

func (serializer) ToVisitLogRow(r entity.ReportVisit, status string) dto.VisitLogRow {
	return dto.VisitLogRow{ID: strconv.FormatInt(int64(r.ID), 10), Reference: r.Reference, Name: r.Name,
		Organization: r.Organization, OfficerID: strconv.FormatInt(int64(r.OfficerID), 10), Host: r.Host,
		Date: r.Date, Time: r.Time, Duration: r.Duration, Room: r.Room, Purpose: r.Purpose,
		Kind: r.Kind, Visitors: r.Visitors, Status: status, CheckedInAt: utc(r.CheckedInAt), CheckedOutAt: utc(r.CheckedOutAt)}
}

func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	value := t.UTC()
	return &value
}

func (serializer) ToSummaryExport(summary dto.VisitSummary, statuses []string) dto.SummaryExport {
	statusLabels := map[string]string{
		"pending": "Pending", "approved": "Disetujui", "rejected": "Ditolak", "cancelled": "Dibatalkan",
		"checked_in": "Checked in", "checked_out": "Checked out", "missed": "Terlewat",
	}
	scopeLabels := map[string]string{
		"organization": "Ringkasan organisasi", "assigned": "Pejabat dalam penugasan Anda",
		"attendance": "Lingkup penerimaan", "authorized": "Lingkup kunjungan yang diizinkan",
	}
	var csv strings.Builder
	csv.WriteRune('\ufeff')
	write := func(cells ...string) {
		for i, cell := range cells {
			if i > 0 {
				csv.WriteByte(',')
			}
			csv.WriteByte('"')
			csv.WriteString(strings.ReplaceAll(cell, `"`, `""`))
			csv.WriteByte('"')
		}
	}
	write("Tanggal awal", "Tanggal akhir", "Cakupan", "Status", "Jumlah")
	for _, status := range statuses {
		csv.WriteString("\r\n")
		write(summary.Filter.From, summary.Filter.To, scopeLabels[summary.Scope], statusLabels[status], strconv.Itoa(summary.Counts[status]))
	}
	return dto.SummaryExport{Filename: "kunjungan-ringkasan-" + summary.Filter.From + "-" + summary.Filter.To + ".csv", CSV: csv.String(), Revision: summary.Revision}
}
