package staffbookingusecaseimplementation

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/policy"
	staffbookingrepository "github.com/azhkadaes/vms/backend/internal/repository/staffbooking"
	staffbookingserializer "github.com/azhkadaes/vms/backend/internal/serializer/staffbooking"
	staffbookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

type usecase struct {
	reader     staffbookingrepository.StaffBookingReader
	serializer staffbookingserializer.StaffBookingSerializer
	tracer     trace.Tracer
}

func NewStaffBookingUsecase(reader staffbookingrepository.StaffBookingReader, serializer staffbookingserializer.StaffBookingSerializer, tracer trace.Tracer) staffbookingusecase.StaffBookingUsecase {
	return &usecase{reader: reader, serializer: serializer, tracer: tracer}
}

func (u *usecase) Get(ctx context.Context, session dto.Session, id int32) (dto.StaffBooking, error) {
	ctx, span := u.tracer.Start(ctx, "staff_bookings.get")
	defer span.End()
	if !policy.Has(session, policy.PermBookings) || session.Role == policy.RoleReceptionist {
		return dto.StaffBooking{}, failure.New(403, "forbidden")
	}
	visit, at, err := u.reader.Find(ctx, session, id)
	if err != nil {
		return dto.StaffBooking{}, err
	}
	return u.serializer.Project(visit, session, at, false), nil
}

func (u *usecase) List(ctx context.Context, session dto.Session, query staffbookingusecase.Query) (dto.StaffBookingPage, error) {
	ctx, span := u.tracer.Start(ctx, "staff_bookings.list")
	defer span.End()
	if !policy.Has(session, policy.PermBookings) {
		return dto.StaffBookingPage{}, failure.New(403, "forbidden")
	}
	filter, err := parseFilter(query)
	if err != nil {
		return dto.StaffBookingPage{}, err
	}
	if filter.View == "reception" && session.Role != policy.RoleReceptionist || filter.View == "dashboard" && session.Role == policy.RoleSuperAdmin {
		return dto.StaffBookingPage{}, failure.New(403, "forbidden")
	}
	visits, at, err := u.reader.List(ctx, session)
	if err != nil {
		return dto.StaffBookingPage{}, err
	}
	if filter.Date == "" && filter.View != "list" {
		filter.Date = at.In(validate.WITA).Format(time.DateOnly)
	}
	bookings := make([]dto.StaffBooking, 0, len(visits))
	for _, visit := range visits {
		bookings = append(bookings, u.serializer.Project(visit, session, at, false))
	}
	return selectPage(bookings, filter), nil
}

type filter struct {
	staffbookingusecase.Query
	page, pageSize int
}

func parseFilter(q staffbookingusecase.Query) (filter, error) {
	fields := failure.Validation()
	q.View = defaultTo(q.View, "list")
	q.Search = strings.ToLower(strings.TrimSpace(q.Search))
	q.Status = defaultTo(q.Status, "all")
	q.Sort = defaultTo(q.Sort, "priority")
	q.Direction = defaultTo(q.Direction, "asc")
	if !slices.Contains([]string{"list", "dashboard", "reception"}, q.View) {
		fields.Add("view", "Pilih tampilan daftar yang tersedia.")
	}
	if utf8.RuneCountInString(q.Search) > 200 {
		fields.Add("q", "Pencarian maksimal 200 karakter.")
	}
	if !slices.Contains([]string{"all", "pending", "approved", "missed", "rejected", "cancelled", "checked_in", "checked_out"}, q.Status) {
		fields.Add("status", "Pilih status kunjungan yang tersedia.")
	}
	if q.Date != "" {
		date, err := time.Parse(time.DateOnly, q.Date)
		if err != nil || date.Year() < 1 || date.Format(time.DateOnly) != q.Date {
			fields.Add("date", "Masukkan tanggal yang valid.")
		}
	}
	if q.VIP != "" && q.VIP != "0" && q.VIP != "1" {
		fields.Add("vip", "Pilih filter VIP yang tersedia.")
	}
	if !slices.Contains([]string{"priority", "name", "organization", "host", "time", "status"}, q.Sort) {
		fields.Add("sort", "Pilih urutan yang tersedia.")
	}
	if q.Direction != "asc" && q.Direction != "desc" {
		fields.Add("dir", "Pilih arah urutan yang tersedia.")
	}
	if (q.View == "reception" || q.Mode != "") && q.Mode != "checkin" && q.Mode != "checkout" {
		fields.Add("mode", "Pilih antrean kedatangan atau kepulangan.")
	}
	page, err := strconv.Atoi(defaultTo(q.Page, "1"))
	if err != nil || page < 1 || q.Page != "" && strconv.Itoa(page) != q.Page {
		fields.Add("page", "Masukkan nomor halaman yang valid.")
	}
	pageSize, err := strconv.Atoi(defaultTo(q.PageSize, "50"))
	if err != nil || pageSize < 1 || pageSize > 100 || q.PageSize != "" && strconv.Itoa(pageSize) != q.PageSize {
		fields.Add("pageSize", "Ukuran halaman harus antara 1 dan 100.")
	}
	return filter{Query: q, page: page, pageSize: pageSize}, fields.Err()
}

func defaultTo(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

type match struct {
	booking dto.StaffBooking
	rank    int
}

func selectPage(items []dto.StaffBooking, f filter) dto.StaffBookingPage {
	out := dto.StaffBookingPage{Items: []dto.StaffBooking{}, TotalScoped: len(items), Page: f.page, PageSize: f.pageSize}
	if f.View == "dashboard" {
		out.Metrics = &dto.StaffBookingMetrics{}
	} else if f.View == "list" {
		out.StatusCounts = &dto.StaffBookingCounts{}
	}
	filtered := make([]match, 0, len(items))
	for _, booking := range items {
		status := booking.Timing.DisplayStatus
		if out.Metrics != nil && !dashboard(out.Metrics, booking, f.Date) {
			continue
		}
		if f.View == "reception" && !inReception(booking, f) {
			continue
		}
		if f.View == "list" && (f.Date != "" && booking.Date != f.Date || f.VIP == "1" && (booking.VIP == nil || !*booking.VIP)) {
			continue
		}
		rank, found := searchRank(booking, f.Search)
		if !found {
			continue
		}
		if out.StatusCounts != nil {
			countStatus(out.StatusCounts, status)
			if f.Status != "all" && f.Status != status {
				continue
			}
		}
		filtered = append(filtered, match{booking, rank})
	}
	out.Total = len(filtered)
	if f.page-1 > out.Total/f.pageSize {
		return out
	}
	start := (f.page - 1) * f.pageSize
	if start == out.Total {
		return out
	}
	slices.SortFunc(filtered, func(a, b match) int { return compare(a, b, f) })
	for _, row := range filtered[start:min(start+f.pageSize, out.Total)] {
		out.Items = append(out.Items, row.booking)
	}
	return out
}

func inReception(b dto.StaffBooking, f filter) bool {
	if f.Mode == "checkout" {
		return b.Status == "checked_in"
	}
	return b.Date == f.Date && b.Timing.DisplayStatus == "approved"
}

func dashboard(m *dto.StaffBookingMetrics, b dto.StaffBooking, date string) bool {
	today, pending, inside := b.Date == date, b.Status == "pending", b.Status == "checked_in"
	if today {
		m.Today++
		if b.Timing.DisplayStatus == "approved" {
			m.Arrivals++
		}
		if b.Status == "checked_out" {
			m.Completed++
		}
	}
	if pending {
		m.Pending++
	}
	if inside {
		m.Inside++
	}
	return today || pending || inside
}

func countStatus(counts *dto.StaffBookingCounts, status string) {
	counts.All++
	switch status {
	case "pending":
		counts.Pending++
	case "approved":
		counts.Approved++
	case "missed":
		counts.Missed++
	case "rejected":
		counts.Rejected++
	case "cancelled":
		counts.Cancelled++
	case "checked_in":
		counts.CheckedIn++
	case "checked_out":
		counts.CheckedOut++
	}
}

func host(b dto.StaffBooking) string {
	if b.Officer != nil {
		return b.Officer.Name
	}
	return ""
}

func searchRank(b dto.StaffBooking, needle string) (int, bool) {
	if needle == "" || strings.HasPrefix(strings.ToLower(b.Name), needle) {
		return 0, true
	}
	contains := func(s string) bool { return strings.Contains(strings.ToLower(s), needle) }
	if contains(b.Name) {
		return 1, true
	}
	if slices.ContainsFunc(b.Visitors, contains) {
		return 2, true
	}
	if contains(b.Organization) {
		return 3, true
	}
	return 4, contains(host(b)) || contains(b.Code) || contains(b.Reference)
}

func priority(b dto.StaffBooking) int {
	if b.Status == "pending" {
		return 0
	}
	if b.Timing.DisplayStatus == "approved" || b.Status == "checked_in" {
		return 1
	}
	return 2
}

func compare(a, b match, f filter) int {
	if order := cmp.Compare(a.rank, b.rank); order != 0 {
		return order
	}
	x, y := a.booking, b.booking
	byTime := cmp.Compare(x.Timing.OpensAt, y.Timing.OpensAt)
	byID := strings.Compare(x.ID, y.ID)
	if f.Sort == "priority" {
		if group := cmp.Compare(priority(x), priority(y)); group != 0 {
			return group
		}
		if byTime != 0 && priority(x) == 2 {
			return -byTime
		}
		return cmp.Or(byTime, byID)
	}
	var value int
	text := func(x, y string) int { return strings.Compare(strings.ToLower(x), strings.ToLower(y)) }
	switch f.Sort {
	case "name":
		value = text(x.Name, y.Name)
	case "organization":
		value = text(x.Organization, y.Organization)
	case "host":
		value = text(host(x), host(y))
	case "status":
		value = text(x.Timing.DisplayStatus, y.Timing.DisplayStatus)
	}
	order := cmp.Or(value, byTime, byID)
	if f.Direction == "desc" {
		return -order
	}
	return order
}
