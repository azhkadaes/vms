package handler

import (
	"github.com/gofiber/fiber/v3"

	restmiddleware "github.com/azhkadaes/vms/backend/internal/server/rest/middleware"
	reportusecase "github.com/azhkadaes/vms/backend/internal/usecase/report"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type reportHandler struct{ usecase reportusecase.ReportUsecase }

func NewReportHandler(usecase reportusecase.ReportUsecase) *reportHandler {
	return &reportHandler{usecase: usecase}
}

func (h *reportHandler) Summary(c fiber.Ctx) error {
	filter, _, err := summaryFilter(c, false)
	if err != nil {
		return err
	}
	result, err := h.usecase.Summary(c.Context(), restmiddleware.Session(c), filter)
	if err != nil {
		return err
	}
	return c.JSON(result)
}

func (h *reportHandler) ExportSummary(c fiber.Ctx) error {
	filter, revision, err := summaryFilter(c, true)
	if err != nil {
		return err
	}
	result, err := h.usecase.ExportSummary(c.Context(), restmiddleware.Session(c), filter, revision)
	if err != nil {
		return err
	}
	return c.JSON(result)
}

func (h *reportHandler) VisitLog(c fiber.Ctx) error {
	filter, query, err := logFilter(c, false)
	if err != nil {
		return err
	}
	page, size := 1, 25
	if value, set, err := query.Int("page", 1, int(^uint(0)>>1)); err != nil {
		return err
	} else if set {
		page = value
	}
	if value, set, err := query.Int("pageSize", 1, int(^uint(0)>>1)); err != nil {
		return err
	} else if set {
		size = value
	}
	result, err := h.usecase.VisitLog(c.Context(), restmiddleware.Session(c), filter, page, size)
	if err != nil {
		return err
	}
	return c.JSON(result)
}

func (h *reportHandler) ExportVisitLog(c fiber.Ctx) error {
	filter, query, err := logFilter(c, true)
	if err != nil {
		return err
	}
	revision, err := query.String("revision")
	if err != nil {
		return err
	}
	result, err := h.usecase.ExportVisitLog(c.Context(), restmiddleware.Session(c), filter, revision)
	if err != nil {
		return err
	}
	return c.JSON(result)
}

func summaryFilter(c fiber.Ctx, export bool) (dto.SummaryFilter, string, error) {
	keys := []string{"from", "to", "status"}
	if export {
		keys = append(keys, "revision")
	}
	query, err := Query(c, keys...)
	if err != nil {
		return dto.SummaryFilter{}, "", err
	}
	from, err := query.String("from")
	if err != nil {
		return dto.SummaryFilter{}, "", err
	}
	to, err := query.String("to")
	if err != nil {
		return dto.SummaryFilter{}, "", err
	}
	revision, err := query.String("revision")
	return dto.SummaryFilter{From: from, To: to, Status: reportValues(query, "status", "all")}, revision, err
}

func logFilter(c fiber.Ctx, export bool) (dto.VisitLogFilter, QueryValues, error) {
	keys := []string{"from", "to", "q", "status", "kind", "officer", "room", "sort", "dir"}
	if export {
		keys = append(keys, "revision")
	} else {
		keys = append(keys, "page", "pageSize")
	}
	query, err := Query(c, keys...)
	if err != nil {
		return dto.VisitLogFilter{}, QueryValues{}, err
	}
	filter := dto.VisitLogFilter{Status: reportValues(query, "status", "all"), Kind: reportValues(query, "kind", "all"),
		Officer: reportValues(query, "officer", ""), Room: reportValues(query, "room", "")}
	for key, target := range map[string]*string{"from": &filter.From, "to": &filter.To, "q": &filter.Q, "sort": &filter.Sort, "dir": &filter.Dir} {
		*target, err = query.String(key)
		if err != nil {
			return dto.VisitLogFilter{}, QueryValues{}, err
		}
	}
	if filter.Sort == "" {
		filter.Sort = "date"
	}
	if filter.Dir == "" {
		filter.Dir = "desc"
	}
	return filter, query, nil
}

func reportValues(query QueryValues, key, fallback string) dto.ReportFilterValues {
	if values := query.Strings(key); len(values) > 0 {
		return values
	}
	return dto.ReportFilterValues{fallback}
}
