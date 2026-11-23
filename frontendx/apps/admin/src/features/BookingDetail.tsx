import {
  useUnsavedChanges,
  ApiError,
  can,
  canReadInternalRejection,
  canReadInternalCancellation,
  bookingName,
  dateLabel,
  isRestricted,
  preview,
  today,
  useBookingAction,
  useCatalogs,
  useSession,
  validateReschedule,
  type StaffBooking,
} from "@vms/data";
import { rejectionSchema, staffCancellationSchema } from "@vms/data/validation";
import {
  Select,
  Button,
  ErrorNotice,
  Loading,
  Modal,
  RequiredFieldsHint,
  RequiredMark,
  StatusBadge,
  VisitIdentity,
} from "@vms/ui";
import { DatePicker } from "@vms/ui/date-picker";
import { TimePicker } from "@vms/ui/time-picker";
import { ReferenceDataNotice } from "@vms/ui/reference-data-notice";
import { CalendarClock, Check, X } from "lucide-react";
import { lazy, Suspense, useEffect, useRef, useState } from "react";
import VisitNotesFields from "./VisitNotesFields";
const RestrictedAppointment = lazy(() => import("./RestrictedAppointment"));
const AttachmentAccess = lazy(() => import("./AttachmentAccess"));
const VisitorPhoto = lazy(() => import("./VisitorPhoto"));

export default function BookingDetail({
  returnFocusTo,
  booking: currentBooking,
  onClose,
  onSuccess,
}: {
  returnFocusTo?: HTMLElement | null;
  booking: StaffBooking;
  onClose: () => void;
  onSuccess: (text: string) => void;
}) {
  const { session } = useSession();
  const mutation = useBookingAction();
  const catalogs = useCatalogs();
  const rooms = catalogs.data?.rooms || [];
  const [editSnapshot, setEditSnapshot] = useState<StaffBooking | null>(null);
  const b = editSnapshot || currentBooking;
  const [mode, setMode] = useState<"approve" | "reject" | "reschedule" | "cancel" | null>(null);
  const [dirty, setDirty] = useState(false);
  const [schedule, setSchedule] = useState({ date: b.date, time: b.time });
  const room = preview ? (rooms.includes(b.room) ? b.room : "") : (b.roomId ?? (b.room ? "current-room" : ""));
  const roomOptions = (preview ? rooms.map((name) => ({ id: name, name })) : (catalogs.data?.roomOptions ?? []))
    .filter(({ id }) => id !== room);
  const hasChanges =
    dirty || (mode === "reschedule" && (schedule.date !== b.date || schedule.time !== b.time));
  useUnsavedChanges(hasChanges, mutation.isPending);
  const [cancellationReason, setCancellationReason] = useState("");
  const [cancellationError, setCancellationError] = useState<string>();
  const cancellationInput = useRef<HTMLTextAreaElement>(null);
  const [cancellationNote, setCancellationNote] = useState("");
  const [cancellationNoteError, setCancellationNoteError] = useState<string>();
  const cancellationNoteInput = useRef<HTMLTextAreaElement>(null);
  const [error, setError] = useState<unknown>();
  const [editing, setEditing] = useState(false);
  const restricted = isRestricted(b);
  const roomRequired = preview && !restricted;
  const showContactAndCode = session?.role === "PA" || session?.role === "Protocol Officer";
  const protocolReadOnly = session?.role === "Protocol Officer" && !restricted && !b.vip;
  const [rejection, setRejection] = useState({
    rejectionReason: "",
    internalRejectionNote: "",
  });
  type RejectionField = keyof typeof rejection;
  const [rejectionErrors, setRejectionErrors] = useState<Partial<Record<RejectionField, string>>>(
    {},
  );
  const publicReason = useRef<HTMLTextAreaElement>(null);
  const internalNote = useRef<HTMLTextAreaElement>(null);
  useEffect(() => {
    if (mode === "reject") publicReason.current?.focus();
    if (mode === "cancel") cancellationInput.current?.focus();
    if (mode === "approve") document.getElementById("room")?.focus();
    if (mode === "reschedule") document.getElementById("new-date")?.focus();
  }, [mode]);
  function validateRejectionField(key: RejectionField, value: string) {
    const result = rejectionSchema.shape[key].safeParse(value);
    setRejectionErrors((previous) => ({
      ...previous,
      [key]: result.success ? undefined : result.error.issues[0].message,
    }));
  }
  function closeEditor() {
    setEditing(false);
    setEditSnapshot(null);
  }
  if (editing && session && can(session, currentBooking, "update"))
    return (
      <Suspense
        fallback={
          <Modal open title="Edit kunjungan terbatas" onClose={closeEditor}>
            <Loading />
          </Modal>
        }
      >
        <RestrictedAppointment
          booking={b}
          returnFocusTo={returnFocusTo}
          onClose={closeEditor}
          onSaved={() => {
            onSuccess("Detail kunjungan terbatas diperbarui.");
            onClose();
          }}
        />
      </Suspense>
    );
  return (
    <Modal
      returnFocusTo={returnFocusTo}
      open
      onClose={() => {
        if (!mutation.isPending) onClose();
      }}
      title={
        mode === "cancel"
          ? "Batalkan kunjungan"
          : mode === "reschedule"
            ? "Edit kunjungan"
            : mode === "reject"
              ? "Tolak permohonan"
              : "Detail kunjungan"
      }
      description={
        mode === "cancel"
          ? "Pembatalan oleh petugas dicatat dalam audit dan menutup izin masuk serta reservasi. Tindakan ini juga dapat dilakukan sesudah waktu mulai jika Anda berwenang."
          : mode === "reschedule"
            ? "Perbarui tanggal, waktu, ruangan, durasi, dan catatan. Kunjungan tetap disetujui; catatan internal hanya terlihat oleh petugas berwenang."
            : mode === "reject"
              ? "Pisahkan pesan untuk pengaju dan catatan internal staf."
              : "Identitas, jadwal, dan riwayat kunjungan."
      }
    >
      {mode === "cancel" ? (
        <div className="section-gap">
          <h3>{bookingName(b)}</h3>
          <p>
            {dateLabel(b.date)} · {b.time} WITA
          </p>
          {b.room && <p>{b.room}</p>}
        </div>
      ) : (
        <>
          <VisitIdentity
            booking={b}
            collapseVisitorNames
            showDocument
            showBookingCode={showContactAndCode}
            showVisitorContact={showContactAndCode}
          />
          {b.visitorInstructions && (
            <section className="section-gap">
              <h3>Instruksi pengunjung</h3>
              <p className="visit-note-text">{b.visitorInstructions}</p>
            </section>
          )}
          {b.coordinationNote &&
            session &&
            session.permissions.includes("bookings") &&
            (["PA", "Protocol Officer"].includes(session.role) ||
              (session.role === "Super Admin" && restricted)) &&
            (!restricted || b.restrictedView === "full") && (
              <section className="section-gap">
                <h3>Catatan koordinasi internal</h3>
                <p className="visit-note-text">{b.coordinationNote}</p>
              </section>
            )}
        </>
      )}
      {mode !== "cancel" && !preview && b.attachmentAvailable && b.ndName && !restricted && (
        <Suspense fallback={<Loading />}>
          <AttachmentAccess id={b.id} name={b.ndName} />
        </Suspense>
      )}
      {editSnapshot && currentBooking.version !== editSnapshot.version && (
        <p role="status" className="hint section-gap">
          Kunjungan ini diperbarui oleh petugas lain. Isian Anda tetap disimpan; kembali ke detail
          untuk meninjau data terbaru sebelum melanjutkan.
        </p>
      )}
      {b.createdBy && (
        <p className="muted section-gap">
          Dibuat oleh {b.createdBy.name} ·{" "}
          {b.createdBy.role === "Protocol Officer" ? "Protokol" : b.createdBy.role}
        </p>
      )}
      <div className="section-gap">
        <StatusBadge status={b.status} timing={b.timing} rescheduled={b.rescheduled} />
      </div>
      {protocolReadOnly && (
        <p className="hint section-gap" role="note">
          Hanya lihat. Persetujuan dan perubahan kunjungan umum ini dikelola oleh PA pejabat tujuan.
          Protokol mengelola kunjungan VIP dan terbatas.
        </p>
      )}
      {!restricted && b.visitorPhotoAvailable && session && (
        <Suspense fallback={<Loading />}>
          <VisitorPhoto
            key={`${b.id}:${session.role}`}
            id={b.id}
            role={session.role}
            name={b.name}
          />
        </Suspense>
      )}
      {mode ? (
        <form
          className="section-gap"
          onChange={(event) => {
            if (mode !== "reschedule") {
              setDirty(true);
              return;
            }
            const fields = new FormData(event.currentTarget);
            setDirty(
              fields.get("room") !== room ||
                fields.get("duration") !== String(b.duration ?? "") ||
                String(fields.get("visitorInstructions")).trim() !==
                  (b.visitorInstructions || "").trim() ||
                String(fields.get("coordinationNote")).trim() !== (b.coordinationNote || "").trim(),
            );
          }}
          noValidate={mode === "reject"}
          onSubmit={async (e) => {
            e.preventDefault();
            if (mutation.isPending || (mode === "reschedule" && !hasChanges)) return;
            setError(null);
            const f = Object.fromEntries(new FormData(e.currentTarget));
            const { room: selectedRoom, ...fields } = f;
            let values: Record<string, unknown> = {
              ...fields,
              duration: Number(f.duration || 60),
            };
            if (preview) values.room = selectedRoom;
            else if ((mode === "approve" || mode === "reschedule") && selectedRoom !== room)
              values.roomId = selectedRoom || null;
            if (mode === "reject") {
              const parsed = rejectionSchema.safeParse(rejection);
              if (!parsed.success) {
                const errors = parsed.error.flatten().fieldErrors;
                setRejectionErrors({
                  rejectionReason: errors.rejectionReason?.[0],
                  internalRejectionNote: errors.internalRejectionNote?.[0],
                });
                requestAnimationFrame(() => {
                  (errors.rejectionReason ? publicReason : internalNote).current?.focus();
                });
                return;
              }
              setRejectionErrors({});
              values = parsed.data;
            }
            if (mode === "cancel") {
              const parsed = staffCancellationSchema.safeParse({
                reason: cancellationReason,
                internalCancellationNote: cancellationNote,
              });
              if (!parsed.success) {
                const errors = parsed.error.flatten().fieldErrors;
                setCancellationError(errors.reason?.[0]);
                setCancellationNoteError(errors.internalCancellationNote?.[0]);
                requestAnimationFrame(() =>
                  (errors.reason ? cancellationInput : cancellationNoteInput).current?.focus(),
                );
                return;
              }
              setCancellationError(undefined);
              setCancellationNoteError(undefined);
              values = parsed.data;
            }
            if (mode === "reschedule") {
              const msg = validateReschedule(
                String(f.date),
                String(f.time),
                Number(values.duration),
                String(f.room),
                String(f.reason),
                restricted,
                preview ? rooms : null,
              );
              if (msg) {
                setError(msg);
                return;
              }
            }
            try {
              await mutation.mutateAsync({ booking: b, action: mode, values });
              onSuccess(
                mode === "cancel"
                  ? "Kunjungan dibatalkan oleh petugas."
                  : mode === "reschedule"
                    ? "Perubahan kunjungan disimpan."
                    : mode === "approve"
                      ? "Kunjungan disetujui."
                      : "Janji temu ditolak.",
              );
              onClose();
            } catch (err) {
              setError(err);
              if (mode === "cancel" && err instanceof ApiError && err.fieldErrors) {
                setCancellationError(err.fieldErrors.reason?.[0]);
                setCancellationNoteError(err.fieldErrors.internalCancellationNote?.[0]);
                (err.fieldErrors.reason
                  ? cancellationInput
                  : cancellationNoteInput
                ).current?.focus();
              }
              if (mode === "reject" && err instanceof ApiError && err.fieldErrors) {
                setRejectionErrors({
                  rejectionReason: err.fieldErrors.rejectionReason?.[0],
                  internalRejectionNote: err.fieldErrors.internalRejectionNote?.[0],
                });
                const firstField = err.fieldErrors.rejectionReason ? publicReason : internalNote;
                requestAnimationFrame(() => firstField.current?.focus());
              }
            }
          }}
        >
          <RequiredFieldsHint />
          {(mode === "approve" || mode === "reschedule") && (
            <ReferenceDataNotice
              loading={catalogs.isPending}
              error={catalogs.error}
              onRetry={() => void catalogs.refetch()}
            />
          )}
          <div className="form-grid">
            {mode === "reschedule" && (
              <>
                <DatePicker
                  id="new-date"
                  name="date"
                  label="Tanggal kunjungan"
                  min={today()}
                  value={schedule.date}
                  onChange={(date) => setSchedule((current) => ({ ...current, date }))}
                  required
                />
                <TimePicker
                  min="08:00"
                  max="16:00"
                  id="new-time"
                  name="time"
                  label="Waktu mulai · WITA"
                  value={schedule.time}
                  onChange={(time) => setSchedule((current) => ({ ...current, time }))}
                  required
                />
              </>
            )}
            {(mode === "approve" || mode === "reschedule") && (
              <>
                <div className="field">
                  <label htmlFor="room">
                    Ruangan{" "}
                    {roomRequired ? <RequiredMark /> : <span className="muted">(opsional)</span>}
                  </label>
                  <Select
                    name="room"
                    id="room"
                    defaultValue={room}
                    required={roomRequired}
                  >
                    <option value="">{roomRequired ? "Pilih ruangan" : "Tanpa ruangan"}</option>
                    {room && <option value={room}>{b.room}</option>}
                    {roomOptions.map(({ id, name }) => (
                      <option key={id} value={id}>{name}</option>
                    ))}
                  </Select>
                </div>
                <div className="field">
                  <label htmlFor="duration">
                    Durasi · menit{" "}
                    {restricted ? <span className="muted">(opsional)</span> : <RequiredMark />}
                  </label>
                  <input
                    id="duration"
                    name="duration"
                    type="number"
                    min={15}
                    max={240}
                    defaultValue={b.duration}
                    required={!restricted}
                  />
                </div>
                <VisitNotesFields
                  visitor={{ defaultValue: b.visitorInstructions || "" }}
                  coordination={{ defaultValue: b.coordinationNote || "" }}
                />
              </>
            )}
            {mode === "reject" && (
              <>
                <div className="field span-2">
                  <label htmlFor="rejection-reason">
                    Pesan penolakan untuk pengaju <RequiredMark />
                  </label>
                  <textarea
                    id="rejection-reason"
                    name="rejectionReason"
                    ref={publicReason}
                    required
                    minLength={5}
                    maxLength={1000}
                    rows={3}
                    placeholder="Jelaskan alasan penolakan dan arahan yang dapat dilakukan pengaju."
                    value={rejection.rejectionReason}
                    onChange={(event) => {
                      setRejection((current) => ({
                        ...current,
                        rejectionReason: event.target.value,
                      }));
                      if (rejectionErrors.rejectionReason)
                        validateRejectionField("rejectionReason", event.target.value);
                    }}
                    onBlur={(event) =>
                      validateRejectionField("rejectionReason", event.target.value)
                    }
                    aria-invalid={!!rejectionErrors.rejectionReason}
                    aria-describedby={`rejection-reason-hint${rejectionErrors.rejectionReason ? " rejection-reason-error" : ""}`}
                  />
                  <small id="rejection-reason-hint">
                    Ditampilkan kepada pengaju sebagai alasan penolakan. Maksimal 1000 karakter.
                  </small>
                  {rejectionErrors.rejectionReason && (
                    <span className="field-error" id="rejection-reason-error" role="alert">
                      {rejectionErrors.rejectionReason}
                    </span>
                  )}
                </div>
                <div className="field span-2">
                  <label htmlFor="rejection-internal-note">
                    Catatan internal <span className="muted">(opsional)</span>
                  </label>
                  <textarea
                    id="rejection-internal-note"
                    name="internalRejectionNote"
                    ref={internalNote}
                    maxLength={2000}
                    rows={3}
                    placeholder="Tambahkan konteks untuk tindak lanjut internal."
                    value={rejection.internalRejectionNote}
                    onChange={(event) => {
                      setRejection((current) => ({
                        ...current,
                        internalRejectionNote: event.target.value,
                      }));
                      if (rejectionErrors.internalRejectionNote)
                        validateRejectionField("internalRejectionNote", event.target.value);
                    }}
                    onBlur={(event) =>
                      validateRejectionField("internalRejectionNote", event.target.value)
                    }
                    aria-invalid={!!rejectionErrors.internalRejectionNote}
                    aria-describedby={`rejection-internal-hint${rejectionErrors.internalRejectionNote ? " rejection-internal-error" : ""}`}
                  />
                  <small id="rejection-internal-hint">
                    Hanya untuk PA/Protokol yang berwenang. Tidak ditampilkan kepada pengaju.
                    Maksimal 2000 karakter.
                  </small>
                  {rejectionErrors.internalRejectionNote && (
                    <span className="field-error" id="rejection-internal-error" role="alert">
                      {rejectionErrors.internalRejectionNote}
                    </span>
                  )}
                </div>
              </>
            )}
            {mode === "reschedule" && (
              <div className="field span-2">
                <label htmlFor="reason">
                  Alasan perubahan{" "}
                  {restricted ? <span className="muted">(opsional)</span> : <RequiredMark />}
                </label>
                <textarea
                  id="reason"
                  name="reason"
                  minLength={5}
                  maxLength={1000}
                  required={!restricted}
                  rows={3}
                />
              </div>
            )}
          </div>
          {mode === "cancel" && (
            <>
              <div className="field span-2">
                <label htmlFor="staff-cancellation-reason">
                  Alasan pembatalan <RequiredMark />
                </label>
                <textarea
                  ref={cancellationInput}
                  id="staff-cancellation-reason"
                  name="reason"
                  required
                  minLength={5}
                  maxLength={1000}
                  rows={4}
                  value={cancellationReason}
                  disabled={mutation.isPending}
                  onChange={(event) => {
                    setCancellationReason(event.target.value);
                    setCancellationError(undefined);
                  }}
                  aria-invalid={!!cancellationError}
                  aria-describedby={`staff-cancellation-hint${cancellationError ? " staff-cancellation-error" : ""}`}
                />
                <small id="staff-cancellation-hint">
                  Alasan ini dikirim dan ditampilkan kepada pengunjung. Minimal 5, maksimal 1000
                  karakter.
                </small>
                {cancellationError && (
                  <span className="field-error" role="alert" id="staff-cancellation-error">
                    {cancellationError}
                  </span>
                )}
              </div>
              <div className="field span-2 section-gap">
                <label htmlFor="staff-cancellation-internal-note">
                  Catatan internal <span className="muted">(opsional)</span>
                </label>
                <textarea
                  ref={cancellationNoteInput}
                  id="staff-cancellation-internal-note"
                  name="internalCancellationNote"
                  rows={3}
                  maxLength={2000}
                  value={cancellationNote}
                  disabled={mutation.isPending}
                  onChange={(event) => {
                    setCancellationNote(event.target.value);
                    setCancellationNoteError(undefined);
                  }}
                  aria-invalid={!!cancellationNoteError}
                  aria-describedby={`staff-cancellation-internal-hint${cancellationNoteError ? " staff-cancellation-internal-error" : ""}`}
                />
                <small id="staff-cancellation-internal-hint">
                  Hanya untuk staf yang berwenang. Tidak dikirim atau ditampilkan kepada pengunjung.
                  Maksimal 2000 karakter.
                </small>
                {cancellationNoteError && (
                  <span className="field-error" role="alert" id="staff-cancellation-internal-error">
                    {cancellationNoteError}
                  </span>
                )}
              </div>
            </>
          )}
          <ErrorNotice error={error} />
          <div className="form-actions">
            <Button
              kind="ghost"
              type="button"
              onClick={() => {
                setDirty(false);
                setMode(null);
                setEditSnapshot(null);
              }}
              disabled={mutation.isPending}
            >
              Kembali
            </Button>
            {(mode !== "reschedule" || hasChanges) && (
              <Button
                type="submit"
                kind={mode === "reject" || mode === "cancel" ? "danger" : "primary"}
                disabled={mutation.isPending}
              >
                {mutation.isPending
                  ? "Menyimpan…"
                  : mode === "cancel"
                    ? "Batalkan kunjungan"
                    : mode === "reschedule"
                      ? "Simpan perubahan"
                      : mode === "reject"
                        ? "Tolak permohonan"
                        : "Setujui kunjungan"}
              </Button>
            )}
          </div>
        </form>
      ) : (
        <>
          {b.status === "cancelled" && (
            <section className="decision-note section-gap" aria-label="Alasan pembatalan">
              <h3>Alasan pembatalan</h3>
              <p>{b.cancellationReason || "Alasan pembatalan belum tersedia."}</p>
            </section>
          )}
          {b.status === "cancelled" && session && canReadInternalCancellation(session, b) && (
            <section
              className="decision-note internal section-gap"
              aria-label="Catatan internal pembatalan"
            >
              <h3>
                Catatan internal <span className="muted">· khusus staf</span>
              </h3>
              <p>{b.internalCancellationNote || "Tidak ada catatan internal."}</p>
            </section>
          )}
          {b.status === "rejected" && b.rejectionReason && (
            <section
              className="decision-note section-gap"
              aria-label="Pesan penolakan untuk pengaju"
            >
              <h3>Pesan penolakan untuk pengaju</h3>
              <p>{b.rejectionReason}</p>
            </section>
          )}
          {b.status === "rejected" && session && canReadInternalRejection(session, b) && (
            <section className="decision-note internal section-gap" aria-label="Catatan internal">
              <h3>
                Catatan internal <span className="muted">· khusus staf</span>
              </h3>
              <p>{b.internalRejectionNote || "Tidak ada catatan internal."}</p>
            </section>
          )}
          <ul className="timeline">
            {(b.history || []).map((h, i) => (
              <li key={i}>
                {h.message}
                <small>
                  {new Date(h.at).toLocaleString("id-ID", {
                    timeZone: "Asia/Makassar",
                  })}{" "}
                  WITA
                </small>
              </li>
            ))}
          </ul>
          <div className="form-actions">
            {session && can(session, b, "update") && (
              <Button
                kind="secondary"
                onClick={() => {
                  setEditSnapshot(currentBooking);
                  setEditing(true);
                }}
              >
                {b.status === "approved" ? "Edit informasi pengunjung" : "Edit detail"}
              </Button>
            )}
            {(["cancel", "reject", "reschedule", "approve"] as const)
              .filter((a) => a !== "cancel" || b.status !== "pending")
              .filter((a) => !b.createdBy || (a !== "approve" && a !== "reject"))
              .filter((a) => session && can(session, b, a))
              .map((a) => (
                <Button
                  key={a}
                  kind={
                    a === "approve"
                      ? "primary"
                      : a === "reject" || a === "cancel"
                        ? "danger-outline"
                        : "secondary"
                  }
                  onClick={() => {
                    setError(null);
                    setRejectionErrors({});
                    setEditSnapshot(currentBooking);
                    setDirty(false);
                    setSchedule({ date: currentBooking.date, time: currentBooking.time });
                    setMode(a);
                  }}
                >
                  {a === "approve" ? (
                    <Check size={16} />
                  ) : a === "reject" || a === "cancel" ? (
                    <X size={16} />
                  ) : (
                    <CalendarClock size={16} />
                  )}
                  {
                    {
                      approve: "Setujui",
                      reject: "Tolak",
                      reschedule: "Edit kunjungan",
                      cancel: "Batalkan kunjungan",
                    }[a]
                  }
                </Button>
              ))}
          </div>
        </>
      )}
    </Modal>
  );
}
