# Backend interface contract

Current source signatures below are inspected, not assumed. Proposed signatures are documentation only: no interface stubs, second backend or missing-handler placeholders were created. Constructors return the module interface; implementations belong in the matching implementation subpackage. Handler constructors receive the usecase interface; routers are the composition root; repositories own GORM; serializers map explicit allowed fields into DTOs. These are the actual SIHP conventions retained by the current VMS modules.

Phase 1 retains Fiber3, direct DTO/error returns and INTEGER IDs/string wire IDs. It uses a signed access JWT plus a current database account/single-role/PA-assignment read, without stored login sessions. Existing source session signatures below are legacy inventory; the proposed Phase 1 signatures replace their active use, within the same modules. SIHP Req/Res DTO prefixes are reference naming, while existing VMS DTOs use semantic names; preserve existing names and use consistent semantic names for new modules.

## Existing source inventory (auth is superseded)

### internal/repository/auth/auth_repository.go:AuthRepository

```go
GetSessionSnapshot(ctx context.Context, sessionID string) (entity.SessionSnapshot, error)
```

### internal/repository/auth/session_repository.go:SessionRepository

```go
AuthRepository
FindRefresh(ctx context.Context, hash string) (string, error)
FinishLogin(ctx context.Context, reservation string, success bool) error
GetCredential(ctx context.Context, email string) (entity.EmployeeCredential, error)
ReserveLogin(ctx context.Context, ipKey, accountKey string) (string, int, error)
Transaction(ctx context.Context, employeeID int32, operation func(tx SessionTransaction) error) error
```

### internal/repository/auth/session_repository.go:SessionTransaction

```go
Audit(employeeID int32, action string) error
ClearRecoveries(sessionID string) error
Create(session entity.StaffSession) error
Credential(employeeID int32) (entity.EmployeeCredential, error)
Now() time.Time
Recovery(sessionID, requestID string) (entity.SessionRecovery, error)
RememberRefresh(sessionID, hash string) error
Roles(employeeID int32) (map[string]int32, error)
Save(session entity.StaffSession) error
SaveRecovery(recovery entity.SessionRecovery) error
Session(id string) (entity.StaffSession, error)
SetPassword(employeeID int32, encodedHash string, currentSessionID string) error
```

### internal/repository/catalog/catalog_repository.go:CatalogRepository

```go
GetPublicCatalogs(ctx context.Context) (entity.PublicCatalog, error)
```

### internal/serializer/auth/auth_serializer.go:AuthSerializer

```go
ToSession(snapshot entity.SessionSnapshot) dto.Session
```

### internal/serializer/catalog/catalog_serializer.go:CatalogSerializer

```go
ToPublicCatalog(in entity.PublicCatalog) dto.PublicCatalog
```

### internal/usecase/auth/auth_usecase.go:AuthUsecase

```go
GetSession(ctx context.Context, proof entity.SessionProof) (dto.Session, error)
```

### internal/usecase/auth/credential.go:CredentialVerifier

```go
Hash(ctx context.Context, password string) (string, error)
Verify(ctx context.Context, encodedHash, password string) (bool, error)
```

### internal/usecase/auth/session_usecase.go:SessionUsecase

```go
Binding(ctx context.Context, secret string, allowConsumed bool) (string, time.Time, error)
ChangePassword(ctx context.Context, proof entity.SessionProof, request dto.PasswordChange) (CredentialResult, error)
Login(ctx context.Context, request dto.Login, ip string) (CredentialResult, error)
Logout(ctx context.Context, secret string) error
Refresh(ctx context.Context, secret, requestID string) (CredentialResult, error)
Renew(ctx context.Context, proof entity.SessionProof) (dto.Session, error)
SwitchRole(ctx context.Context, proof entity.SessionProof, role string) (CredentialResult, error)
```

### internal/usecase/catalog/catalog_usecase.go:CatalogUsecase

```go
GetPublicCatalogs(ctx context.Context) (dto.PublicCatalog, error)
```

## Development sequence and type boundaries

Follow the inspected SIHP composition and layer boundaries: confirm the frontend action and selected contract; define the typed request/response and usecase/repository/serializer interfaces; implement them behind injected interfaces; wire the router explicitly; then verify contract and real behavior. Source code establishes these boundaries, not the historical order in which SIHP was written.

Interface implementation slices must include a compile-time assertion, an interface-returning constructor, typed handler input, and explicit serializer projection. The repository owns database access and its transaction-bound connection; usecases coordinate validation and policy. Existing SessionRepository callback signatures are source evidence only; Phase 1 must not depend on session or refresh state. Visit/integration transactions still belong to repositories.

Omitted, empty and null are distinct wire cases. Future Go DTOs must preserve presence for fields whose omission preserves and whose empty value clears; a plain string with a zero-value fallback cannot implement that distinction. No request schema accepts null. Discriminated standard/restricted inputs and projections must retain their variant validation instead of one permissive bag of optional fields.

Aliases: current dto.PublicCatalog maps to wire Catalogs; integration reads are arrays. Phase 1 Login returns dto.AuthResult directly, containing an access token, expiry and dto.CurrentUser. No CredentialResult cookie wrapper is part of the target. PrivateFile is a bounded private stream, not a JSON DTO. Query DTOs describe parsed query parameters, not GET bodies.

Proposed entity/DTO names below are interface design obligations. Their Go representation must be defined and compiled in the owning slice; resolving a documentation symbol does not prove the future type exists. OPEN-READ-ORDER remains an active decision. The old cookie/logout-proof issue is deferred and does not apply to local-only Phase 1 logout.

## Proposed additions (not implemented)

Signatures specify reviewable boundaries within existing layers. Domain entity structs are to be introduced with their owning slices; repository entities are not HTTP DTO aliases. All resource usecases must derive current authority from proof and recheck it transactionally for writes. Constructors follow NewVisitUsecase/NewVisitRepository/NewVisitSerializer etc.; router dependencies are injected explicitly. Do not pass a Fiber context to a repository or hide a GORM transaction in context values.

CommandResult[T] is an internal result containing Value T and Replayed bool, letting handlers select201 versus200 without duplicate usecase methods. SubmissionReference includes audience, principal/email binding, UUID, normalized digest and resolve/close intent; ResolveSubmission/CreateVisit coordinate outcome linkage in the same transaction. VisitActor identifies authenticated proof or public pair/reference without granting authority from an actor label alone. ScheduleChange carries complete old/new resource interval data. ResourceKey order is stable by type/ID. AttendanceCommand carries visitID/version/credential/action and no fabricated identityVerified field.

PrivateFile contains an io.ReadCloser plus verified MIME/safe display filename/size; it never serializes a private path into JSON. Multipart DTO Upload is decoded into a bounded reader plus reference/declared metadata before StageUpload; field/token ownership and content verification remain required. AttachmentQuery contains disposition=inline|attachment default inline. ReportQuery is from/to/status and ReportExportQuery adds revision. BookingQuery is the typed view union; EmployeeQuery contains q/page/pageSize. These query structs are not extra request-body schemas.

Repositories read current authorized aggregates, but serializers still enforce per-field projection. Phase 1 permissions come from the fixed backend role mapping; no editable policy graph or generic updateMap is exposed. Transaction methods use one transaction-bound connection; AppendAudit/EnqueueNotifications are atomic parts of the command, not unrelated external calls. StageUpload cannot satisfy visit linkage by itself; CreateVisit coordinates owned verified linkage and cleanup exclusion within its aggregate transaction.

### internal/usecase/catalog/catalog_usecase.go:CatalogUsecase

```go
GetStaffCatalogs(ctx context.Context, proof entity.AccessProof) (dto.PublicCatalog, error)
```

### internal/usecase/visit/visit_usecase.go:VisitUsecase

```go
CreatePublicBooking(ctx context.Context, request dto.PublicCreate) (entity.CommandResult[dto.PublicBooking], error)
ResolveSubmission(ctx context.Context, request dto.SubmissionResolve) (dto.Resolution, error)
GetPublicBooking(ctx context.Context, request dto.Pair) (dto.PublicBooking, error)
CancelPublicBooking(ctx context.Context, id int32, request dto.PairVersion) (dto.PublicBooking, error)
GetBookings(ctx context.Context, proof entity.AccessProof, request dto.BookingQuery) (dto.ReadModel, error)
GetBooking(ctx context.Context, proof entity.AccessProof, id int32) (dto.StaffBooking, error)
CreateStaffBooking(ctx context.Context, proof entity.AccessProof, request dto.StaffCreate) (entity.CommandResult[dto.StaffBooking], error)
Approve(ctx context.Context, proof entity.AccessProof, id int32, request dto.Approve) (dto.StaffBooking, error)
Reject(ctx context.Context, proof entity.AccessProof, id int32, request dto.Reject) (dto.StaffBooking, error)
Reschedule(ctx context.Context, proof entity.AccessProof, id int32, request dto.Reschedule) (dto.StaffBooking, error)
UpdateRestricted(ctx context.Context, proof entity.AccessProof, id int32, request dto.RestrictedUpdate) (dto.RestrictedFull, error)
CancelStaffBooking(ctx context.Context, proof entity.AccessProof, id int32, request dto.StaffCancel) (dto.StaffBooking, error)
```

### internal/usecase/reception/reception_usecase.go:ReceptionUsecase

```go
IssueAdmission(ctx context.Context, id int32, request dto.PairVersion) (dto.AdmissionResult, error)
Checkin(ctx context.Context, proof entity.AccessProof, id int32, request dto.Attendance) (dto.ArrivalBooking, error)
Checkout(ctx context.Context, proof entity.AccessProof, id int32, request dto.Attendance) (dto.ArrivalBooking, error)
Resolve(ctx context.Context, proof entity.AccessProof, request dto.Resolve) (dto.ArrivalBooking, error)
```

### internal/usecase/document/document_usecase.go:DocumentUsecase

```go
UploadPublic(ctx context.Context, request dto.Upload) (entity.CommandResult[dto.UploadResult], error)
UploadStaff(ctx context.Context, proof entity.AccessProof, request dto.Upload) (entity.CommandResult[dto.UploadResult], error)
GetAttachment(ctx context.Context, proof entity.AccessProof, id int32, request dto.AttachmentQuery) (entity.PrivateFile, error)
```

### internal/usecase/management/management_usecase.go:ManagementUsecase

```go
GetEmployees(ctx context.Context, proof entity.AccessProof, request dto.EmployeeQuery) (dto.EmployeePage, error)
GetEmployee(ctx context.Context, proof entity.AccessProof, id int32) (dto.Employee, error)
GetIntegrations(ctx context.Context, proof entity.AccessProof) ([]dto.Integration, error)
UpdateIntegration(ctx context.Context, proof entity.AccessProof, id int32, request dto.IntegrationPatch) (dto.Integration, error)
```

### internal/usecase/report/report_usecase.go:ReportUsecase

```go
GetSummary(ctx context.Context, proof entity.AccessProof, request dto.ReportQuery) (dto.Summary, error)
Export(ctx context.Context, proof entity.AccessProof, request dto.ReportExportQuery) (dto.Export, error)
```

### internal/repository/catalog/catalog_repository.go:CatalogRepository

```go
GetStaffCatalogs(ctx context.Context, proof entity.AccessProof) (entity.PublicCatalog, error)
```

### internal/repository/visit/visit_repository.go:VisitRepository

```go
GetPublicBooking(ctx context.Context, pair entity.PublicPair) (entity.VisitAggregate, error)
GetBookings(ctx context.Context, proof entity.AccessProof, filter entity.BookingFilter) (entity.VisitReadModel, error)
GetBooking(ctx context.Context, proof entity.AccessProof, id int32) (entity.VisitAggregate, error)
Transaction(ctx context.Context, actor entity.VisitActor, operation func(tx VisitTransaction) error) error
```

### internal/repository/visit/visit_repository.go:VisitTransaction

```go
Now() time.Time
ResolveSubmission(reference entity.SubmissionReference) (entity.SubmissionOutcome, error)
CreateVisit(input entity.VisitCreate) (entity.VisitAggregate, error)
LockVisit(id int32, version int32) (entity.VisitAggregate, error)
LockResources(resources []entity.ResourceKey) error
CheckOverlap(schedule entity.ScheduleChange) error
SaveVisit(visit entity.VisitAggregate) (entity.VisitAggregate, error)
RevokeAdmission(visitID int32) error
ReplaceAdmission(input entity.AdmissionIssue) (entity.AdmissionCredential, error)
ConfirmAttendance(input entity.AttendanceCommand) (entity.VisitAggregate, error)
AppendAudit(event entity.AuditEvent) error
EnqueueNotifications(events []entity.NotificationIntent) error
```

### internal/repository/reception/reception_repository.go:ReceptionRepository

```go
ResolveCredential(ctx context.Context, proof entity.AccessProof, credential string) (entity.VisitAggregate, error)
```

### internal/repository/document/document_repository.go:DocumentRepository

```go
StageUpload(ctx context.Context, owner entity.UploadOwner, input entity.UploadInput) (entity.CommandResult[entity.StagedUpload], error)
GetAttachment(ctx context.Context, proof entity.AccessProof, visitID int32) (entity.PrivateFile, error)
AuditAccess(ctx context.Context, event entity.FileAccess) error
```

### internal/repository/management/management_repository.go:ManagementRepository

```go
GetEmployees(ctx context.Context, proof entity.AccessProof, filter entity.EmployeeFilter) (entity.EmployeePage, error)
GetEmployee(ctx context.Context, proof entity.AccessProof, id int32) (entity.EmployeeAggregate, error)
GetIntegrations(ctx context.Context, proof entity.AccessProof) ([]entity.Integration, error)
Transaction(ctx context.Context, proof entity.AccessProof, operation func(tx ManagementTransaction) error) error
```

### internal/repository/management/management_repository.go:ManagementTransaction

```go
UpdateIntegration(id int32, input entity.IntegrationChange) (entity.Integration, error)
AppendAudit(event entity.AuditEvent) error
```

### internal/repository/report/report_repository.go:ReportRepository

```go
GetSummary(ctx context.Context, proof entity.AccessProof, filter entity.ReportFilter) (entity.SummarySnapshot, error)
```

### internal/serializer/catalog/catalog_serializer.go:CatalogSerializer

```go
ToStaffCatalog(in entity.PublicCatalog) dto.PublicCatalog
```

### internal/serializer/visit/visit_serializer.go:VisitSerializer

```go
ToPublicBooking(in entity.VisitAggregate) dto.PublicBooking
ToStaffBooking(in entity.VisitAggregate, authority entity.VisitAuthority) dto.StaffBooking
ToReadModel(in entity.VisitReadModel, authority entity.VisitAuthority) dto.ReadModel
ToResolution(in entity.SubmissionOutcome) dto.Resolution
ToRestrictedFull(in entity.VisitAggregate, authority entity.VisitAuthority) dto.RestrictedFull
```

### internal/serializer/reception/reception_serializer.go:ReceptionSerializer

```go
ToAdmissionResult(in entity.AdmissionCredential) dto.AdmissionResult
ToArrivalBooking(in entity.VisitAggregate, authority entity.VisitAuthority) dto.ArrivalBooking
```

### internal/serializer/document/document_serializer.go:DocumentSerializer

```go
ToUploadResult(in entity.StagedUpload) dto.UploadResult
```

### internal/serializer/management/management_serializer.go:ManagementSerializer

```go
ToEmployee(in entity.EmployeeAggregate) dto.Employee
ToEmployeePage(in entity.EmployeePage) dto.EmployeePage
ToIntegration(in entity.Integration) dto.Integration
```

### internal/serializer/report/report_serializer.go:ReportSerializer

```go
ToSummary(in entity.SummarySnapshot) dto.Summary
ToExport(in entity.SummarySnapshot) dto.Export
```

### internal/usecase/auth/auth_usecase.go:AuthUsecase

```go
Login(ctx context.Context, request dto.Login, clientIP string) (dto.AuthResult, error)
GetCurrentUser(ctx context.Context, proof entity.AccessProof) (dto.CurrentUser, error)
```

### internal/repository/auth/auth_repository.go:AuthRepository

```go
FindCredentialByEmail(ctx context.Context, email string) (entity.EmployeeCredential, error)
GetCurrentAuthority(ctx context.Context, employeeID int32) (entity.CurrentAuthority, error)
```

### internal/serializer/auth/auth_serializer.go:AuthSerializer

```go
ToCurrentUser(in entity.CurrentAuthority) dto.CurrentUser
ToAuthResult(user dto.CurrentUser, accessToken string, accessExpiresAt time.Time) dto.AuthResult
```

## Per-operation mappings

| HTTP operation | Handler | Usecase | Request -> response DTO | Serializer | Repository operations | State |
| --- | --- | --- | --- | --- | --- | --- |
| POST /v1/staff/auth/login | auth_handler.go:Login | AuthUsecase.Login | Login -> AuthResult | AuthSerializer.ToAuthResult | AuthRepository.FindCredentialByEmail, AuthRepository.GetCurrentAuthority | proposed |
| GET /v1/staff/auth/me | auth_handler.go:Me | AuthUsecase.GetCurrentUser | none -> CurrentUser | AuthSerializer.ToCurrentUser | AuthRepository.GetCurrentAuthority | proposed |
| GET /v1/public/catalogs | catalog_handler.go:PublicCatalogs | CatalogUsecase.GetPublicCatalogs | none -> PublicCatalog | CatalogSerializer.ToPublicCatalog | CatalogRepository.GetPublicCatalogs | source registered |
| GET /v1/staff/catalogs | catalog_handler.go:StaffCatalogs | CatalogUsecase.GetStaffCatalogs | none -> PublicCatalog | CatalogSerializer.ToStaffCatalog | CatalogRepository.GetStaffCatalogs | proposed |
| POST /v1/public/bookings | visit_handler.go:PublicCreate | VisitUsecase.CreatePublicBooking | PublicCreate -> PublicBooking | VisitSerializer.ToPublicBooking | VisitRepository.Transaction, VisitTransaction.ResolveSubmission, VisitTransaction.CreateVisit, VisitTransaction.AppendAudit, VisitTransaction.EnqueueNotifications | proposed |
| POST /v1/public/submissions/resolve | visit_handler.go:ResolveSubmission | VisitUsecase.ResolveSubmission | SubmissionResolve -> Resolution | VisitSerializer.ToResolution | VisitRepository.Transaction, VisitTransaction.ResolveSubmission | proposed |
| POST /v1/public/lookup | visit_handler.go:PublicLookup | VisitUsecase.GetPublicBooking | Pair -> PublicBooking | VisitSerializer.ToPublicBooking | VisitRepository.GetPublicBooking | proposed |
| POST /v1/public/bookings/{id}/cancel | visit_handler.go:PublicCancel | VisitUsecase.CancelPublicBooking | PairVersion -> PublicBooking | VisitSerializer.ToPublicBooking | VisitRepository.Transaction, VisitTransaction.LockVisit, VisitTransaction.SaveVisit, VisitTransaction.RevokeAdmission, VisitTransaction.AppendAudit, VisitTransaction.EnqueueNotifications | proposed |
| POST /v1/public/bookings/{id}/admission | reception_handler.go:PublicAdmission | ReceptionUsecase.IssueAdmission | PairVersion -> AdmissionResult | ReceptionSerializer.ToAdmissionResult | VisitRepository.Transaction, VisitTransaction.LockVisit, VisitTransaction.ReplaceAdmission | proposed |
| POST /v1/public/attachments | document_handler.go:PublicUpload | DocumentUsecase.UploadPublic | Upload -> UploadResult | DocumentSerializer.ToUploadResult | DocumentRepository.StageUpload | proposed |
| POST /v1/staff/attachments | document_handler.go:StaffUpload | DocumentUsecase.UploadStaff | Upload -> UploadResult | DocumentSerializer.ToUploadResult | DocumentRepository.StageUpload | proposed |
| GET /v1/staff/bookings/{id}/attachment | document_handler.go:StaffAttachment | DocumentUsecase.GetAttachment | AttachmentQuery -> PrivateFile | no JSON projection | DocumentRepository.GetAttachment, DocumentRepository.AuditAccess | proposed |
| GET /v1/staff/bookings | visit_handler.go:List | VisitUsecase.GetBookings | BookingQuery -> ReadModel | VisitSerializer.ToReadModel | VisitRepository.GetBookings | proposed |
| GET /v1/staff/bookings/{id} | visit_handler.go:Detail | VisitUsecase.GetBooking | none -> StaffBooking | VisitSerializer.ToStaffBooking | VisitRepository.GetBooking | proposed |
| POST /v1/staff/bookings | visit_handler.go:StaffCreate | VisitUsecase.CreateStaffBooking | StaffCreate -> StaffBooking | VisitSerializer.ToStaffBooking | VisitRepository.Transaction, VisitTransaction.ResolveSubmission, VisitTransaction.LockResources, VisitTransaction.CheckOverlap, VisitTransaction.CreateVisit, VisitTransaction.AppendAudit, VisitTransaction.EnqueueNotifications | proposed |
| POST /v1/staff/bookings/{id}/approve | visit_handler.go:Approve | VisitUsecase.Approve | Approve -> StaffBooking | VisitSerializer.ToStaffBooking | VisitRepository.Transaction, VisitTransaction.LockVisit, VisitTransaction.LockResources, VisitTransaction.CheckOverlap, VisitTransaction.SaveVisit, VisitTransaction.AppendAudit, VisitTransaction.EnqueueNotifications | proposed |
| POST /v1/staff/bookings/{id}/reject | visit_handler.go:Reject | VisitUsecase.Reject | Reject -> StaffBooking | VisitSerializer.ToStaffBooking | VisitRepository.Transaction, VisitTransaction.LockVisit, VisitTransaction.SaveVisit, VisitTransaction.AppendAudit, VisitTransaction.EnqueueNotifications | proposed |
| POST /v1/staff/bookings/{id}/reschedule | visit_handler.go:Reschedule | VisitUsecase.Reschedule | Reschedule -> StaffBooking | VisitSerializer.ToStaffBooking | VisitRepository.Transaction, VisitTransaction.LockVisit, VisitTransaction.LockResources, VisitTransaction.CheckOverlap, VisitTransaction.SaveVisit, VisitTransaction.RevokeAdmission, VisitTransaction.AppendAudit, VisitTransaction.EnqueueNotifications | proposed |
| POST /v1/staff/bookings/{id}/update | visit_handler.go:UpdateRestricted | VisitUsecase.UpdateRestricted | RestrictedUpdate -> RestrictedFull | VisitSerializer.ToRestrictedFull | VisitRepository.Transaction, VisitTransaction.LockVisit, VisitTransaction.LockResources, VisitTransaction.CheckOverlap, VisitTransaction.SaveVisit, VisitTransaction.AppendAudit | proposed |
| POST /v1/staff/bookings/{id}/cancel | visit_handler.go:StaffCancel | VisitUsecase.CancelStaffBooking | StaffCancel -> StaffBooking | VisitSerializer.ToStaffBooking | VisitRepository.Transaction, VisitTransaction.LockVisit, VisitTransaction.SaveVisit, VisitTransaction.RevokeAdmission, VisitTransaction.AppendAudit, VisitTransaction.EnqueueNotifications | proposed |
| POST /v1/staff/bookings/{id}/checkin | reception_handler.go:Checkin | ReceptionUsecase.Checkin | Attendance -> ArrivalBooking | ReceptionSerializer.ToArrivalBooking | VisitRepository.Transaction, VisitTransaction.LockVisit, VisitTransaction.ConfirmAttendance, VisitTransaction.AppendAudit | proposed |
| POST /v1/staff/bookings/{id}/checkout | reception_handler.go:Checkout | ReceptionUsecase.Checkout | Attendance -> ArrivalBooking | ReceptionSerializer.ToArrivalBooking | VisitRepository.Transaction, VisitTransaction.LockVisit, VisitTransaction.ConfirmAttendance, VisitTransaction.AppendAudit | proposed |
| POST /v1/staff/resolve | reception_handler.go:Resolve | ReceptionUsecase.Resolve | Resolve -> ArrivalBooking | ReceptionSerializer.ToArrivalBooking | ReceptionRepository.ResolveCredential | proposed |
| GET /v1/staff/employees | management_handler.go:Employees | ManagementUsecase.GetEmployees | EmployeeQuery -> EmployeePage | ManagementSerializer.ToEmployeePage | ManagementRepository.GetEmployees | proposed |
| GET /v1/staff/employees/{id} | management_handler.go:Employee | ManagementUsecase.GetEmployee | none -> Employee | ManagementSerializer.ToEmployee | ManagementRepository.GetEmployee | proposed |
| GET /v1/staff/integrations | management_handler.go:Integrations | ManagementUsecase.GetIntegrations | none -> []Integration | ManagementSerializer.ToIntegration | ManagementRepository.GetIntegrations | proposed |
| PATCH /v1/staff/integrations/{id} | management_handler.go:UpdateIntegration | ManagementUsecase.UpdateIntegration | IntegrationPatch -> Integration | ManagementSerializer.ToIntegration | ManagementRepository.Transaction, ManagementTransaction.UpdateIntegration, ManagementTransaction.AppendAudit | proposed |
| GET /v1/staff/reports/summary | report_handler.go:Summary | ReportUsecase.GetSummary | ReportQuery -> Summary | ReportSerializer.ToSummary | ReportRepository.GetSummary | proposed |
| GET /v1/staff/reports/export | report_handler.go:Export | ReportUsecase.Export | ReportExportQuery -> Export | ReportSerializer.ToExport | ReportRepository.GetSummary | proposed |

Implementations live at internal/usecase/{module}/implementation, internal/repository/{module}/implementation and internal/serializer/{module}/implementation; DTOs stay in pkg/dto and domain aggregates in internal/entity. Public catalog and staff catalog intentionally reuse the same wire shape. Phase 1 AuthUsecase.Login returns dto.AuthResult; AuthUsecase.GetCurrentUser returns dto.CurrentUser. AccessProof contains the verified employee subject and token issue/expiry/issuer/audience context, never a session ID or role grant. CurrentAuthority contains account active/name, exactly one role and current PA assignments; serializers derive the fixed permission list. Existing SessionUsecase/GetSession signatures above describe the superseded binary only, not required Phase 1 work.

