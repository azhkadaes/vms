package staffbookingusecaseimplementation

import (
	"slices"
	"testing"

	staffbookingusecase "github.com/azhkadaes/vms/backend/internal/usecase/staffbooking"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func TestStaffBookingViewCountsAndQueues(t *testing.T) {
	visit := func(id, date, status, display string, vip bool) dto.StaffBooking {
		return dto.StaffBooking{ID: id, Date: date, Status: status, VIP: &vip, Timing: dto.BookingTiming{DisplayStatus: display}}
	}
	today := "2026-11-19"
	yesterday := "2026-11-18"
	items := []dto.StaffBooking{
		visit("1", today, "approved", "missed", false),
		visit("2", today, "approved", "approved", true),
		visit("3", yesterday, "checked_in", "checked_in", false),
		visit("4", yesterday, "pending", "pending", false),
		visit("5", today, "checked_out", "checked_out", false),
	}
	list, err := parseFilter(staffbookingusecase.Query{View: "list", Status: "missed", VIP: "0", Page: "2", PageSize: "1"})
	if err != nil {
		t.Fatal(err)
	}
	page := selectPage(items, list)
	if page.Total != 1 || page.TotalScoped != 5 || len(page.Items) != 0 || page.StatusCounts.Missed != 1 || page.StatusCounts.All != 5 || page.StatusCounts.Approved != 1 {
		t.Fatalf("unexpected paged counts: %+v %+v", page, page.StatusCounts)
	}
	dashboard, _ := parseFilter(staffbookingusecase.Query{View: "dashboard", Date: today})
	page = selectPage(items, dashboard)
	if page.Total != 5 || *page.Metrics != (dto.StaffBookingMetrics{Today: 3, Pending: 1, Inside: 1, Arrivals: 1, Completed: 1}) {
		t.Fatalf("unexpected dashboard: %+v %+v", page, page.Metrics)
	}
	for mode, expected := range map[string]string{"checkin": "2", "checkout": "3"} {
		queue, _ := parseFilter(staffbookingusecase.Query{View: "reception", Date: today, Mode: mode})
		page = selectPage(items, queue)
		if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != expected || page.TotalScoped != 5 {
			t.Fatalf("%s queue: %+v", mode, page)
		}
	}
}

func TestListFiltersByStatusAndKindLists(t *testing.T) {
	visit := func(id, display, kind string, vip bool) dto.StaffBooking {
		return dto.StaffBooking{ID: id, VisitKind: kind, VIP: &vip, Timing: dto.BookingTiming{DisplayStatus: display}}
	}
	items := []dto.StaffBooking{
		visit("1", "pending", "standard", false),
		visit("2", "approved", "standard", true),
		visit("3", "approved", "restricted", false),
		visit("4", "rejected", "standard", false),
		visit("5", "pending", "restricted", false),
	}
	ids := func(q staffbookingusecase.Query) []string {
		t.Helper()
		f, err := parseFilter(q)
		if err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, b := range selectPage(items, f).Items {
			out = append(out, b.ID)
		}
		slices.Sort(out)
		return out
	}
	for name, tc := range map[string]struct {
		q    staffbookingusecase.Query
		want []string
	}{
		"two statuses":        {staffbookingusecase.Query{Status: "pending,approved"}, []string{"1", "2", "3", "5"}},
		"one kind":            {staffbookingusecase.Query{Kind: "restricted"}, []string{"3", "5"}},
		"vip and regular":     {staffbookingusecase.Query{Kind: "vip,standard"}, []string{"1", "2", "4"}},
		"status and kind":     {staffbookingusecase.Query{Status: "pending", Kind: "restricted"}, []string{"5"}},
		"all keeps every row": {staffbookingusecase.Query{Status: "all"}, []string{"1", "2", "3", "4", "5"}},
	} {
		if got := ids(tc.q); !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %v, want %v", name, got, tc.want)
		}
	}
	for _, bad := range []staffbookingusecase.Query{{Status: "pending,unknown"}, {Status: "all,pending"}, {Kind: "vvip"}, {Status: "pending,"}} {
		if _, err := parseFilter(bad); err == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
}

func TestBookingSearchMatchesOnlyProjectedFields(t *testing.T) {
	items := []dto.StaffBooking{
		{ID: "1", Name: "Custodian", Visitors: []string{"Visitor One"}, Timing: dto.BookingTiming{DisplayStatus: "approved"}},
		{ID: "2", Reference: "TRB-2", RestrictedView: "reception", Timing: dto.BookingTiming{DisplayStatus: "approved"}},
	}
	query, err := parseFilter(staffbookingusecase.Query{Search: "  vIsItOR  ", Sort: "name"})
	if err != nil {
		t.Fatal(err)
	}
	page := selectPage(items, query)
	if page.Total != 1 || page.Items[0].ID != "1" || page.TotalScoped != 2 || page.StatusCounts.All != 1 {
		t.Fatalf("unexpected projected search: %+v", page)
	}
}
