import { zodResolver } from "@hookform/resolvers/zod";
import {
  useUnsavedChanges,
  ApiError,
  canCreateAppointment,
  notaDinasType,
  phoneDigits,
  today,
  useCatalogs,
  useCreateAppointment,
  useSession,
  validateND,
  validateVisitorPhoto,
  visitorPhotoTypes,
  type StaffBooking,
} from "@vms/data";
import VisitNotesFields from "./VisitNotesFields";
import { AppointmentRoomField } from "./AppointmentRoomField";
import { visitFiles } from "@vms/data/visit-files";
import { staffAppointmentSchemaFor, type StaffAppointmentDraft } from "@vms/data/validation";
import {
  Select,
  Button,
  EmailInput,
  FieldMessage,
  Empty,
  ErrorNotice,
  Modal,
  RequiredFieldsHint,
  RequiredMark,
  UploadField,
  FormSection,
} from "@vms/ui";
import { DatePicker } from "@vms/ui/date-picker";
import { ReferenceDataNotice } from "@vms/ui/reference-data-notice";
import { TimePicker } from "@vms/ui/time-picker";
import { toast } from "@vms/ui/toast";
import { Plus, Trash2 } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Controller, useFieldArray, useForm, type FieldPath } from "react-hook-form";

export default function CreateAppointment({
  onClose,
  onCreated,
  returnFocusTo,
}: {
  onClose: () => void;
  onCreated: (booking: StaffBooking) => void;
  returnFocusTo?: HTMLElement | null;
}) {
  const { session } = useSession();
  const catalogs = useCatalogs();
  const officers = catalogs.data?.officers || [];
  const rooms = catalogs.data?.roomOptions ?? [];
  const rules = useMemo(() => staffAppointmentSchemaFor(officers), [catalogs.data]);
  const allowed = officers.filter(
    (o) => session && canCreateAppointment(session, o.id, "standard", officers),
  );
  const mutation = useCreateAppointment();
  const [error, setError] = useState<unknown>();
  const [document, setDocument] = useState<File | null>(null);
  const [photo, setPhoto] = useState<File | null>(null);
  const [photoError, setPhotoError] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const busy = uploading || mutation.isPending;
  const [fileError, setFileError] = useState<string | null>(null);
  const [discard, setDiscard] = useState(false);
  const [step, setStep] = useState(0);
  const attempt = useRef<{ payload: string; key: string } | null>(null);
  const submitLock = useRef(false);
  const resumeButton = useRef<HTMLButtonElement>(null);
  const reviewHeading = useRef<HTMLHeadingElement>(null);
  const editingReview = useRef(false);
  const f = useForm<StaffAppointmentDraft>({
    resolver: zodResolver(rules),
    mode: "onBlur",
    reValidateMode: "onChange",
    defaultValues: {
      officerId: allowed.length === 1 ? allowed[0].id : "",
      date: "",
      time: "",
      duration: 60,
      room: "",
      purpose: "",
      name: "",
      organization: "",
      email: "",
      phone: "",
      identityType: "KTP",
      visitors: [],
      visitorInstructions: "",
      coordinationNote: "",
    },
  });
  const assignedDestination =
    session?.role === "PA" && allowed.length === 1 ? allowed[0] : undefined;
  const soleDestinationId = allowed.length === 1 ? allowed[0].id : undefined;
  const { getValues, setValue } = f;
  useEffect(() => {
    if (soleDestinationId && getValues("officerId") !== soleDestinationId) {
      setValue("officerId", soleDestinationId, { shouldValidate: true });
    }
  }, [soleDestinationId, getValues, setValue]);
  const roster = useFieldArray({ control: f.control, name: "visitors" });
  const { isDirty } = f.formState;
  useUnsavedChanges(isDirty || !!document || !!photo, busy);
  const stepNames = ["Kunjungan", "Pengunjung", "Periksa"];
  const values = f.getValues();
  const roomName = rooms.find((room) => room.id === values.room)?.name;
  const reviewDate = values.date
    ? new Intl.DateTimeFormat("id-ID", {
        dateStyle: "long",
        timeZone: "Asia/Makassar",
      }).format(new Date(`${values.date}T12:00:00+08:00`))
    : "";
  function goToStep(next: number, focusId?: string) {
    setStep(next);
    requestAnimationFrame(() => {
      if (focusId) window.document.getElementById(focusId)?.focus();
      else if (next === 2) {
        reviewHeading.current?.focus({ preventScroll: true });
        const scroll = reviewHeading.current?.closest(".modal-scroll");
        if (scroll) scroll.scrollTop = 0;
      } else f.setFocus(next === 0 ? "officerId" : "name");
    });
  }
  function editSection(next: number, focusId?: string) {
    editingReview.current = true;
    goToStep(next, focusId);
  }
  async function nextStep() {
    const fields =
      step === 0
        ? ([
            "officerId",
            "date",
            "time",
            "duration",
            "room",
            "purpose",
            "visitorInstructions",
            "coordinationNote",
          ] as const)
        : (["name", "organization", "email", "phone", "identityType", "visitors"] as const);
    if (await f.trigger([...fields], { shouldFocus: true })) {
      if (step === 0 && fileError) {
        window.document.getElementById("appointment-document")?.focus();
        return;
      }
      if (step === 1 && (!photo || photoError)) {
        setPhotoError(photoError || "Unggah foto wajah perwakilan.");
        window.document.getElementById("appointment-photo")?.focus();
        return;
      }
      const next = editingReview.current ? 2 : step + 1;
      editingReview.current = false;
      goToStep(next);
    }
  }
  function close() {
    if (submitLock.current) return;
    if (isDirty || document || photo) {
      setDiscard(true);
      requestAnimationFrame(() => resumeButton.current?.focus());
    } else onClose();
  }
  async function submit(values: StaffAppointmentDraft) {
    if (submitLock.current) return;
    if (fileError || photoError || !photo) {
      if (!fileError) setPhotoError(photoError || "Unggah foto wajah perwakilan.");
      goToStep(fileError ? 0 : 1, fileError ? "appointment-document" : "appointment-photo");
      return;
    }
    if (!session || !canCreateAppointment(session, values.officerId, "standard", officers)) {
      f.setError(
        "officerId",
        { message: "Pilih tujuan dalam kewenangan Anda." },
        { shouldFocus: true },
      );
      return;
    }
    submitLock.current = true;
    setError(null);
    setUploading(true);
    try {
      const { room, ...details } = values;
      const input = {
        ...details,
        roomId: room || undefined,
        visitors: [values.name, ...values.visitors.map((v) => v.name)],
        ndName: document?.name,
        attachmentToken: await visitFiles.upload(document, "staff", "attachment"),
        visitorPhotoName: photo.name,
        visitorPhotoToken: await visitFiles.upload(photo, "staff", "visitor-photo"),
      };
      const payload = JSON.stringify(input);
      if (attempt.current?.payload !== payload)
        attempt.current = { payload, key: crypto.randomUUID() };
      const created = await mutation.mutateAsync({
        input,
        idempotencyKey: attempt.current.key,
      });
      onCreated(created);
    } catch (err) {
      setError(err);
      if (err instanceof ApiError) {
        const documentIssue = err.fieldErrors?.ndName?.[0] || err.fieldErrors?.attachmentToken?.[0];
        const photoIssue = err.fieldErrors?.visitorPhotoName?.[0] || err.fieldErrors?.visitorPhotoToken?.[0];
        if (documentIssue) setFileError(documentIssue);
        if (photoIssue) setPhotoError(photoIssue);
        if (documentIssue || photoIssue) {
          editingReview.current = true;
          goToStep(documentIssue ? 0 : 1, documentIssue ? "appointment-document" : "appointment-photo");
        }
      }
      if (err instanceof ApiError && err.status === 409)
        toast("Jadwal ruangan atau pejabat bertabrakan. Pilih ruangan atau waktu lain.", "error");
      if (err instanceof ApiError && err.fieldErrors) {
        const fieldErrors = { ...err.fieldErrors };
        if (fieldErrors.roomId) fieldErrors.room = fieldErrors.roomId;
        const keys = [
          "officerId",
          "date",
          "time",
          "duration",
          "room",
          "purpose",
          "name",
          "organization",
          "email",
          "phone",
          "identityType",
          "visitors",
        ] as const;
        const first = keys.find((key) => fieldErrors[key]?.[0]);
        for (const key of keys)
          if (fieldErrors[key]?.[0])
            f.setError(key, {
              type: "server",
              message: fieldErrors[key][0],
            });
        if (first === "room") editSection(0, "appointment-room");
        else if (first && first !== "visitors") requestAnimationFrame(() => f.setFocus(first));
      }
    } finally {
      submitLock.current = false;
      setUploading(false);
    }
  }
  function field(key: "name" | "organization" | "phone", label: string, type = "text") {
    const registration = f.register(key);
    const issue = f.formState.errors[key];
    return (
      <div className="field">
        <label htmlFor={`appointment-${key}`}>
          {label} <RequiredMark />
        </label>
        <input
          {...registration}
          id={`appointment-${key}`}
          type={type}
          required
          autoComplete={
            key === "name"
              ? "name"
              : key === "organization"
                ? "organization"
                : key === "phone"
                  ? "tel"
                  : undefined
          }
          inputMode={key === "phone" ? "numeric" : undefined}
          maxLength={key === "phone" ? 20 : key === "name" ? 120 : undefined}
          onChange={async (event) => {
            if (key === "phone") event.target.value = phoneDigits(event.target.value);
            await registration.onChange(event);
            if (f.formState.errors[key]) void f.trigger(key);
          }}
          onBlur={async (event) => {
            await registration.onBlur(event);
            if (f.formState.errors[key]) void f.trigger(key);
          }}
          aria-invalid={!!issue}
          aria-describedby={issue ? `appointment-${key}-error` : undefined}
        />
        {issue && (
          <span className="field-error" role="alert" id={`appointment-${key}-error`}>
            {issue.message}
          </span>
        )}
      </div>
    );
  }
  return (
    <Modal
      open
      className="staff-appointment-modal"
      onClose={close}
      title="Buat janji temu"
      returnFocusTo={returnFocusTo}
      description="Catat kunjungan atas nama pengaju. Janji temu langsung disetujui setelah disimpan."
    >
      {discard && (
        <div className="decision-note section-gap" role="alert">
          <h3>Tutup tanpa menyimpan?</h3>
          <p>Isian janji temu ini akan dihapus.</p>
          <div className="form-actions">
            <button
              className="button secondary"
              type="button"
              ref={resumeButton}
              onClick={() => {
                setDiscard(false);
                requestAnimationFrame(() => f.setFocus("officerId"));
              }}
            >
              Lanjutkan mengisi
            </button>
            <Button kind="danger" type="button" onClick={onClose}>
              Buang isian
            </Button>
          </div>
        </div>
      )}
      {!allowed.length ? (
        <Empty
          title="Belum ada tujuan dalam kewenangan Anda"
          text="Penugasan pejabat perlu tersedia sebelum membuat janji temu."
        />
      ) : (
        <form
          className="staff-appointment-form section-gap"
          hidden={discard}
          noValidate
          onSubmit={(event) => {
            event.preventDefault();
            if (step < 2) void nextStep();
            else void f.handleSubmit(submit)(event);
          }}
        >
          <nav className="appointment-steps" aria-label="Langkah janji temu">
            {stepNames.map((name, index) => (
              <span
                key={name}
                className={index === step ? "active" : ""}
                aria-current={index === step ? "step" : undefined}
              >
                {index + 1}. {name}
              </span>
            ))}
          </nav>
          {step < 2 && <RequiredFieldsHint />}
          <div hidden={step !== 0}>
            <FormSection disabled={busy} legend="Detail kunjungan">
              <div className="form-grid">
                <div className="field span-2">
                  <label htmlFor="appointment-officerId">
                    Pejabat atau unit tujuan <RequiredMark />
                  </label>
                  {assignedDestination ? (
                    <>
                      <input type="hidden" {...f.register("officerId")} />
                      <input
                        id="appointment-officerId"
                        value={assignedDestination.name}
                        readOnly
                        aria-invalid={!!f.formState.errors.officerId}
                        aria-describedby={
                          f.formState.errors.officerId
                            ? "appointment-officerId-error"
                            : "appointment-scope-hint"
                        }
                      />
                    </>
                  ) : (
                    <Select
                      {...f.register("officerId")}
                      id="appointment-officerId"
                      required
                      aria-invalid={!!f.formState.errors.officerId}
                      aria-describedby={
                        f.formState.errors.officerId
                          ? "appointment-officerId-error"
                          : "appointment-scope-hint"
                      }
                    >
                      <option value="">Pilih tujuan kunjungan</option>
                      {allowed.map((o) => (
                        <option key={o.id} value={o.id}>
                          {o.name}
                          {o.vip ? " · VIP" : ""}
                        </option>
                      ))}
                    </Select>
                  )}
                  <small id="appointment-scope-hint">
                    {session?.role === "PA"
                      ? assignedDestination
                        ? "Tujuan otomatis sesuai pejabat yang ditugaskan kepada Anda."
                        : "Tujuan sesuai penugasan PA Anda."
                      : "Kunjungan VIP dalam kewenangan Protokol."}
                  </small>
                  {f.formState.errors.officerId && (
                    <span id="appointment-officerId-error" className="field-error" role="alert">
                      {f.formState.errors.officerId.message}
                    </span>
                  )}
                </div>
                <Controller
                  name="date"
                  control={f.control}
                  render={({ field, fieldState }) => (
                    <DatePicker
                      {...field}
                      id="appointment-date"
                      label="Tanggal kunjungan"
                      min={today()}
                      required
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
                      id="appointment-time"
                      label="Waktu mulai · WITA"
                      required
                      disabled={busy}
                      error={fieldState.error?.message}
                    />
                  )}
                />
                <UploadField
                  id="appointment-document"
                  label="Nota Dinas atau Permohonan Kunjungan Resmi (opsional)"
                  accept={notaDinasType}
                  file={document}
                  error={fileError}
                  hint="PDF · Maksimal 5 MB."
                  choose="Pilih dokumen"
                  remove="Hapus dokumen"
                  onFileChange={(file) => {
                    const message = validateND(file);
                    setFileError(message);
                    setDocument(message ? null : file);
                  }}
                />
                <div className="field">
                  <label htmlFor="appointment-duration">
                    Durasi kunjungan (menit) <RequiredMark />
                  </label>
                  <input
                    {...f.register("duration", { valueAsNumber: true })}
                    id="appointment-duration"
                    type="number"
                    min={15}
                    max={240}
                    step={1}
                    required
                    aria-invalid={!!f.formState.errors.duration}
                    aria-describedby={
                      f.formState.errors.duration ? "appointment-duration-error" : undefined
                    }
                  />
                  {f.formState.errors.duration && (
                    <span id="appointment-duration-error" role="alert" className="field-error">
                      {f.formState.errors.duration.message}
                    </span>
                  )}
                </div>
                <AppointmentRoomField
                  id="appointment-room"
                  rooms={rooms}
                  registration={f.register("room")}
                  error={f.formState.errors.room?.message}
                />
                <div className="field span-2">
                  <label htmlFor="appointment-purpose">
                    Keperluan kunjungan <RequiredMark />
                  </label>
                  <textarea
                    {...f.register("purpose")}
                    id="appointment-purpose"
                    rows={3}
                    required
                    maxLength={1000}
                    aria-invalid={!!f.formState.errors.purpose}
                    aria-describedby={
                      f.formState.errors.purpose ? "appointment-purpose-error" : undefined
                    }
                  />
                  {f.formState.errors.purpose && (
                    <span id="appointment-purpose-error" role="alert" className="field-error">
                      {f.formState.errors.purpose.message}
                    </span>
                  )}
                </div>
              </div>
              <VisitNotesFields
                visitor={f.register("visitorInstructions")}
                coordination={f.register("coordinationNote")}
                visitorError={f.formState.errors.visitorInstructions?.message}
                coordinationError={f.formState.errors.coordinationNote?.message}
              />
            </FormSection>
          </div>
          <div hidden={step !== 1}>
            <FormSection disabled={busy} legend="Data pengunjung">
              <div className="form-grid">
                {field("name", "Nama lengkap perwakilan")}
                {field("organization", "Instansi atau asal")}
                <Controller
                  name="email"
                  control={f.control}
                  render={({ field: email, fieldState }) => (
                    <div className="field span-2">
                      <label htmlFor="appointment-email">
                        Email pengunjung <RequiredMark />
                      </label>
                      <EmailInput
                        id="appointment-email"
                        name={email.name}
                        ref={email.ref}
                        value={email.value}
                        onValueChange={email.onChange}
                        onBlur={email.onBlur}
                        onDomainPick={() => {
                          void f.trigger("email");
                        }}
                        required
                        aria-invalid={!!fieldState.error}
                        aria-describedby={fieldState.error ? "appointment-email-error" : undefined}
                      />
                      {fieldState.error && (
                        <FieldMessage id="appointment-email-error" tone="error">
                          {fieldState.error.message}
                        </FieldMessage>
                      )}
                    </div>
                  )}
                />
                {field("phone", "Nomor telepon", "tel")}
                <div className="field">
                  <label htmlFor="appointment-identityType">
                    Identitas yang dibawa <RequiredMark />
                  </label>
                  <Select
                    id="appointment-identityType"
                    required
                    {...f.register("identityType")}
                    aria-invalid={!!f.formState.errors.identityType}
                    aria-describedby="appointment-identity-hint"
                  >
                    <option>KTP</option>
                    <option>Paspor</option>
                    <option>SIM</option>
                  </Select>
                  <FieldMessage id="appointment-identity-hint">
                    Identitas asli diperiksa saat kedatangan. Nomor dan foto identitas tidak diminta.
                  </FieldMessage>
                </div>
              </div>
              <div className="form-group-gap">
                <UploadField
                  id="appointment-photo"
                  label="Foto wajah perwakilan"
                  required
                  accept={visitorPhotoTypes.join(",")}
                  file={photo}
                  error={photoError}
                  hint="JPEG, PNG · Maksimal 5 MB"
                  choose="Pilih foto wajah"
                  remove="Hapus foto"
                  photoPreviewLabel="Pratinjau foto wajah perwakilan"
                  onFileChange={(file) => {
                    const message = validateVisitorPhoto(file);
                    setPhotoError(message);
                    setPhoto(message ? null : file);
                  }}
                />
              </div>
            </FormSection>
            <FormSection disabled={busy} className="form-group-gap" legend="Rombongan pengunjung">
              <div className="form-stack">
                <small className="form-help">
                  {roster.fields.length + 1} orang termasuk perwakilan. Maksimal 20 orang.
                </small>
                {roster.fields.map((person, i) => {
                  const key = `visitors.${i}.name` as FieldPath<StaffAppointmentDraft>;
                  const issue = f.formState.errors.visitors?.[i]?.name;
                  return (
                    <div className="form-roster-row" key={person.id}>
                      <div className="field">
                        <label htmlFor={`appointment-visitor-${i}`}>
                          Nama pengunjung {i + 2} <RequiredMark />
                        </label>
                        <input
                          {...f.register(key)}
                          id={`appointment-visitor-${i}`}
                          maxLength={120}
                          required
                          aria-invalid={!!issue}
                          aria-describedby={issue ? `appointment-visitor-${i}-error` : undefined}
                        />
                        {issue && (
                          <span
                            id={`appointment-visitor-${i}-error`}
                            className="field-error"
                            role="alert"
                          >
                            {issue.message}
                          </span>
                        )}
                      </div>
                      <Button
                        type="button"
                        kind="ghost"
                        aria-label={`Hapus pengunjung ${i + 2}`}
                        onClick={() => roster.remove(i)}
                      >
                        <Trash2 size={18} />
                      </Button>
                    </div>
                  );
                })}
                {f.formState.errors.visitors?.message && (
                  <span className="field-error" role="alert">
                    {f.formState.errors.visitors.message}
                  </span>
                )}
                <Button
                  kind="secondary"
                  type="button"
                  onClick={() => {
                    roster.append({ name: "" });
                  }}
                  disabled={roster.fields.length >= 19}
                >
                  <Plus size={16} />
                  Tambah pengunjung
                </Button>
              </div>
            </FormSection>
          </div>
          {step === 2 && (
            <div className="appointment-review">
              <div className="appointment-review-intro">
                <h3 ref={reviewHeading} tabIndex={-1}>
                  Periksa janji temu
                </h3>
                <p>Pastikan rincian kunjungan dan data pengunjung sudah benar sebelum menyimpan.</p>
              </div>
              <section className="appointment-review-section" aria-labelledby="review-visit-title">
                <div className="appointment-review-heading">
                  <h4 id="review-visit-title">Detail kunjungan</h4>
                  <Button
                    type="button"
                    kind="ghost"
                    disabled={busy}
                    onClick={() => editSection(0)}
                    aria-label="Ubah detail kunjungan"
                  >
                    Ubah
                  </Button>
                </div>
                <dl className="appointment-review-grid">
                  <div className="appointment-review-wide">
                    <dt>Pejabat atau unit tujuan</dt>
                    <dd>{allowed.find((o) => o.id === values.officerId)?.name}</dd>
                  </div>
                  <div>
                    <dt>Tanggal kunjungan</dt>
                    <dd>{reviewDate}</dd>
                  </div>
                  <div>
                    <dt>Waktu mulai</dt>
                    <dd>{values.time} WITA</dd>
                  </div>
                  <div>
                    <dt>Durasi kunjungan</dt>
                    <dd>{values.duration} menit</dd>
                  </div>
                  <div>
                    <dt>Ruangan kunjungan</dt>
                    <dd>{roomName || "Belum ditetapkan"}</dd>
                  </div>
                  <div className="appointment-review-wide">
                    <dt>Keperluan kunjungan</dt>
                    <dd className="appointment-review-purpose">{values.purpose}</dd>
                  </div>
                  {values.visitorInstructions && (
                    <div className="appointment-review-wide">
                      <dt>Instruksi pengunjung</dt>
                      <dd className="visit-note-text">{values.visitorInstructions}</dd>
                    </div>
                  )}
                  {values.coordinationNote && (
                    <div className="appointment-review-wide">
                      <dt>Catatan koordinasi internal</dt>
                      <dd className="visit-note-text">{values.coordinationNote}</dd>
                    </div>
                  )}
                </dl>
              </section>
              <section
                className="appointment-review-section"
                aria-labelledby="review-applicant-title"
              >
                <div className="appointment-review-heading">
                  <h4 id="review-applicant-title">Data perwakilan</h4>
                  <Button
                    type="button"
                    kind="ghost"
                    disabled={busy}
                    onClick={() => editSection(1)}
                    aria-label="Ubah data perwakilan"
                  >
                    Ubah
                  </Button>
                </div>
                <dl className="appointment-review-grid">
                  <div>
                    <dt>Nama lengkap</dt>
                    <dd>{values.name}</dd>
                  </div>
                  <div>
                    <dt>Instansi atau asal</dt>
                    <dd>{values.organization}</dd>
                  </div>
                  <div className="appointment-review-wide">
                    <dt>Email pengunjung</dt>
                    <dd>{values.email}</dd>
                  </div>
                  <div>
                    <dt>Nomor telepon</dt>
                    <dd>{values.phone}</dd>
                  </div>
                  <div>
                    <dt>Identitas yang dibawa</dt>
                    <dd>{values.identityType}</dd>
                  </div>
                  <div className="appointment-review-wide">
                    <dt>Foto wajah perwakilan</dt>
                    <dd>{photo?.name || "Tidak dilampirkan."}</dd>
                  </div>
                </dl>
              </section>
              <section className="appointment-review-section" aria-labelledby="review-roster-title">
                <div className="appointment-review-heading">
                  <h4 id="review-roster-title">
                    Rombongan pengunjung <span>· {values.visitors.length + 1} orang</span>
                  </h4>
                  <Button
                    type="button"
                    kind="ghost"
                    disabled={busy}
                    onClick={() => editSection(1)}
                    aria-label="Ubah rombongan pengunjung"
                  >
                    Ubah
                  </Button>
                </div>
                <ol className="appointment-review-roster">
                  <li>
                    {values.name} <span>Perwakilan</span>
                  </li>
                  {values.visitors.map((person, index) => (
                    <li key={roster.fields[index].id}>{person.name}</li>
                  ))}
                </ol>
              </section>
              <section
                className="appointment-review-section"
                aria-labelledby="review-document-title"
              >
                <div className="appointment-review-heading">
                  <h4 id="review-document-title">Dokumen pendukung</h4>
                  <Button
                    type="button"
                    kind="ghost"
                    disabled={busy}
                    onClick={() => editSection(0, "appointment-document")}
                    aria-label="Ubah dokumen pendukung"
                  >
                    Ubah
                  </Button>
                </div>
                <p className="appointment-review-document">
                  {document?.name || "Tidak ada dokumen dilampirkan."}
                </p>
              </section>
            </div>
          )}
          <ReferenceDataNotice
            loading={catalogs.isPending}
            error={catalogs.error}
            onRetry={() => void catalogs.refetch()}
          />
          <ErrorNotice error={error} />
          <div className="form-actions">
            <Button
              type="button"
              kind="ghost"
              onClick={
                step === 0
                  ? close
                  : () => {
                      editingReview.current = false;
                      goToStep(step - 1);
                    }
              }
              disabled={busy}
            >
              {step === 0 ? "Batal" : "Kembali"}
            </Button>
            <Button
              type="submit"
              disabled={busy}
            >
              {busy
                ? "Menyimpan…"
                : step === 2
                  ? "Buat janji temu"
                  : editingReview.current
                    ? "Kembali ke periksa"
                    : "Lanjutkan"}
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
}
