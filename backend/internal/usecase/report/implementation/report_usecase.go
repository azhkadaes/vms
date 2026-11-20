package reportusecaseimplementation

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	reportrepo "github.com/azhkadaes/vms/backend/internal/repository/report"
	reportserializer "github.com/azhkadaes/vms/backend/internal/serializer/report"
	reportusecase "github.com/azhkadaes/vms/backend/internal/usecase/report"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"go.opentelemetry.io/otel/trace"
)

var reportStatuses = []string{"pending", "approved", "rejected", "cancelled", "checked_in", "checked_out", "missed"}

type reports struct {
	repo       reportrepo.ReportReader
	serializer reportserializer.ReportSerializer
	tracer     trace.Tracer
}

func NewReportUsecase(repo reportrepo.ReportReader, serializer reportserializer.ReportSerializer, tracer trace.Tracer) reportusecase.ReportUsecase {
	return &reports{repo: repo, serializer: serializer, tracer: tracer}
}

func scope(s dto.Session) string {
	if !policy.Has(s, policy.PermReports) {
		return ""
	}
	switch s.Role {
	case policy.RoleSuperAdmin:
		return "organization"
	case policy.RoleProtocol:
		return "authorized"
	case policy.RolePA:
		return "assigned"
	case policy.RoleReceptionist:
		return "attendance"
	default:
		return ""
	}
}

func (u *reports) Summary(ctx context.Context, s dto.Session, filter dto.SummaryFilter) (dto.VisitSummary, error) {
	ctx, span := u.tracer.Start(ctx, "report.summary")
	defer span.End()
	return u.summary(ctx, s, filter, false)
}

type revisionRow struct {
	ID      int32
	Version int
	Status  string
}

func (u *reports) summary(ctx context.Context, s dto.Session, filter dto.SummaryFilter, export bool) (dto.VisitSummary, error) {
	label := scope(s)
	canExport := policy.Has(s, policy.PermReportExport)
	if label == "" || (export && !canExport) {
		return dto.VisitSummary{}, failure.New(403, "forbidden")
	}
	if err := validateSummaryFilter(filter); err != nil {
		return dto.VisitSummary{}, err
	}
	rows, at, err := u.repo.Summary(ctx, s, filter)
	if err != nil {
		return dto.VisitSummary{}, err
	}
	result := dto.VisitSummary{Filter: filter, Scope: label, GeneratedAt: at.UTC(), CanExport: canExport, Counts: map[string]int{}}
	for _, status := range reportStatuses {
		result.Counts[status] = 0
	}
	matched := make([]revisionRow, 0, len(rows))
	for _, row := range rows {
		status := displayStatus(row, at)
		if !matches(filter.Status, status, "all") {
			continue
		}
		result.Counts[status]++
		result.Total++
		matched = append(matched, revisionRow{row.ID, row.Version, status})
	}
	slices.SortFunc(matched, func(a, b revisionRow) int { return cmp.Compare(a.ID, b.ID) })
	canonical := filter
	canonical.Status = ordered(filter.Status)
	result.Revision, err = revision(canonical, matched)
	return result, err
}

func (u *reports) ExportSummary(ctx context.Context, s dto.Session, filter dto.SummaryFilter, expected string) (dto.SummaryExport, error) {
	ctx, span := u.tracer.Start(ctx, "report.summary.export")
	defer span.End()
	summary, err := u.summary(ctx, s, filter, true)
	if err != nil {
		return dto.SummaryExport{}, err
	}
	if err := requireRevision(summary.Revision, expected); err != nil {
		return dto.SummaryExport{}, err
	}
	statuses := slices.DeleteFunc(slices.Clone(reportStatuses), func(status string) bool { return !matches(filter.Status, status, "all") })
	return u.serializer.ToSummaryExport(summary, statuses), nil
}

func (u *reports) VisitLog(ctx context.Context, s dto.Session, filter dto.VisitLogFilter, page, size int) (dto.VisitLogResponse, error) {
	ctx, span := u.tracer.Start(ctx, "report.visits")
	defer span.End()
	if page < 1 {
		return dto.VisitLogResponse{}, failure.Validation().Add("page", "Nomor halaman harus lebih dari nol.").Err()
	}
	if !slices.Contains([]int{10, 25, 50}, size) {
		return dto.VisitLogResponse{}, failure.Validation().Add("pageSize", "Gunakan ukuran halaman 10, 25, atau 50.").Err()
	}
	snapshot, options, err := u.visitLog(ctx, s, filter, false)
	if err != nil {
		return dto.VisitLogResponse{}, err
	}
	officers := make([]dto.VisitLogOfficer, 0, len(options.Officers))
	for _, officer := range options.Officers {
		officers = append(officers, dto.VisitLogOfficer{ID: strconv.FormatInt(int64(officer.ID), 10), Name: officer.Name})
	}
	page = min(page, max(1, (snapshot.Summary.Total+size-1)/size))
	start := (page - 1) * size
	items := snapshot.Items[start:min(start+size, snapshot.Summary.Total)]
	return dto.VisitLogResponse{Filter: filter, Items: items, Summary: snapshot.Summary, GeneratedAt: snapshot.GeneratedAt,
		Revision: snapshot.Revision, Page: page, PageSize: size, Total: snapshot.Summary.Total,
		CanExport: policy.Has(s, policy.PermReportExport), Officers: officers, Rooms: append([]string{}, options.Rooms...)}, nil
}

func (u *reports) ExportVisitLog(ctx context.Context, s dto.Session, filter dto.VisitLogFilter, expected string) (dto.VisitLogSnapshot, error) {
	ctx, span := u.tracer.Start(ctx, "report.visits.export")
	defer span.End()
	snapshot, _, err := u.visitLog(ctx, s, filter, true)
	if err != nil {
		return dto.VisitLogSnapshot{}, err
	}
	if err := requireRevision(snapshot.Revision, expected); err != nil {
		return dto.VisitLogSnapshot{}, err
	}
	if snapshot.Summary.Total > 10_000 {
		return dto.VisitLogSnapshot{}, failure.New(422, "export_too_large")
	}
	return snapshot, nil
}

func displayStatus(row entity.ReportStatus, at time.Time) string {
	return policy.DisplayStatus(row.Status, row.StartsAt, row.Duration, at)
}

type visitRevision struct {
	Row     dto.VisitLogRow
	Version int
}

func (u *reports) visitLog(ctx context.Context, s dto.Session, filter dto.VisitLogFilter, export bool) (dto.VisitLogSnapshot, entity.ReportSnapshot, error) {
	if scope(s) != "organization" || (export && !policy.Has(s, policy.PermReportExport)) {
		return dto.VisitLogSnapshot{}, entity.ReportSnapshot{}, failure.New(403, "forbidden")
	}
	if err := validateLogFilter(filter); err != nil {
		return dto.VisitLogSnapshot{}, entity.ReportSnapshot{}, err
	}
	source, err := u.repo.VisitLog(ctx, filter)
	if err != nil {
		return dto.VisitLogSnapshot{}, entity.ReportSnapshot{}, err
	}
	result := dto.VisitLogSnapshot{Filter: filter, GeneratedAt: source.At.UTC(), Items: []dto.VisitLogRow{}}
	needle := strings.ToLower(strings.TrimSpace(filter.Q))
	selected := make([]visitRevision, 0, len(source.Rows))
	for _, record := range source.Rows {
		if err := ctx.Err(); err != nil {
			return dto.VisitLogSnapshot{}, entity.ReportSnapshot{}, err
		}
		status := displayStatus(record.ReportStatus, source.At)
		roomID := ""
		if record.RoomID != nil {
			roomID = strconv.FormatInt(int64(*record.RoomID), 10)
		}
		if !matches(filter.Status, status, "all") || !matches(filter.Kind, record.Kind, "all") ||
			!matches(filter.Officer, strconv.FormatInt(int64(record.OfficerID), 10), "") || !matches(filter.Room, roomID, "") {
			continue
		}
		row := u.serializer.ToVisitLogRow(record, status)
		if !search(row, needle) {
			continue
		}
		selected = append(selected, visitRevision{row, record.Version})
		result.Summary.Total++
		result.Summary.Visitors += visitorCount(row, 0)
		switch status {
		case "checked_in":
			result.Summary.Inside++
		case "checked_out":
			result.Summary.Completed++
		case "pending":
			result.Summary.Pending++
		}
	}
	slices.SortFunc(selected, func(a, b visitRevision) int { return compareRows(a.Row, b.Row, filter.Sort, filter.Dir) })
	for _, row := range selected {
		result.Items = append(result.Items, row.Row)
	}
	canonical := filter
	canonical.Status, canonical.Kind, canonical.Officer, canonical.Room = ordered(filter.Status), ordered(filter.Kind), ordered(filter.Officer), ordered(filter.Room)
	canonical.Q = needle
	result.Revision, err = revision(canonical, selected)
	return result, source, err
}

func search(row dto.VisitLogRow, needle string) bool {
	if needle == "" {
		return true
	}
	for _, value := range []string{row.Reference, row.Name, row.Organization, row.Host, row.Room, row.Purpose} {
		if strings.Contains(strings.ToLower(value), needle) {
			return true
		}
	}
	return false
}

func visitorCount(row dto.VisitLogRow, fallback int) int {
	if row.Visitors == nil {
		return fallback
	}
	return *row.Visitors
}

func compareRows(a, b dto.VisitLogRow, key, dir string) int {
	var c int
	switch key {
	case "date":
		c = cmp.Compare(a.Date+a.Time, b.Date+b.Time)
	case "name":
		c = compareText(a.Name, b.Name)
	case "organization":
		c = compareText(a.Organization, b.Organization)
	case "host":
		c = compareText(a.Host, b.Host)
	case "status":
		c = cmp.Compare(a.Status, b.Status)
	case "kind":
		c = cmp.Compare(a.Kind, b.Kind)
	case "visitors":
		c = cmp.Compare(visitorCount(a, -1), visitorCount(b, -1))
	}
	if c == 0 {
		c = cmp.Compare(a.ID, b.ID)
	}
	if dir == "desc" {
		return -c
	}
	return c
}

func compareText(a, b string) int { return cmp.Compare(strings.ToLower(a), strings.ToLower(b)) }

func matches(values dto.ReportFilterValues, value, all string) bool {
	return len(values) == 0 || (len(values) == 1 && values[0] == all) || slices.Contains(values, value)
}

func ordered(values dto.ReportFilterValues) dto.ReportFilterValues {
	cloned := slices.Clone(values)
	slices.Sort(cloned)
	return cloned
}

func revision(filter, rows any) (string, error) {
	data, err := json.Marshal([]any{filter, rows})
	if err != nil {
		return "", fmt.Errorf("encode report revision: %w", err)
	}
	return security.Hash(string(data)), nil
}

func requireRevision(got, want string) error {
	if len(want) != 64 || strings.Trim(want, "0123456789abcdef") != "" {
		return failure.Validation().Add("revision", "Versi laporan tidak valid.").Err()
	}
	if got != want {
		return failure.New(409, "revision_conflict")
	}
	return nil
}
