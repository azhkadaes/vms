//go:build integration

package bookingfilerepositoryimplementation_test

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"

	storage "github.com/azhkadaes/vms/backend/internal/infrastructure/storage/implementation"
	"github.com/azhkadaes/vms/backend/internal/policy"
	filerepo "github.com/azhkadaes/vms/backend/internal/repository/bookingfile/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
	fileusecase "github.com/azhkadaes/vms/backend/internal/usecase/bookingfile/implementation"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

func newFileVisit(t *testing.T, db *gorm.DB, key string, size int) (int32, string) {
	t.Helper()
	var now time.Time
	var officer int32
	var fileID, scheduleID, visitorID, visitID int32
	read := func(target any, query string, params ...any) {
		t.Helper()
		if err := db.Raw(query, params...).Scan(target).Error; err != nil {
			t.Fatal(err)
		}
	}
	read(&now, "SELECT transaction_timestamp()")
	read(&officer, "SELECT id FROM vms.pejabat WHERE status_aktif ORDER BY id LIMIT 1")
	if officer <= 0 {
		t.Fatal("missing active catalog official")
	}
	read(&fileID, `INSERT INTO vms.form_documents(nama,storage_path,mime_type,ukuran_file_bytes,purpose)
		VALUES ('synthetic.pdf',?,'application/pdf',?,'document') RETURNING id`, key, size)
	if err := db.Exec(`INSERT INTO support.form_document_uploads(id_form_documents,audience,owner_digest,reference,
		content_sha256,token_hash,key_version,created_at,expires_at,linked_at)
		VALUES (?,'public',?,?,?,?,1,transaction_timestamp()-INTERVAL '5 minutes',transaction_timestamp()+INTERVAL '55 minutes',
			transaction_timestamp()-INTERVAL '4 minutes')`, fileID, security.Hash("public"), uuid.NewString(),
		security.Hash("synthetic PDF"), security.Hash(uuid.NewString())).Error; err != nil {
		t.Fatal(err)
	}
	wita := now.In(validate.WITA)
	read(&scheduleID, `INSERT INTO vms.detail_kunjungan(id_pejabat,id_formdocuments,tanggal,waktu_mulai,visit_duration)
		VALUES (?,?,?::date,?::time,60) RETURNING id`, officer, fileID, wita.Format("2006-01-02"), wita.Format("15:04"))
	read(&visitorID, "INSERT INTO vms.detail_pengunjung DEFAULT VALUES RETURNING id")
	value, err := rand.Int(rand.Reader, big.NewInt(100_000_000))
	if err != nil {
		t.Fatal(err)
	}
	code := fmt.Sprintf("%08d", value.Int64())
	read(&visitID, `INSERT INTO vms.kunjungan(id_detail_pengunjung,id_detail_kunjungan,kode_kunjungan,tipe_kunjungan,status_penerimaan,tanggal_dibuat)
		VALUES (?,?,?,'vip','approved',transaction_timestamp()) RETURNING id`, visitorID, scheduleID, code)
	return visitID, code
}

func TestBookingFilesRecheckRealCredentialAndPrivateStorage(t *testing.T) {
	db := integrationtest.RolledBack(t, integrationtest.Open(t))
	ctx := context.Background()
	files, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("%PDF-1.4\n%%EOF")
	key, err := files.Put(ctx, content)
	if err != nil {
		t.Fatal(err)
	}
	id, code := newFileVisit(t, db, key, len(content))
	usecase := fileusecase.NewBookingFileUsecase(filerepo.NewBookingFileRepository(db), files, noop.NewTracerProvider().Tracer("test"))
	protocol := dto.Session{Active: true, Role: policy.RoleProtocol, Permissions: []string{policy.PermBookings}}
	reception := dto.Session{Active: true, Role: policy.RoleReceptionist, Permissions: []string{policy.PermCheckin, policy.PermCheckout}}
	if opened, err := usecase.OpenStaffFile(ctx, protocol, id, policy.PurposeDocument); err != nil {
		t.Fatal(err)
	} else {
		defer opened.Content.Close()
		got, err := io.ReadAll(opened.Content)
		if err != nil || string(got) != string(content) || opened.MIME != "application/pdf" || opened.Size != len(content) {
			t.Fatal("staff did not receive the linked private PDF")
		}
	}
	if opened, err := usecase.OpenReceptionFile(ctx, reception, id, policy.PurposeDocument, dto.ReceptionResolve{Credential: code}); err != nil {
		t.Fatalf("live booking code did not open the file: %v", err)
	} else {
		_ = opened.Content.Close()
	}
	checkNotFound := func(err error) {
		t.Helper()
		var f *failure.Error
		if !errors.As(err, &f) || f.Status != 404 || f.Code != "not_found" {
			t.Fatalf("want uniform not_found, got %v", err)
		}
	}
	_, err = usecase.OpenReceptionFile(ctx, reception, id+1, policy.PurposeDocument, dto.ReceptionResolve{Credential: code})
	checkNotFound(err)
	const qr = "synthetic booking file QR value"
	if err := db.Exec(`INSERT INTO vms.qr_kunjungan(id_kunjungan, token_hash, tanggal_dibuat, tanggal_expire)
		VALUES (?,?,transaction_timestamp()-INTERVAL '2 hours',transaction_timestamp()-INTERVAL '1 hour')`, id, security.Hash(qr)).Error; err != nil {
		t.Fatal(err)
	}
	request := dto.ReceptionResolve{Credential: qr}
	_, err = usecase.OpenReceptionFile(ctx, reception, id, policy.PurposeDocument, request)
	checkNotFound(err)
	if err := db.Exec(`UPDATE vms.kunjungan SET status_penerimaan='checked_in', waktu_cek_in=transaction_timestamp()-INTERVAL '90 minutes'
		WHERE id=?`, id).Error; err != nil {
		t.Fatal(err)
	}
	if opened, err := usecase.OpenReceptionFile(ctx, reception, id, policy.PurposeDocument, request); err != nil {
		t.Fatalf("the same older QR was not accepted for the checkout file read: %v", err)
	} else {
		_ = opened.Content.Close()
	}
	if err := db.Exec("UPDATE vms.qr_kunjungan SET dicabut_pada=transaction_timestamp() WHERE id_kunjungan=?", id).Error; err != nil {
		t.Fatal(err)
	}
	_, err = usecase.OpenReceptionFile(ctx, reception, id, policy.PurposeDocument, request)
	checkNotFound(err)
}
