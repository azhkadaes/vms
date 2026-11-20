package employeeusecaseimplementation

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	employeerepository "github.com/azhkadaes/vms/backend/internal/repository/employee"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

// CreateEmployee is op 28. It refuses a session without people (403) and invalid fields (422)
// before hashing the initial password, then rechecks authority, the email and the PA assignment
// inside the transaction that stores the employee, its credential and the audit event.
func (u *usecase) CreateEmployee(ctx context.Context, session dto.Session, proof entity.SessionProof, req dto.EmployeeCreate) (dto.Employee, error) {
	ctx, span := u.tracer.Start(ctx, "employee.create")
	defer span.End()
	if !policy.CanAdministerEmployees(session) {
		return dto.Employee{}, forbidden()
	}
	draft, err := newEmployee(req)
	if err != nil {
		return dto.Employee{}, err
	}
	encoded, err := u.verifier.Hash(ctx, req.InitialPassword)
	if err != nil {
		return dto.Employee{}, fmt.Errorf("hash initial password: %w", err)
	}
	var created entity.Employee
	err = u.repo.Transaction(ctx, func(tx employeerepository.EmployeeTransaction) error {
		snapshot, err := u.authorize(tx, proof)
		if err != nil {
			return err
		}
		fields := failure.Validation()
		if err := checkEmail(tx, fields, draft.Email, 0); err != nil {
			return err
		}
		if err := checkAssignment(tx, fields, draft.Roles, draft.OfficerIDs, nil); err != nil {
			return err
		}
		if err := fields.Err(); err != nil {
			return err
		}
		created, err = tx.CreateEmployee(draft, encoded)
		if errors.Is(err, employeerepository.ErrEmailTaken) {
			return emailTaken()
		}
		if err != nil {
			return err
		}
		return tx.Audit(entity.EmployeeEvent{Actor: snapshot.EmployeeID, Action: "employee.created", After: created, At: snapshot.ServerTime})
	})
	if err != nil {
		return dto.Employee{}, fmt.Errorf("create employee: %w", err)
	}
	return u.serializer.ToEmployee(created), nil
}

// UpdateEmployee is op 29.
func (u *usecase) UpdateEmployee(ctx context.Context, session dto.Session, proof entity.SessionProof, id int32, req dto.EmployeeUpdate) (dto.Employee, error) {
	ctx, span := u.tracer.Start(ctx, "employee.update")
	defer span.End()
	if !policy.CanAdministerEmployees(session) {
		return dto.Employee{}, forbidden()
	}
	patch, err := employeePatch(req)
	if err != nil {
		return dto.Employee{}, err
	}
	var updated entity.Employee
	err = u.repo.Transaction(ctx, func(tx employeerepository.EmployeeTransaction) error {
		snapshot, err := u.authorize(tx, proof)
		if err != nil {
			return err
		}
		before, err := tx.LockEmployee(id)
		if err != nil {
			return err
		}
		if before.Version != req.Version {
			return failure.New(409, "version_conflict")
		}
		after := patch(before)
		current, requested := employeeAccess(before), employeeAccess(after)
		if !policy.MayChangeAccess(snapshot.EmployeeID, id, current, requested) {
			return forbidden()
		}
		fields := failure.Validation()
		if after.Email != before.Email {
			if err := checkEmail(tx, fields, after.Email, id); err != nil {
				return err
			}
		}
		if req.Roles != nil || req.AssignedOfficerIDs != nil {
			if err := checkAssignment(tx, fields, after.Roles, after.OfficerIDs, before.OfficerIDs); err != nil {
				return err
			}
		}
		if policy.RemovesLastSuperAdmin(current, requested, 0) {
			others, err := tx.OtherActiveSuperAdmins(id)
			if err != nil {
				return err
			}
			if others == 0 {
				field := "roles"
				if !after.Active {
					field = "active"
				}
				fields.Add(field, "Sistem harus memiliki minimal satu Super Admin aktif.")
			}
		}
		if err := fields.Err(); err != nil {
			return err
		}
		if sameEmployee(before, after) {
			updated = before
			return nil
		}
		updated, err = tx.SaveEmployee(before, after)
		if errors.Is(err, employeerepository.ErrEmailTaken) {
			return emailTaken()
		}
		if err != nil {
			return err
		}
		effect := policy.EmployeeSessionEffect(current, requested)
		switch {
		case effect.End:
			err = tx.EndSessions(id, snapshot.ServerTime)
		case effect.RoleRevision:
			err = tx.ReloadSessions(id, effect.RemovedRoles, snapshot.ServerTime)
		}
		if err != nil {
			return err
		}
		return tx.Audit(entity.EmployeeEvent{Actor: snapshot.EmployeeID, Action: "employee.updated", Before: &before, After: updated, At: snapshot.ServerTime})
	})
	if err != nil {
		return dto.Employee{}, fmt.Errorf("update employee: %w", err)
	}
	return u.serializer.ToEmployee(updated), nil
}

func employeeAccess(e entity.Employee) policy.EmployeeAccess {
	return policy.EmployeeAccess{Active: e.Active, Roles: e.Roles, OfficerIDs: e.OfficerIDs}
}

func sameEmployee(a, b entity.Employee) bool {
	return a.Name == b.Name && a.Email == b.Email && a.Active == b.Active && a.JobTitle == b.JobTitle &&
		a.Deputy == b.Deputy && a.Directorate == b.Directorate && a.Phone == b.Phone && a.NIP == b.NIP &&
		slices.Equal(a.Roles, b.Roles) && slices.Equal(a.OfficerIDs, b.OfficerIDs)
}

func employeePatch(req dto.EmployeeUpdate) (func(entity.Employee) entity.Employee, error) {
	fields := failure.Validation()
	if req.Version <= 0 {
		fields.Add("version", "Versi pegawai wajib diisi.")
	}
	if req.Name.Set {
		req.Name.Value = checkName(fields, req.Name.Value)
	}
	if req.Email.Set {
		req.Email.Value = checkEmailFormat(fields, req.Email.Value)
	}
	if req.Active.Null {
		fields.Add("active", "Status akun wajib diisi.")
	}
	if req.Roles != nil {
		req.Roles = checkRoles(fields, req.Roles)
	}
	var officerIDs []int32
	if req.AssignedOfficerIDs != nil {
		officerIDs = checkOfficerIDs(fields, req.AssignedOfficerIDs)
	}
	req.JobTitle.Value = checkProfile(fields, "jobTitle", req.JobTitle.Value)
	req.Deputy.Value = checkProfile(fields, "deputy", req.Deputy.Value)
	req.Directorate.Value = checkProfile(fields, "directorate", req.Directorate.Value)
	req.Phone.Value = checkProfile(fields, "phone", req.Phone.Value)
	req.NIP.Value = checkProfile(fields, "nip", req.NIP.Value)

	return func(e entity.Employee) entity.Employee {
		e.Name = updateValue(e.Name, req.Name)
		e.Email = updateValue(e.Email, req.Email)
		e.Active = updateValue(e.Active, req.Active)
		e.JobTitle = updateValue(e.JobTitle, req.JobTitle)
		e.Deputy = updateValue(e.Deputy, req.Deputy)
		e.Directorate = updateValue(e.Directorate, req.Directorate)
		e.Phone = updateValue(e.Phone, req.Phone)
		e.NIP = updateValue(e.NIP, req.NIP)
		if req.Roles != nil {
			e.Roles = req.Roles
		}
		if officerIDs != nil {
			e.OfficerIDs = officerIDs
		}
		return e
	}, fields.Err()
}

func updateValue[T any](current T, patch dto.Optional[T]) T {
	if patch.Set {
		return patch.Value
	}
	return current
}

// authorize rechecks the caller inside a write: lock the caller's employee, validate the locked
// session against the token proof, then require people from that live authority.
func (u *usecase) authorize(tx employeerepository.EmployeeTransaction, proof entity.SessionProof) (entity.SessionSnapshot, error) {
	snapshot, err := tx.LockAuthority(proof)
	if err != nil {
		return entity.SessionSnapshot{}, err
	}
	session, err := u.validator.Validate(snapshot, proof)
	if err != nil {
		return entity.SessionSnapshot{}, err
	}
	if !policy.CanAdministerEmployees(session) {
		return entity.SessionSnapshot{}, forbidden()
	}
	return snapshot, nil
}

const emailTakenMessage = "Email sudah digunakan pegawai lain."

func emailTaken() error { return failure.Validation().Add("email", emailTakenMessage).Err() }

// checkEmail adds a field error when an employee other than except already uses email.
func checkEmail(tx employeerepository.EmployeeTransaction, fields *failure.ValidationBuilder, email string, except int32) error {
	taken, err := tx.EmailInUse(email, except)
	if err != nil {
		return err
	}
	if taken {
		fields.Add("email", emailTakenMessage)
	}
	return nil
}

// checkAssignment adds a field error when the requested assignment breaks A-16. current is the
// assignment the employee holds now; keeping it does not require the official to be active.
func checkAssignment(tx employeerepository.EmployeeTransaction, fields *failure.ValidationBuilder, roles []string, requested, current []int32) error {
	if len(requested) == 0 {
		return nil
	}
	found, err := tx.Officers(requested)
	if err != nil {
		return err
	}
	officers := make([]policy.AssignableOfficer, 0, len(requested))
	for _, id := range requested {
		officer := policy.AssignableOfficer{ID: id}
		for _, row := range found {
			if row.ID == id {
				officer = policy.AssignableOfficer{ID: id, Found: true, VIP: row.VIP, Active: row.Active}
			}
		}
		officers = append(officers, officer)
	}
	if message := policy.AssignmentProblem(roles, officers, current); message != "" {
		fields.Add("assignedOfficerIds", message)
	}
	return nil
}

// roleOrder is vms.role ID order, in which roles are stored and returned.
var roleOrder = []string{policy.RolePA, policy.RoleProtocol, policy.RoleReceptionist, policy.RoleSuperAdmin}

// checkRoles requires at least one known role, each once, and returns them in role order.
func checkRoles(fields *failure.ValidationBuilder, roles []string) []string {
	if len(roles) == 0 {
		fields.Add("roles", "Pilih minimal satu peran.")
		return nil
	}
	for i, role := range roles {
		if !policy.KnownRole(role) {
			fields.Add("roles", "Pilih peran dari daftar yang tersedia.")
		} else if slices.Contains(roles[:i], role) {
			fields.Add("roles", "Setiap peran hanya boleh dipilih sekali.")
		}
	}
	out := []string{}
	for _, role := range roleOrder {
		if slices.Contains(roles, role) {
			out = append(out, role)
		}
	}
	return out
}

// checkOfficerIDs parses assignment IDs as canonical positive integers.
func checkOfficerIDs(fields *failure.ValidationBuilder, raw []string) []int32 {
	if len(raw) > 1 {
		fields.Add("assignedOfficerIds", "Pegawai hanya dapat ditugaskan ke satu pejabat.")
		return nil
	}
	out := []int32{}
	for _, value := range raw {
		id, err := strconv.ParseInt(value, 10, 32)
		if err != nil || id <= 0 || strconv.FormatInt(id, 10) != value {
			fields.Add("assignedOfficerIds", "Pilih pejabat dari daftar yang tersedia.")
			continue
		}
		out = append(out, int32(id))
	}
	return out
}

// checkName trims the name and requires 2-255 characters, as the vms.pegawai check does.
func checkName(fields *failure.ValidationBuilder, raw string) string {
	name := strings.TrimSpace(raw)
	if n := utf8.RuneCountInString(name); n < 2 || n > maxText {
		fields.Add("name", "Masukkan nama lengkap (2-255 karakter).")
	}
	return name
}

// checkEmailFormat normalizes the email as vms.pegawai stores it.
func checkEmailFormat(fields *failure.ValidationBuilder, raw string) string {
	email, ok := validate.Email(raw)
	if !ok {
		fields.Add("email", "Periksa email pegawai.")
	}
	return email
}

// checkProfile trims an optional profile field; "" leaves it empty.
func checkProfile(fields *failure.ValidationBuilder, key, raw string) string {
	value := strings.TrimSpace(raw)
	if utf8.RuneCountInString(value) > maxText {
		fields.Add(key, "Maksimal 255 karakter.")
	}
	return value
}

// newEmployee validates op 28's fields on their own and returns the employee to store.
func newEmployee(req dto.EmployeeCreate) (entity.Employee, error) {
	fields := failure.Validation()
	out := entity.Employee{Name: checkName(fields, req.Name), Email: checkEmailFormat(fields, req.Email), Roles: checkRoles(fields, req.Roles),
		JobTitle: checkProfile(fields, "jobTitle", req.JobTitle), Deputy: checkProfile(fields, "deputy", req.Deputy),
		Directorate: checkProfile(fields, "directorate", req.Directorate), Phone: checkProfile(fields, "phone", req.Phone),
		NIP: checkProfile(fields, "nip", req.NIP), OfficerIDs: checkOfficerIDs(fields, req.AssignedOfficerIDs)}
	if req.Active == nil {
		fields.Add("active", "Status akun wajib diisi.")
	} else {
		out.Active = *req.Active
	}
	if req.RequirePasswordChange == nil {
		fields.Add("requirePasswordChange", "Tentukan apakah pegawai wajib mengganti kata sandi.")
	} else {
		out.MustChangePassword = *req.RequirePasswordChange
	}
	if !policy.AcceptablePassword(req.InitialPassword) {
		fields.Add("initialPassword", fmt.Sprintf("Kata sandi awal minimal %d karakter dan maksimal %d byte.", policy.PasswordMinLength, policy.PasswordMaxBytes))
	}
	return out, fields.Err()
}
