import { zodResolver } from "@hookform/resolvers/zod";
import {
  useUnsavedChanges,
  ApiError,
  canCreateAppointment,
  preview,
  today,
  useBookingAction,
  useCatalogs,
  useCreateAppointment,
  useSession,
  type StaffBooking,
} from "@vms/data";
import { restrictedAppointmentSchema, type RestrictedAppointmentDraft } from "@vms/data/validation";
import { FieldMessage, Select, Button, ErrorNotice, FormSection, Modal, RequiredFieldsHint } from "@vms/ui";
import { DatePicker } from "@vms/ui/date-picker";
import { ReferenceDataNotice } from "@vms/ui/reference-data-notice";
import { TimePicker } from "@vms/ui/time-picker";
import { useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import type { z } from "zod";
import VisitNotesFields from "./VisitNotesFields";
import { AppointmentRoomField } from "./AppointmentRoomField";

type Values = z.output<typeof restrictedAppointmentSchema>;
export default function RestrictedAppointment({
  onClose,
  onSaved,
  returnFocusTo,
  booking,
}: {
  onClose: () => void;
  onSaved: (booking: StaffBooking) => void;
  returnFocusTo?: HTMLElement | null;
  booking?: StaffBooking;
}) {
  const { session } = useSession();
  const catalogs = useCatalogs();
  const officers = catalogs.data?.officers || [];
  const rooms = preview ? (catalogs.data?.rooms ?? []) : (catalogs.data?.roomOptions ?? []);
  const create = useCreateAppointment();
  const update = useBookingAction();
  const [error, setError] = useState<unknown>();
  const [discard, setDiscard] = useState(false);
  const attempt = useRef<{ payload: string; key: string } | null>(null);
  const lock = useRef(false);
  const resume = useRef<HTMLButtonElement>(null);
  const busy = create.isPending || update.isPending;
  const f = useForm<RestrictedAppointmentDraft, unknown, Values>({
    resolver: zodResolver(restrictedAppointmentSchema),
    mode: "onBlur",
    reValidateMode: "onChange",
    defaultValues: {
      visitKind: "restricted",
      date: booking?.date || "",
      time: booking?.time || "",
      officerId: booking?.officerId || "",
      name: booking?.name || "",
      organization: booking?.organization || "",
      serviceId: booking?.serviceId || "",
      visitorInstructions: booking?.visitorInstructions || "",
      coordinationNote: booking?.coordinationNote || "",
    },
  });
  const dirty = f.formState.isDirty;
  useUnsavedChanges(dirty, busy);
  function close() {
    if (lock.current) return;
    if (dirty) {
      setDiscard(true);
      requestAnimationFrame(() => resume.current?.focus());
    } else onClose();
  }
  async function submit(values: Values) {
    if (lock.current || (booking && !dirty)) return;
    if (
      !session ||
      (!preview && !values.officerId) ||
      !canCreateAppointment(session, values.officerId || "", "restricted", officers)
    ) {
      f.setError(
        "officerId",
        { message: preview ? "Pilih pejabat aktif atau biarkan kosong." : "Pilih pejabat yang aktif." },
        { shouldFocus: true },
      );
      return;
    }
    lock.current = true;
    setError(null);
    const { room, ...details } = values;
    const input = booking
      ? {
          ...details,
          name: values.name || "",
          organization: values.organization || "",
          officerId: values.officerId || "",
          serviceId: values.serviceId || "",
        }
      : { ...details, ...(preview ? { room } : { roomId: room || undefined }) };
    const payload = JSON.stringify(input);
    if (attempt.current?.payload !== payload)
      attempt.current = { payload, key: crypto.randomUUID() };
    try {
      const saved = booking
        ? await update.mutateAsync({ booking, action: "update", values: input })
        : await create.mutateAsync({ input, idempotencyKey: attempt.current.key });
      onSaved(saved);
    } catch (err) {
      setError(err);
      if (err instanceof ApiError && err.fieldErrors) {
        const fieldErrors: Record<string, string[] | undefined> = {
          ...err.fieldErrors,
          room: err.fieldErrors.roomId ?? err.fieldErrors.room,
        };
        const fields = ["date", "time", "name", "organization", "serviceId", "officerId", "room"] as const;
        for (const key of fields)
          if (fieldErrors[key]?.[0]) f.setError(key, { message: fieldErrors[key][0] });
        const first = fields.find((key) => fieldErrors[key]?.[0]);
        if (first) requestAnimationFrame(() => f.setFocus(first));
      }
    } finally {
      lock.current = false;
    }
  }
  function field(key: "name" | "organization" | "serviceId", label: string, limit: number) {
    const issue = f.formState.errors[key];
    return (
      <div className="field">
        <label htmlFor={`restricted-${key}`}>
          {label} <span className="muted">(opsional)</span>
        </label>
        <input
          {...f.register(key)}
          id={`restricted-${key}`}
          type="text"
          maxLength={limit}
          autoComplete={key === "name" ? "name" : key === "organization" ? "organization" : "off"}
          aria-invalid={!!issue}
          aria-describedby={issue ? `restricted-${key}-error` : undefined}
        />
        {issue && (
          <FieldMessage tone="error" id={`restricted-${key}-error`}>
            {issue.message}
          </FieldMessage>
        )}
      </div>
    );
  }
  return (
    <Modal
      open
      title={booking ? "Edit kunjungan terbatas" : "Buat kunjungan terbatas"}
      description={
        preview
          ? "Tanggal dan waktu wajib diisi. Rincian lainnya opsional dan ditampilkan sesuai hak akses petugas."
          : "Tanggal, waktu, dan pejabat tujuan wajib diisi. Rincian lainnya ditampilkan sesuai hak akses petugas."
      }
      onClose={close}
      returnFocusTo={returnFocusTo}
    >
      {discard ? (
        <div className="discard-prompt">
          <h3>Buang perubahan?</h3>
          <p>Isian yang belum disimpan akan dihapus.</p>
          <div className="form-actions">
            <Button
              type="button"
              kind="secondary"
              ref={resume}
              onClick={() => setDiscard(false)}
            >
              Lanjutkan mengisi
            </Button>
            <Button kind="danger" onClick={onClose}>
              Buang perubahan
            </Button>
          </div>
        </div>
      ) : (
        <form noValidate onSubmit={f.handleSubmit(submit)}>
          <RequiredFieldsHint />
          <FormSection disabled={busy} legend="Detail kunjungan">
            <div className="form-grid">
              <Controller
                name="date"
                control={f.control}
                render={({ field, fieldState }) => (
                  <DatePicker
                    {...field}
                    id="restricted-date"
                    label="Tanggal"
                    min={today()}
                    required
                    readOnly={!!booking}
                    disabled={busy}
                    error={fieldState.error?.message}
                  />
                )}
              />
              <Controller
                name="time"
                control={f.control}
                render={({ field, fieldState }) => (
                  <TimePicker
                    min="08:00"
                    max="16:00"
                    {...field}
                    id="restricted-time"
                    label="Waktu mulai · WITA"
                    required
                    readOnly={!!booking}
                    disabled={busy}
                    error={fieldState.error?.message}
                  />
                )}
              />
            </div>
            {booking && (
              <p className="muted">
                Gunakan Ubah jadwal pada detail kunjungan untuk mengganti tanggal atau waktu.
              </p>
            )}
            <div className="form-grid section-gap">
              {field("name", "Nama", 120)}
              {field("organization", "Instansi", 255)}
              {field("serviceId", "No ID Dinas", 100)}
              <div className="field">
                <label htmlFor="restricted-host">
                  Pejabat yang dituju {preview && <span className="muted">(opsional)</span>}
                </label>
                <Select
                  {...f.register("officerId")}
                  id="restricted-host"
                  required={!preview}
                  aria-invalid={!!f.formState.errors.officerId}
                  aria-describedby={
                    f.formState.errors.officerId ? "restricted-host-error" : undefined
                  }
                >
                  <option value="">Belum ditentukan</option>
                  {officers
                    .filter((officer) => officer.active)
                    .map((officer) => (
                      <option key={officer.id} value={officer.id}>
                        {officer.name}
                      </option>
                    ))}
                </Select>
                {f.formState.errors.officerId && (
                  <FieldMessage tone="error" id="restricted-host-error">
                    {f.formState.errors.officerId.message}
                  </FieldMessage>
                )}
              </div>
              {!booking && (
                <AppointmentRoomField
                  id="restricted-room"
                  rooms={rooms}
                  registration={f.register("room")}
                  error={f.formState.errors.room?.message}
                />
              )}
            </div>
            <VisitNotesFields
              visitor={f.register("visitorInstructions")}
              coordination={f.register("coordinationNote")}
              visitorError={f.formState.errors.visitorInstructions?.message}
              coordinationError={f.formState.errors.coordinationNote?.message}
            />
          </FormSection>
          <ReferenceDataNotice
            loading={catalogs.isPending}
            error={catalogs.error}
            onRetry={() => void catalogs.refetch()}
          />
          <ErrorNotice error={error} />
          <div className="form-actions">
            <Button type="button" kind="ghost" onClick={close} disabled={busy}>
              Batal
            </Button>
            {(!booking || dirty) && (
              <Button type="submit" disabled={busy}>
                {busy ? "Menyimpan…" : booking ? "Simpan perubahan" : "Buat kunjungan terbatas"}
              </Button>
            )}
          </div>
        </form>
      )}
    </Modal>
  );
}
