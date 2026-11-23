package employeeusecaseimplementation

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	employeerepository "github.com/azhkadaes/vms/backend/internal/repository/employee"
	employeeserializer "github.com/azhkadaes/vms/backend/internal/serializer/employee"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	employeeusecase "github.com/azhkadaes/vms/backend/internal/usecase/employee"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// Paging and text limits of employee administration.
const (
	defaultPageSize = 25
	maxPageSize     = 250
	// maxPage keeps the list offset far inside the integer range.
	maxPage = 1_000_000
	// maxText is the 255-character limit of every employee text column and of the search.
	maxText = 255
)

type usecase struct {
	repo       employeerepository.EmployeeRepository
	serializer employeeserializer.EmployeeSerializer
	validator  authusecase.SessionValidator
	verifier   authusecase.CredentialVerifier
	tracer     trace.Tracer
}

var _ employeeusecase.EmployeeUsecase = (*usecase)(nil)

// NewEmployeeUsecase composes employee administration. The validator rechecks authority inside
// writes; the verifier hashes initial passwords with the shared bcrypt work bound.
func NewEmployeeUsecase(repo employeerepository.EmployeeRepository, serializer employeeserializer.EmployeeSerializer, validator authusecase.SessionValidator,
	verifier authusecase.CredentialVerifier, tracer trace.Tracer) employeeusecase.EmployeeUsecase {
	return &usecase{repo: repo, serializer: serializer, validator: validator, verifier: verifier, tracer: tracer}
}

func forbidden() error { return failure.New(403, "forbidden") }

// GetEmployees refuses a session without people, then invalid paging (422), before reading.
func (u *usecase) GetEmployees(ctx context.Context, session dto.Session, query employeeusecase.EmployeeQuery) (dto.EmployeePage, error) {
	ctx, span := u.tracer.Start(ctx, "employee.list")
	defer span.End()
	if !policy.CanAdministerEmployees(session) {
		return dto.EmployeePage{}, forbidden()
	}
	filter, err := employeeFilter(query)
	if err != nil {
		return dto.EmployeePage{}, err
	}
	page, err := u.repo.GetEmployees(ctx, filter)
	if err != nil {
		return dto.EmployeePage{}, fmt.Errorf("list employees: %w", err)
	}
	return u.serializer.ToEmployeePage(page, filter), nil
}

// employeeFilter validates op 26's query: page from 1, pageSize 1-250 (default 25), and a search
// of at most 255 characters after trimming. Paging errors are field errors (422), not 400.
func employeeFilter(query employeeusecase.EmployeeQuery) (entity.EmployeeFilter, error) {
	filter := entity.EmployeeFilter{Query: strings.TrimSpace(query.Q), Page: 1, PageSize: defaultPageSize}
	fields := failure.Validation()
	if !utf8.ValidString(filter.Query) || utf8.RuneCountInString(filter.Query) > maxText {
		fields.Add("q", "Kata kunci pencarian maksimal 255 karakter.")
	}
	if query.Page != "" {
		page, ok := bounded(query.Page, 1, maxPage)
		if !ok {
			fields.Add("page", "Halaman harus berupa bilangan bulat positif.")
		}
		filter.Page = page
	}
	if query.PageSize != "" {
		size, ok := bounded(query.PageSize, 1, maxPageSize)
		if !ok {
			fields.Add("pageSize", "Jumlah per halaman harus antara 1 dan 250.")
		}
		filter.PageSize = size
	}
	return filter, fields.Err()
}

// bounded parses a canonical decimal (no sign, no leading zero) within [min, max].
func bounded(raw string, min, max int) (int, bool) {
	n, err := strconv.Atoi(raw)
	return n, err == nil && strconv.Itoa(n) == raw && n >= min && n <= max
}

// GetEmployee allows staff to read their own record. Other records require directory access.
func (u *usecase) GetEmployee(ctx context.Context, session dto.Session, id int32) (dto.Employee, error) {
	ctx, span := u.tracer.Start(ctx, "employee.get")
	defer span.End()
	self := policy.WorkspaceReady(session) && session.ID == strconv.FormatInt(int64(id), 10)
	if !self && !policy.CanAdministerEmployees(session) {
		return dto.Employee{}, forbidden()
	}
	employee, err := u.repo.GetEmployee(ctx, id)
	if err != nil {
		return dto.Employee{}, fmt.Errorf("get employee: %w", err)
	}
	return u.serializer.ToEmployee(employee), nil
}
