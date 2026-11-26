import {
  ApiError,
  bookingName,
  isRestricted,
  partyLabel,
  preview,
  repository,
  today,
  useVisitCommand,
  useBookings,
  useSession,
  visitTiming,
  type StaffBooking,
} from "@vms/data";
import {
  Button,
  Empty,
  ErrorNotice,
  Loading,
  Modal,
  QueryState,
  RequiredFieldsHint,
  RequiredMark,
  VisitIdentity,
  VisitKind,
} from "@vms/ui";
import { InputOTP } from "@vms/ui/input-otp";
import { ArrowRight, Check, KeyRound, ScanLine, ShieldCheck } from "lucide-react";
import { lazy, Suspense, useEffect, useRef, useState } from "react";
const Scanner = lazy(() => import("../features/Scanner"));
const VisitFileViewer = lazy(() => import("../features/VisitFileViewer"));
export default function Reception({ mode }: { mode: "checkin" | "checkout" }) {
  const [page, setPage] = useState(1);
  const q = useBookings({
    filter: {
      view: "reception",
      mode,
      ...(mode === "checkin" ? { date: today() } : {}),
      page,
      pageSize: 25,
    },
  });
  const { role } = useSession();
  const action = useVisitCommand();
  const [method, setMethod] = useState<"code" | "qr" | null>(null);
  const [resolved, setResolved] = useState<StaffBooking | null>(null);
  const [credential, setCredential] = useState("");
  const attempt = useRef(0);
  const resolvedAt = useRef(0);
  const [checked, setChecked] = useState(false);
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const dialogTrigger = useRef<HTMLElement | null>(null);
  const incoming = mode === "checkin";
  const current = resolved && q.data?.find((booking) => booking.id === resolved.id);
  const attendance =
    resolved &&
    visitTiming(
      current && (current.timing?.serverTime || 0) >= (resolved.timing?.serverTime || 0)
        ? current
        : resolved,
    );
  const permitted = !!attendance && (incoming ? attendance.canCheckIn : attendance.canCheckOut);
  useEffect(() => {
    const denied = q.error instanceof ApiError && [401, 403].includes(q.error.status);
    if (
      resolved &&
      (denied ||
        (q.data &&
          q.dataUpdatedAt > resolvedAt.current &&
          ((preview && !current) || (current && current.version !== resolved.version))))
    ) {
      setResolved(null);
      setCredential("");
      setChecked(false);
      setError(
        new Error("Kunjungan diperbarui. Periksa kode atau QR kembali sebelum melanjutkan."),
      );
    }
  }, [current, q.data, q.dataUpdatedAt, q.error, resolved]);
  useEffect(
    () => () => {
      attempt.current++;
    },
    [],
  );
  const list = (q.data || []).filter((b) =>
    incoming
      ? b.date === today() && visitTiming(b).displayStatus === "approved"
      : b.status === "checked_in",
  );
  async function resolve(code: string) {
    const requestId = ++attempt.current;
    setBusy(true);
    setError(null);
    setResolved(null);
    setCredential("");
    setChecked(false);
    try {
      if (!role) throw new ApiError(401, "Sesi berakhir. Silakan masuk kembali.");
      const b = await repository.resolve(code, role);
      if (requestId !== attempt.current) return;
      if (b.status !== (incoming ? "approved" : "checked_in"))
        throw new Error(
          incoming
            ? "Kunjungan belum disetujui atau sudah diproses."
            : "Pengunjung belum check in atau sudah check out.",
        );
      setResolved(b);
      resolvedAt.current = Date.now();
      setCredential(code);
      setChecked(false);
    } catch (err) {
      if (requestId === attempt.current) setError(err);
    } finally {
      if (requestId === attempt.current) setBusy(false);
    }
  }
  function close() {
    if (action.isPending) return;
    attempt.current++;
    setBusy(false);
    setMethod(null);
    setResolved(null);
    setCredential("");
    setError(null);
    setChecked(false);
  }
  async function confirmAttendance(booking: StaffBooking) {
    if (action.isPending || !checked || !permitted) return;
    setError(null);
    try {
      await action.mutateAsync({
        booking,
        command: { type: "attend", direction: mode === "checkin" ? "in" : "out", credential },
      });
      setNotice(`${bookingName(booking)} berhasil ${incoming ? "check in" : "check out"}.`);
      close();
    } catch (err) {
      if (err instanceof ApiError && [401, 403, 404, 409].includes(err.status)) {
        setResolved(null);
        setCredential("");
        setChecked(false);
      }
      setError(err);
    }
  }
  return (
    <>
      <div className="page-intro">
        <div>
          <h1>{incoming ? "Check in pengunjung" : "Check out pengunjung"}</h1>
          <p>
            {incoming
              ? "Sambut pengunjung. Validasi identitas sebelum mengizinkan masuk."
              : "Pastikan identitas dan kunjungan sebelum menyelesaikan kunjungan."}
          </p>
        </div>
        <span className="date-chip">
          <ShieldCheck size={16} />
          Verifikasi oleh resepsionis
        </span>
      </div>
      {notice && (
        <div role="status" className="hint">
          <Check size={18} />
          {notice}
        </div>
      )}
      <section className="reception-entry panel">
        <div className="reception-heading">
          <h2>Mulai dengan kode atau QR</h2>
          <p>Pilih metode identifikasi pengunjung.</p>
        </div>
        <div className="method-options">
          <button
            onClick={(event) => {
              dialogTrigger.current = event.currentTarget;
              setMethod("code");
            }}
          >
            <KeyRound size={28} strokeWidth={1.6} />
            <span>
              <strong>Kode kunjungan</strong>
              <small>Masukkan kode yang dibawa pengunjung</small>
            </span>
            <ArrowRight size={20} />
          </button>
          <button
            onClick={(event) => {
              dialogTrigger.current = event.currentTarget;
              setMethod("qr");
            }}
          >
            <ScanLine size={28} strokeWidth={1.6} />
            <span>
              <strong>Scan QR</strong>
              <small>Pindai QR kunjungan dengan kamera</small>
            </span>
            <ArrowRight size={20} />
          </button>
        </div>
      </section>
      <section className="panel section-gap">
        <div className="panel-heading">
          <div>
            <h2>{incoming ? "Kedatangan yang dijadwalkan" : "Pengunjung di dalam kawasan"}</h2>
            <p>
              {incoming
                ? "Ringkasan kedatangan hari ini dalam akses Anda."
                : "Semua pengunjung yang belum check out, termasuk kunjungan sebelumnya."}
            </p>
          </div>
          <span className="count">
            {q.data === undefined ? "…" : preview ? list.length : q.total}
          </span>
        </div>
        <QueryState
          pending={q.isPending}
          error={q.error}
          retry={q.refetch}
          hasData={q.data !== undefined}
        >
          {list.length ? (
            <div
              className="table-wrap"
              tabIndex={0}
              role="region"
              aria-label="Daftar kehadiran kunjungan"
            >
              <table>
                <thead>
                  <tr>
                    <th>Nama pengunjung</th>
                    <th>Detail kunjungan</th>
                  </tr>
                </thead>
                <tbody>
                  {list.map((b) => (
                    <tr key={b.id}>
                      <td>
                        <strong>
                          {bookingName(b)} <VisitKind booking={b} />
                        </strong>
                        <small>
                          {isRestricted(b)
                            ? b.reference
                            : `${b.organization || "Instansi belum diisi"} · ${partyLabel(b)}`}
                        </small>
                      </td>
                      <td>
                        {isRestricted(b)
                          ? "Rincian identitas melalui kode atau QR"
                          : b.host || "Belum ditentukan"}
                        <small>
                          {b.time} WITA · {b.room || "Ruangan belum ditentukan"}
                        </small>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <Empty
              title={incoming ? "Belum ada kedatangan berikutnya" : "Belum ada pengunjung di dalam"}
              text="Data akan diperbarui setelah proses kunjungan atau kedatangan."
            />
          )}
        </QueryState>
        {!preview && q.total !== undefined && (page > 1 || q.total > 25) && (
          <nav className="form-actions" aria-label="Halaman kehadiran">
            <Button kind="secondary" disabled={page <= 1} onClick={() => setPage(page - 1)}>
              Sebelumnya
            </Button>
            <span>Halaman {page}</span>
            <Button
              kind="secondary"
              disabled={page * 25 >= q.total}
              onClick={() => setPage(page + 1)}
            >
              Berikutnya
            </Button>
          </nav>
        )}
      </section>
      <Modal
        returnFocusTo={dialogTrigger.current}
        open={!!method}
        onClose={close}
        title={
          resolved
            ? "Validasi pengunjung"
            : method === "qr"
              ? "Scan QR kunjungan"
              : "Masukkan kode kunjungan"
        }
        description={
          resolved
            ? "Cocokkan identitas dan rincian kunjungan yang tersedia sebelum mencatat kehadiran."
            : "Identifikasi tidak langsung mencatat kehadiran."
        }
      >
        {resolved ? (
          <form
            onSubmit={(event) => {
              event.preventDefault();
              void confirmAttendance(resolved);
            }}
          >
            <VisitIdentity booking={resolved} collapseVisitorNames />
            {role &&
              !isRestricted(resolved) &&
              (resolved.attachmentAvailable || resolved.visitorPhotoAvailable) && (
                <Suspense fallback={<Loading />}>
                  <VisitFileViewer
                    key={`${role}:${resolved.id}`}
                    id={resolved.id}
                    role={role}
                    credential={credential}
                    attachmentName={resolved.ndName}
                    showAttachment={!!resolved.attachmentAvailable}
                    showPhoto={!!resolved.visitorPhotoAvailable}
                    visitorName={resolved.name}
                  />
                </Suspense>
              )}
            {!permitted && (
              <ErrorNotice
                error={
                  attendance?.displayStatus === "missed"
                    ? "Waktu check-in sudah berakhir. Kunjungan terlewat."
                    : incoming
                      ? "Check-in belum tersedia. Check-in dibuka 30 menit sebelum jadwal hingga kunjungan selesai."
                      : "Check-out belum tersedia untuk kunjungan ini."
                }
              />
            )}
            {(!incoming || permitted) && (
              <>
                <RequiredFieldsHint />
                <label className="check-label section-gap">
                  <input
                    type="checkbox"
                    required
                    checked={checked}
                    onChange={(e) => setChecked(e.target.checked)}
                  />
                  <span>
                    {isRestricted(resolved)
                      ? "Saya telah mencocokkan kode atau QR dan rincian kunjungan yang tersedia untuk verifikasi kehadiran."
                      : "Saya telah mencocokkan identitas pengunjung dan detail kunjungan dengan dokumen asli."}{" "}
                    <RequiredMark />
                  </span>
                </label>
              </>
            )}
            <ErrorNotice error={error} />
            <div className="form-actions">
              <Button
                type="button"
                kind="secondary"
                onClick={() => {
                  setResolved(null);
                  setCredential("");
                  setError(null);
                }}
              >
                Periksa kode lain
              </Button>
              <Button type="submit" disabled={action.isPending || !checked || !permitted}>
                {action.isPending
                  ? "Menyimpan…"
                  : `Konfirmasi ${incoming ? "check in" : "check out"}`}
              </Button>
            </div>
          </form>
        ) : (
          <>
            {method === "qr" ? (
              <Suspense fallback={<Loading />}>
                <Scanner onResult={resolve} />
              </Suspense>
            ) : (
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  void resolve(String(new FormData(e.currentTarget).get("code")));
                }}
              >
                <div className="field">
                  <RequiredFieldsHint />
                  <label htmlFor="booking-code">
                    Kode kunjungan <RequiredMark />
                  </label>
                  <InputOTP
                    autoFocus
                    id="booking-code"
                    name="code"
                    minLength={8}
                    required
                    aria-label="8 digit kode kunjungan"
                  />
                </div>
                <div className="form-actions">
                  <Button type="submit" disabled={busy}>
                    {busy ? "Memeriksa…" : "Periksa kunjungan"}
                    <ArrowRight size={16} />
                  </Button>
                </div>
              </form>
            )}
            <ErrorNotice error={error} />
          </>
        )}
      </Modal>
    </>
  );
}
