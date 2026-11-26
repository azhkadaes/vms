import { zodResolver } from "@hookform/resolvers/zod";
import {
  ApiError,
  dateLabel,
  notaDinasType,
  phoneDigits,
  preview,
  repository,
  today,
  useCatalogs,
  validateND,
  validateVisitorPhoto,
  visitorPhotoTypes,
  type PublicBooking,
} from "@vms/data";
import { visitFiles } from "@vms/data/visit-files";
import { BookingSubmission, type SubmissionPhase } from "@vms/data/submission";
import {
  bookingSchemaFor,
  publicBookingInputSchema,
  visitSchemaFor,
  type BookingDraft,
} from "@vms/data/validation";
import {
  Select,
  Button,
  EmailInput,
  FieldMessage,
  ErrorNotice,
  Modal,
  RequiredFieldsHint,
  RequiredMark,
  UploadField,
} from "@vms/ui";
import { DatePicker } from "@vms/ui/date-picker";
import { localizeMessage } from "@vms/ui/language";
import { ReferenceDataNotice } from "@vms/ui/reference-data-notice";
import { TimePicker } from "@vms/ui/time-picker";
import {
  ArrowLeft,
  ArrowRight,
  CalendarDays,
  Check,
  Copy,
  FileText,
  Plus,
  ShieldCheck,
  Trash2,
  Users,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Controller, useFieldArray, useForm, type FieldPath } from "react-hook-form";
import { Link, useLocation, useNavigate } from "react-router";
import PrivacyNotice from "../features/PrivacyNotice";
import SubmissionRecovery from "../features/SubmissionRecovery";
import { useLanguage } from "../i18n";

const submissionStorageKey = "vms-pending-submission";
const visitKeys = ["officerId", "date", "time", "purpose"] as const;
const visitorKeys = ["name", "organization", "email", "phone", "identityType", "visitors"] as const;
type BookingFocus = FieldPath<BookingDraft> | "nd" | "visitorPhoto";
const submissionStore = {
  read() {
    try {
      return sessionStorage.getItem(submissionStorageKey);
    } catch {
      return null;
    }
  },
  write(key: string) {
    try {
      sessionStorage.setItem(submissionStorageKey, key);
    } catch {
      throw new Error(
        "Browser tidak dapat menyimpan referensi pengiriman. Izinkan penyimpanan sesi lalu coba lagi.",
      );
    }
  },
  clear() {
    sessionStorage.removeItem(submissionStorageKey);
  },
};
export default function BookingForm({
  embedded = false,
  active = true,
  headingLevel = 1,
}: {
  embedded?: boolean;
  active?: boolean;
  headingLevel?: 1 | 2;
}) {
  const { language, t } = useLanguage();
  const catalogs = useCatalogs();
  const officers = catalogs.data?.officers || [];
  const bookingRules = useMemo(() => bookingSchemaFor(officers), [catalogs.data]);
  const visitRules = useMemo(() => visitSchemaFor(officers), [catalogs.data]);
  const IntroHeading = headingLevel === 1 ? "h1" : "h2";
  const FormHeading = headingLevel === 1 ? "h2" : "h3";
  const GroupHeading = headingLevel === 1 ? "h3" : "h4";
  const navigate = useNavigate();
  const location = useLocation();
  const step = location.pathname.endsWith("/review")
    ? 2
    : location.pathname.endsWith("/visitors")
      ? 1
      : 0;
  const stepTitles = [
    t("Rencanakan kunjungan Anda.", "Plan your visit."),
    t("Lengkapi data pengunjung.", "Add visitor details."),
    t("Tinjau dan kirim permohonan.", "Review and submit your request."),
  ];
  const stepDescriptions = [
    t(
      "Pilih tujuan, tanggal, dan waktu sebelum mengisi data pengunjung.",
      "Choose who to meet, a date, and a time before adding visitor details.",
    ),
    t(
      "Tambahkan perwakilan, kontak, dan anggota rombongan yang hadir.",
      "Add the representative, contact details, and other visitors.",
    ),
    t(
      "Periksa rincian kunjungan dan data pengunjung sebelum mengirim.",
      "Review the visit and visitor details before submitting.",
    ),
  ];
  const stepPaths = ["/booking", "/booking/visitors", "/booking/review"];
  const [submission] = useState(() => new BookingSubmission(repository, submissionStore));
  const [phase, setPhase] = useState<SubmissionPhase>(submission.phase);
  const [recoveryEmail, setRecoveryEmail] = useState("");
  const [submissionNotice, setSubmissionNotice] = useState("");
  const recoveryHeading = useRef<HTMLDivElement>(null);
  const successHeading = useRef<HTMLHeadingElement>(null);
  const [uploading, setUploading] = useState(false);
  const submitLock = useRef(false);
  const pending = uploading || ["sending", "retrying", "checking"].includes(phase);
  const unresolved = phase === "uncertain" || phase === "checking";
  const focusAfterNavigation = useRef<BookingFocus | null>(null);
  const [privacy, setPrivacy] = useState(false);
  const privacyTrigger = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (!active) setPrivacy(false);
  }, [active]);
  const [submitted, setSubmitted] = useState<PublicBooking | null>(null);
  const [hasNotaDinas, setHasNotaDinas] = useState(false);
  const [nd, setND] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [visitorPhoto, setVisitorPhoto] = useState<File | null>(null);
  const [photoError, setPhotoError] = useState<string | null>(null);
  const [error, setError] = useState<unknown>();
  const [copied, setCopied] = useState(false);
  const f = useForm<BookingDraft>({
    resolver: zodResolver(bookingRules),
    mode: "onBlur",
    reValidateMode: "onChange",
    defaultValues: {
      officerId: "",
      date: "",
      time: "",
      purpose: "",
      name: "",
      organization: "",
      email: "",
      phone: "",
      identityType: "KTP",
      visitors: [],
      ack: false,
    },
  });
  const array = useFieldArray({ control: f.control, name: "visitors" });
  const policyVersion = catalogs.data?.privacyNotice?.version;
  const acknowledgedVersion = useRef<string | undefined>(policyVersion);
  useEffect(() => {
    if (policyVersion !== acknowledgedVersion.current) {
      f.setValue("ack", false);
      acknowledgedVersion.current = policyVersion;
    }
  }, [policyVersion, f]);
  const values = f.watch();
  const resetBooking = useCallback(() => {
    f.reset();
    setSubmitted(null);
    setHasNotaDinas(false);
    setND(null);
    setFileError(null);
    setVisitorPhoto(null);
    setPhotoError(null);
    setError(undefined);
    setCopied(false);
    setSubmissionNotice("");
  }, [f]);
  useEffect(() => {
    if (step > 0 && !f.getValues("date")) {
      navigate("/booking", { replace: true });
      return;
    }
    const field = focusAfterNavigation.current;
    if (field) {
      focusAfterNavigation.current = null;
      requestAnimationFrame(() => focusField(field));
    }
  }, [step, navigate, f, submitted, location.pathname, resetBooking]);
  useEffect(() => {
    if (submitted && active) {
      successHeading.current?.focus();
      try {
        submission.confirmReceipt(setPhase);
      } catch (err) {
        setError(err);
      }
    } else if (phase === "uncertain" && active)
      recoveryHeading.current?.querySelector<HTMLElement>("#submission-recovery-title")?.focus();
  }, [submitted, phase, active, submission]);
  useEffect(() => {
    if (!unresolved && !pending) return;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [unresolved, pending]);
  const selectedOfficer = officers.find((o) => o.id === values.officerId);
  function focusField(field: BookingFocus) {
    if (field === "nd" || field === "visitorPhoto") document.getElementById(field)?.focus();
    else f.setFocus(field);
  }
  function focusUpload(field: "nd" | "visitorPhoto") {
    const owner = field === "nd" ? 0 : 1;
    if (owner === step) focusField(field);
    else {
      focusAfterNavigation.current = field;
      navigate(stepPaths[owner]);
    }
  }
  const fields = (
    key: keyof BookingDraft,
    label: string,
    type = "text",
    extra: Record<string, unknown> = {},
  ) => {
    const registration = f.register(key);
    return (
      <div className="field">
        <label htmlFor={key}>
          {label} <RequiredMark />
        </label>
        <input
          id={key}
          type={type}
          {...registration}
          {...extra}
          onChange={(event) => {
            if (key === "phone") event.target.value = phoneDigits(event.target.value);
            void registration.onChange(event);
          }}
          aria-invalid={!!f.formState.errors[key]}
          aria-describedby={f.formState.errors[key] ? `${key}-error` : undefined}
        />
        {f.formState.errors[key] && (
          <FieldMessage tone="error" id={`${key}-error`}>
            {localizeMessage(String(f.formState.errors[key]?.message), language)}
          </FieldMessage>
        )}
      </div>
    );
  };
  async function next() {
    if (step === 0) {
      f.clearErrors(["officerId", "date", "time", "purpose"]);
      const result = visitRules.safeParse(f.getValues());
      if (!result.success) {
        for (const issue of result.error.issues)
          f.setError(issue.path[0] as keyof BookingDraft, {
            type: "validate",
            message: issue.message,
          });
        f.setFocus(result.error.issues[0].path[0] as keyof BookingDraft);
        return;
      }
      if (fileError) {
        focusUpload("nd");
        return;
      }
      navigate("/booking/visitors");
    } else if (
      await f.trigger(["name", "organization", "email", "phone", "identityType", "visitors"], {
        shouldFocus: true,
      })
    ) {
      if (photoError || !visitorPhoto) {
        setPhotoError(photoError || "Unggah foto wajah perwakilan.");
        focusUpload("visitorPhoto");
        return;
      }
      navigate("/booking/review");
    }
  }
  async function submit(data: BookingDraft) {
    if (submitLock.current || pending || unresolved || !submission.canSubmit) return;
    if (fileError || photoError || !visitorPhoto) {
      if (!fileError) setPhotoError(photoError || "Unggah foto wajah perwakilan.");
      focusUpload(fileError ? "nd" : "visitorPhoto");
      return;
    }
    if (!preview && !catalogs.data?.privacyNotice) {
      setError(
        new Error(
          t(
            "Pemberitahuan privasi belum tersedia. Coba lagi sebelum mengirim.",
            "The privacy notice is unavailable. Try again before submitting.",
          ),
        ),
      );
      return;
    }
    submitLock.current = true;
    setError(null);
    setSubmissionNotice("");
    try {
      setUploading(true);
      const parsed = publicBookingInputSchema.safeParse({
        officerId: data.officerId,
        date: data.date,
        time: data.time,
        purpose: data.purpose,
        name: data.name,
        organization: data.organization,
        email: data.email,
        phone: data.phone,
        identityType: data.identityType,
        visitors: [data.name, ...data.visitors.map((v) => v.name)],
        ndName: nd?.name,
        attachmentToken: await visitFiles.upload(nd, "public", "attachment"),
        visitorPhotoName: visitorPhoto.name,
        visitorPhotoToken: await visitFiles.upload(visitorPhoto, "public", "visitor-photo"),
        privacyVersion: catalogs.data?.privacyNotice?.version,
        ack: data.ack,
      });
      if (!parsed.success) {
        showFieldErrors(parsed.error.flatten().fieldErrors);
        return;
      }
      setRecoveryEmail(parsed.data.email);
      const b = await submission.submit(parsed.data, setPhase);
      setSubmitted(b);
    } catch (err) {
      setError(
        submission.phase === "uncertain"
          ? new Error(
              t(
                "Hasil pengiriman belum dapat dipastikan. Periksa pengiriman sebelumnya untuk melanjutkan.",
                "The submission result could not be confirmed. Check the previous submission to continue.",
              ),
            )
          : err,
      );
      if (err instanceof ApiError && err.fieldErrors) showFieldErrors(err.fieldErrors);
    } finally {
      setUploading(false);
      submitLock.current = false;
    }
  }
  function showFieldErrors(errors: Record<string, string[] | undefined>) {
    for (const key of [...visitKeys, ...visitorKeys, "ack"] as const) {
      if (errors[key]?.[0]) {
        if (key === "visitors") {
          const member = f
            .getValues("visitors")
            .findIndex((visitor) => visitor.name.trim().length < 2);
          f.setError(member < 0 ? "visitors" : `visitors.${member}.name`, {
            type: "server",
            message: errors[key]![0],
          });
        } else f.setError(key, { type: "server", message: errors[key]![0] });
      }
    }
    if (errors.ndName?.[0] || errors.attachmentToken?.[0]) setHasNotaDinas(true);
    if (errors.ndName?.[0]) setFileError(errors.ndName[0]);
    if (errors.attachmentToken?.[0]) setFileError(errors.attachmentToken[0]);
    if (errors.visitorPhotoName?.[0]) setPhotoError(errors.visitorPhotoName[0]);
    if (errors.visitorPhotoToken?.[0]) setPhotoError(errors.visitorPhotoToken[0]);
    const firstVisit = visitKeys.find((key) => errors[key]?.length);
    const firstVisitor = visitorKeys.find((key) => errors[key]?.length);
    const documentError = errors.ndName?.length || errors.attachmentToken?.length;
    const visitorPhotoError = errors.visitorPhotoName?.length || errors.visitorPhotoToken?.length;
    const owner = firstVisit || documentError ? 0 : firstVisitor || visitorPhotoError ? 1 : 2;
    const first: BookingFocus =
      firstVisit ||
      (documentError ? "nd" : firstVisitor || (visitorPhotoError ? "visitorPhoto" : "ack"));
    if (first === "nd" || first === "visitorPhoto") focusUpload(first);
    else {
      const members = f.getValues("visitors");
      const member = Math.max(
        0,
        members.findIndex((visitor) => visitor.name.trim().length < 2),
      );
      const target: FieldPath<BookingDraft> =
        first === "visitors" ? (members.length ? `visitors.${member}.name` : "name") : first;
      if (owner === step) f.setFocus(target);
      else {
        focusAfterNavigation.current = target;
        navigate(stepPaths[owner]);
      }
    }
  }
  async function recover(email: string) {
    setError(null);
    setSubmissionNotice("");
    try {
      const result = await submission.recover(email, setPhase);
      if (result.outcome === "committed") setSubmitted(result.booking);
      else
        setSubmissionNotice(
          t(
            "Pengiriman sebelumnya belum diterima dan referensinya sudah ditutup. Tinjau isian Anda, lalu kirim kembali bila sudah benar.",
            "The previous submission was not received and its reference is now closed. Review your entries, then submit again when they are correct.",
          ),
        );
    } catch (err) {
      setError(err);
    }
  }
  if (submitted)
    return (
      <div className="booking-container submission">
        <div className="submission-icon">
          <Check size={34} />
        </div>
        <IntroHeading className="submission-title" ref={successHeading} tabIndex={-1}>
          {t("Permohonan Anda", "Your request")}
          <br />
          {t("sudah kami terima.", "has been received.")}
        </IntroHeading>
        <p>
          {submitted.status === "pending"
            ? t(
                "Permohonan kunjungan masih menunggu persetujuan. Simpan kode berikut untuk memeriksa jadwal dan status Anda.",
                "Your visit request is awaiting approval. Save this code to check your schedule and status.",
              )
            : t(
                "Permohonan Anda sudah tercatat. Simpan kode ini dan periksa status kunjungan terbaru sebelum datang.",
                "Your request has been recorded. Save this code and check the current visit status before arriving.",
              )}
        </p>
        <div className="booking-code-display">
          <span>{t("Kode kunjungan", "Visit code")}</span>
          <strong>{submitted.code}</strong>
          <Button
            kind="ghost"
            aria-label={t("Salin kode kunjungan", "Copy visit code")}
            onClick={async () => {
              try {
                await navigator.clipboard.writeText(submitted.code);
                setCopied(true);
              } catch {
                setError(
                  t(
                    "Salin kode secara manual. Clipboard tidak tersedia.",
                    "Copy the code manually. Clipboard access is unavailable.",
                  ),
                );
              }
            }}
          >
            <Copy size={18} />
            {copied ? t("Tersalin", "Copied") : t("Salin", "Copy")}
          </Button>
        </div>
        <div className="hint">
          <ShieldCheck size={18} />
          {submitted.status === "pending"
            ? t(
                "Pengajuan belum menjadi izin masuk. Datang setelah kunjungan disetujui dan bawa identitas asli.",
                "A request does not grant entry. Come after your visit is approved and bring your original ID.",
              )
            : t(
                "Kode kunjungan saja bukan izin masuk. Periksa status dan jadwal terbaru melalui Cek status.",
                "A visit code alone does not grant entry. Use Check status to confirm the current status and schedule.",
              )}
        </div>
        <ErrorNotice error={error} />
        <div className="form-actions">
          <Link className="link-button secondary" to="/" onClick={resetBooking}>
            {t("Kembali ke beranda", "Back to home")}
          </Link>
          <Link
            className="link-button primary"
            to="/status"
            state={{ code: submitted.code, email: submitted.email }}
          >
            {t("Cek status", "Check status")} <ArrowRight size={17} />
          </Link>
        </div>
      </div>
    );
  return (
    <section
      aria-labelledby="registration-title"
      className={`booking-container${embedded ? " booking-embedded" : ""}`}
    >
      {!embedded && (
        <Link
          className="text-link"
          to={step === 0 ? "/" : step === 1 ? "/booking" : "/booking/visitors"}
        >
          <ArrowLeft size={16} />
          {step === 0
            ? t("Kembali ke beranda", "Back to home")
            : t("Langkah sebelumnya", "Previous step")}
        </Link>
      )}
      <div className="booking-page-intro">
        <IntroHeading id="registration-title">
          <span className="booking-mobile-copy">
            {embedded ? (
              t("Rencanakan kunjungan Anda.", "Plan your visit.")
            ) : (
              <>
                {t("Rencanakan", "Plan")}
                <br />
                {t("kunjungan Anda.", "your visit.")}
              </>
            )}
          </span>
          <span className="booking-desktop-copy">{stepTitles[step]}</span>
        </IntroHeading>
        <p className="booking-mobile-copy">
          {t(
            "Langkah sederhana untuk pertemuan yang lebih terarah.",
            "Simple steps to a better planned meeting.",
          )}
        </p>
        <p className="booking-desktop-copy">{stepDescriptions[step]}</p>
      </div>
      <ol className="booking-progress">
        {[
          t("Detail kunjungan", "Visit details"),
          t("Data pengunjung", "Visitor details"),
          t("Tinjau & kirim", "Review & submit"),
        ].map((s, i) => (
          <li
            key={s}
            aria-current={step === i ? "step" : undefined}
            className={i <= step ? "current" : ""}
          >
            <span>{i < step ? <Check size={15} /> : i + 1}</span>
            {i < step ? (
              <>
                <strong className="booking-mobile-copy">{s}</strong>
                <Link
                  className="booking-desktop-copy booking-progress-link"
                  to={stepPaths[i]}
                  aria-label={`${t("Kembali ke", "Back to")} ${s.toLowerCase()}`}
                >
                  {s}
                </Link>
              </>
            ) : (
              <strong>{s}</strong>
            )}
          </li>
        ))}
      </ol>
      {unresolved && (
        <div ref={recoveryHeading}>
          <SubmissionRecovery
            headingLevel={headingLevel === 1 ? 2 : 3}
            email={recoveryEmail}
            onEmailChange={setRecoveryEmail}
            checking={phase === "checking"}
            onRecover={recover}
            error={error}
          />
        </div>
      )}
      {phase === "retrying" && (
        <p role="status" className="hint section-gap">
          {t(
            "Koneksi terputus. Mencoba sekali lagi dengan referensi pengiriman yang sama…",
            "The connection was interrupted. Retrying once with the same submission reference…",
          )}
        </p>
      )}
      {submissionNotice && (
        <p role="status" className="hint section-gap">
          {submissionNotice}
        </p>
      )}
      <ReferenceDataNotice
        loading={catalogs.isPending}
        error={catalogs.error}
        onRetry={() => void catalogs.refetch()}
      />
      <div className="booking-layout">
        <form
          className="panel booking-form"
          onSubmit={(event) => {
            if (step < 2) {
              event.preventDefault();
              void next();
              return;
            }
            if (unresolved) {
              event.preventDefault();
              recoveryHeading.current
                ?.querySelector<HTMLElement>("#submission-recovery-title")
                ?.focus();
              return;
            }
            void f.handleSubmit(submit, (errors) => {
              const firstVisit = visitKeys.find((key) => errors[key]);
              const firstVisitor = visitorKeys.find((key) => errors[key]);
              const first = firstVisit || firstVisitor || "ack";
              const owner = firstVisit ? 0 : firstVisitor ? 1 : 2;
              if (owner === step) {
                f.setFocus(first);
                return;
              }
              focusAfterNavigation.current = first;
              navigate(
                owner === 0 ? "/booking" : owner === 1 ? "/booking/visitors" : "/booking/review",
              );
            })(event);
          }}
        >
          <fieldset
            className="booking-fields"
            disabled={pending}
            aria-label={t("Isian kunjungan", "Visit fields")}
          >
            {step === 0 ? (
              <>
                <FormHeading>
                  {t("Apa rencana kunjungan Anda?", "What is your visit plan?")}
                </FormHeading>
                <p className="form-subtitle">
                  {t(
                    "Pilih tujuan dan waktu mulai. Durasi ditetapkan petugas.",
                    "Choose who to meet and when to start. Staff will set the duration.",
                  )}
                </p>
                <RequiredFieldsHint />
                <div className="form-grid" key="visit-details">
                  <div className="field span-2">
                    <label htmlFor="officerId">
                      {t("Pejabat atau unit tujuan", "Official or unit to visit")} <RequiredMark />
                    </label>
                    <Select
                      id="officerId"
                      required
                      {...f.register("officerId", {
                        onChange: () => {
                          if (f.formState.errors.officerId) void f.trigger("officerId");
                        },
                      })}
                      aria-invalid={!!f.formState.errors.officerId}
                      aria-describedby={
                        f.formState.errors.officerId ? "officerId-error" : undefined
                      }
                    >
                      <option value="">{t("Pilih tujuan kunjungan", "Choose who to visit")}</option>
                      {officers
                        .filter((officer) => officer.active)
                        .map((o) => (
                          <option key={o.id} value={o.id}>
                            {o.name}
                          </option>
                        ))}
                    </Select>
                    {f.formState.errors.officerId && (
                      <FieldMessage tone="error" id="officerId-error">
                        {localizeMessage(f.formState.errors.officerId.message, language)}
                      </FieldMessage>
                    )}
                  </div>
                  <Controller
                    name="date"
                    control={f.control}
                    render={({ field, fieldState }) => (
                      <DatePicker
                        {...field}
                        onChange={(value) => {
                          field.onChange(value);
                          if (fieldState.error) void f.trigger("date");
                        }}
                        id="date"
                        label={t("Tanggal kunjungan", "Visit date")}
                        min={today()}
                        required
                        error={localizeMessage(fieldState.error?.message, language)}
                      />
                    )}
                  />
                  <Controller
                    name="time"
                    control={f.control}
                    render={({ field, fieldState }) => (
                      <TimePicker
                        {...field}
                        id="time"
                        label={t("Waktu mulai · WITA", "Start time · WITA")}
                        onChange={(value) => {
                          field.onChange(value);
                          if (fieldState.error) void f.trigger("time");
                        }}
                        required
                        disabled={pending}
                        error={localizeMessage(fieldState.error?.message, language)}
                      />
                    )}
                  />
                  <div className="field span-2">
                    <label htmlFor="purpose">
                      {t("Tujuan kunjungan", "Purpose of visit")} <RequiredMark />
                    </label>
                    <textarea
                      id="purpose"
                      required
                      rows={4}
                      placeholder={t(
                        "Ceritakan tujuan pertemuan Anda…",
                        "Tell us the purpose of your meeting…",
                      )}
                      {...f.register("purpose", {
                        onChange: () => {
                          if (f.formState.errors.purpose) void f.trigger("purpose");
                        },
                      })}
                      aria-invalid={!!f.formState.errors.purpose}
                      aria-describedby={f.formState.errors.purpose ? "purpose-error" : undefined}
                    />
                    {f.formState.errors.purpose && (
                      <FieldMessage tone="error" id="purpose-error">
                        {localizeMessage(f.formState.errors.purpose.message, language)}
                      </FieldMessage>
                    )}
                  </div>
                </div>
                <div className="section-gap">
                  <p id="nota-dinas-question" className="field-label">
                    {t("Apakah Anda memiliki Nota Dinas?", "Do you have an official memo?")}
                  </p>
                  <label className="check-label" htmlFor="hasNotaDinas">
                    <input
                      id="hasNotaDinas"
                      type="checkbox"
                      checked={hasNotaDinas}
                      aria-describedby="nota-dinas-question"
                      aria-controls={hasNotaDinas ? "nota-dinas-upload" : undefined}
                      onChange={(event) => {
                        setHasNotaDinas(event.target.checked);
                        if (!event.target.checked) {
                          setND(null);
                          setFileError(null);
                        }
                      }}
                    />
                    <span>
                      {t(
                        "Ya, saya memiliki Nota Dinas atau Permohonan Kunjungan Resmi.",
                        "Yes, I have an official memo or visit request.",
                      )}
                    </span>
                  </label>
                </div>
                {hasNotaDinas && (
                  <div id="nota-dinas-upload" className="section-gap">
                    <UploadField
                      id="nd"
                      label={t("Nota Dinas atau Permohonan Kunjungan Resmi", "Official memo or visit request")}
                      accept={notaDinasType}
                      file={nd}
                      error={fileError && localizeMessage(fileError, language)}
                      hint={t("PDF · Maksimal 5 MB", "PDF · Up to 5 MB")}
                      choose={t("Pilih dokumen", "Choose document")}
                      remove={t("Hapus dokumen", "Remove document")}
                      onFileChange={(file) => {
                        const message = validateND(file);
                        setFileError(message);
                        setND(message ? null : file);
                      }}
                    />
                  </div>
                )}
                <div className="hint section-gap">
                  <CalendarDays size={18} />
                  {t(
                    "Anda cukup memilih waktu mulai. Petugas akan mengonfirmasi durasi dan ruangan saat menyetujui kunjungan.",
                    "Choose a start time. Staff will confirm the duration and room when approving your visit.",
                  )}
                </div>
              </>
            ) : step === 1 ? (
              <>
                <FormHeading>{t("Siapa yang akan berkunjung?", "Who will visit?")}</FormHeading>
                <p className="form-subtitle">
                  {t(
                    "Lengkapi data perwakilan dan nama anggota rombongan.",
                    "Provide the representative's details and the names of other visitors.",
                  )}
                </p>
                <RequiredFieldsHint />
                <div className="form-grid" key="visitor-details">
                  {fields(
                    "name",
                    t("Nama lengkap perwakilan", "Representative's full name"),
                    "text",
                    {
                      autoComplete: "name",
                      placeholder: t("Masukkan nama lengkap", "Enter full name"),
                      required: true,
                    },
                  )}
                  {fields(
                    "organization",
                    t("Instansi atau asal", "Organization or affiliation"),
                    "text",
                    {
                      autoComplete: "organization",
                      placeholder: t(
                        "Masukkan instansi atau asal",
                        "Enter organization or affiliation",
                      ),
                      required: true,
                    },
                  )}
                  <Controller
                    name="email"
                    control={f.control}
                    render={({ field, fieldState }) => (
                      <div className="field">
                        <label htmlFor="email">
                          {t("Email untuk konfirmasi", "Email for confirmation")} <RequiredMark />
                        </label>
                        <EmailInput
                          id="email"
                          name={field.name}
                          ref={field.ref}
                          value={field.value}
                          onValueChange={field.onChange}
                          onDomainPick={() => {
                            void f.trigger("email");
                          }}
                          onBlur={field.onBlur}
                          required
                          aria-invalid={!!fieldState.error}
                          aria-describedby={fieldState.error ? "email-error" : undefined}
                        />
                        {fieldState.error && (
                          <FieldMessage id="email-error" tone="error">
                            {localizeMessage(fieldState.error.message, language)}
                          </FieldMessage>
                        )}
                      </div>
                    )}
                  />
                  {fields("phone", t("Nomor telepon", "Phone number"), "tel", {
                    autoComplete: "tel",
                    inputMode: "numeric",
                    pattern: "[0-9]{8,20}",
                    placeholder: t("Contoh: 081234567890", "Example: 081234567890"),
                    required: true,
                  })}
                  <div className="field span-2">
                    <label htmlFor="identityType">
                      {t("Identitas yang dibawa saat kedatangan", "ID to bring on arrival")}{" "}
                      <RequiredMark />
                    </label>
                    <Select id="identityType" {...f.register("identityType")} required aria-describedby="identityType-hint">
                      <option>KTP</option>
                      <option value="Paspor">{t("Paspor", "Passport")}</option>
                      <option>SIM</option>
                    </Select>
                    <FieldMessage id="identityType-hint">
                      {t(
                        "Bawa identitas asli untuk pemeriksaan di resepsionis. Nomor dan foto identitas tidak diminta.",
                        "Bring your original ID for reception verification. An ID number or photo is not requested.",
                      )}
                    </FieldMessage>
                  </div>
                </div>
                <div className="form-stack form-group-gap">
                  <div className="roster-heading">
                    <div className="form-stack">
                      <GroupHeading className="roster-title">
                        {t("Rombongan pengunjung", "Visitor group")}
                      </GroupHeading>
                      <FieldMessage>
                        {array.fields.length + 1}{" "}
                        {t(
                          "orang, termasuk perwakilan. Maksimal 20 orang.",
                          "people including the representative. Up to 20 people.",
                        )}
                      </FieldMessage>
                    </div>
                    <Users size={21} />
                  </div>
                  {f.formState.errors.visitors?.message && (
                    <FieldMessage tone="error">
                      {localizeMessage(f.formState.errors.visitors.message, language)}
                    </FieldMessage>
                  )}
                  {array.fields.map((item, i) => (
                    <div className="form-roster-row" key={item.id}>
                      <div className="field">
                        <label htmlFor={`visitor-${i}`}>
                          {t("Nama pengunjung", "Visitor name")} {i + 2} <RequiredMark />
                        </label>
                        <input
                          id={`visitor-${i}`}
                          required
                          placeholder={t("Masukkan nama lengkap", "Enter full name")}
                          {...f.register(`visitors.${i}.name`)}
                          aria-invalid={!!f.formState.errors.visitors?.[i]?.name}
                          aria-describedby={
                            f.formState.errors.visitors?.[i]?.name ? `visitor-${i}-error` : undefined
                          }
                        />
                        {f.formState.errors.visitors?.[i]?.name && (
                          <FieldMessage tone="error" id={`visitor-${i}-error`}>
                            {localizeMessage(f.formState.errors.visitors[i]?.name?.message, language)}
                          </FieldMessage>
                        )}
                      </div>
                      <Button
                        type="button"
                        kind="ghost"
                        size="icon"
                        aria-label={`${t("Hapus pengunjung", "Remove visitor")} ${i + 2}`}
                        onClick={() => array.remove(i)}
                      >
                        <Trash2 size={18} />
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    kind="secondary"
                    onClick={() => array.append({ name: "" })}
                    disabled={array.fields.length >= 19}
                  >
                    <Plus size={16} />
                    {t("Tambah pengunjung", "Add visitor")}
                  </Button>
                </div>
                <div className="form-group-gap">
                  <UploadField
                    id="visitorPhoto"
                    label={t("Foto wajah perwakilan", "Representative's face photo")}
                    required
                    accept={visitorPhotoTypes.join(",")}
                    file={visitorPhoto}
                    error={photoError && localizeMessage(photoError, language)}
                    hint={t("JPEG, PNG · Maksimal 5 MB", "JPEG, PNG · Up to 5 MB")}
                    choose={t("Pilih foto wajah", "Choose face photo")}
                    remove={t("Hapus foto", "Remove photo")}
                    photoPreviewLabel={t("Pratinjau foto pengunjung", "Visitor photo preview")}
                    onFileChange={(file) => {
                      const message = validateVisitorPhoto(file);
                      setPhotoError(message);
                      setVisitorPhoto(message ? null : file);
                    }}
                  />
                </div>
              </>
            ) : (
              <>
                <FormHeading>{t("Periksa sekali lagi.", "Review once more.")}</FormHeading>
                <p className="form-subtitle">
                  {t(
                    "Pastikan informasi sudah benar sebelum dikirim.",
                    "Make sure everything is correct before submitting.",
                  )}
                </p>
                <dl className="review-list">
                  {[
                    [t("Tujuan", "Destination"), selectedOfficer?.name],
                    [
                      t("Jadwal", "Schedule"),
                      `${values.date ? dateLabel(values.date, language) : "—"} · ${values.time || "—"} WITA`,
                    ],
                    [t("Perwakilan", "Representative"), `${values.name} · ${values.organization}`],
                    [t("Kontak", "Contact"), `${values.email} · ${values.phone}`],
                    [
                      t("Pengunjung", "Visitors"),
                      [values.name, ...(values.visitors || []).map((v) => v.name)].join(", "),
                    ],
                    [
                      t("Identitas asli", "Original ID"),
                      values.identityType === "Paspor"
                        ? t("Paspor", "Passport")
                        : values.identityType,
                    ],
                    [t("Keperluan", "Purpose"), values.purpose],
                    [
                      t("Foto wajah perwakilan", "Representative's face photo"),
                      visitorPhoto?.name || t("Tidak dilampirkan", "Not attached"),
                    ],
                    [
                      t(
                        "Nota Dinas atau Permohonan Kunjungan Resmi",
                        "Official memo or visit request",
                      ),
                      nd?.name || t("Tidak dilampirkan", "Not attached"),
                    ],
                  ].map(([label, value]) => (
                    <div key={label}>
                      <dt>{label}</dt>
                      <dd>{value}</dd>
                    </div>
                  ))}
                </dl>
                <label className="check-label section-gap">
                  <input
                    type="checkbox"
                    {...f.register("ack")}
                    required
                    aria-invalid={!!f.formState.errors.ack}
                    aria-describedby={f.formState.errors.ack ? "ack-error" : undefined}
                  />
                  <span>
                    {t("Saya telah membaca", "I have read the")}{" "}
                    <Button
                      type="button"
                      kind="inline-link"
                      onClick={(event) => {
                        privacyTrigger.current = event.currentTarget;
                        setPrivacy(true);
                      }}
                    >
                      {t("pemberitahuan privasi", "privacy notice")}
                    </Button>{" "}
                    {t(
                      "dan memastikan data rombongan benar. Data digunakan untuk pengelolaan dan verifikasi kunjungan.",
                      "and confirm that the visitor details are correct. The data is used to manage and verify the visit.",
                    )}{" "}
                    <RequiredMark />
                  </span>
                </label>
                {f.formState.errors.ack && (
                  <FieldMessage tone="error" id="ack-error">
                    {localizeMessage(f.formState.errors.ack.message, language)}
                  </FieldMessage>
                )}
              </>
            )}
          </fieldset>
          <ErrorNotice error={unresolved ? null : error} />
          <div className="form-actions">
            {step < 2 ? (
              <Button
                type="submit"
                formNoValidate
                disabled={pending}
              >
                <span className="booking-mobile-copy">{t("Lanjutkan", "Continue")}</span>
                <span className="booking-desktop-copy">
                  {step === 0
                    ? t("Lanjut ke data pengunjung", "Continue to visitor details")
                    : t("Tinjau permohonan", "Review request")}
                </span>
                <ArrowRight size={17} />
              </Button>
            ) : (
              <Button type="submit" disabled={pending || unresolved}>
                {pending
                  ? t("Mengirim permohonan…", "Submitting request…")
                  : t("Kirim permohonan", "Submit request")}
                <ArrowUpIcon />
              </Button>
            )}
          </div>
        </form>
        <aside className="booking-summary glass">
          <span className="summary-icon">
            <FileText size={23} />
          </span>
          <FormHeading>{t("Rencana kunjungan", "Visit plan")}</FormHeading>
          <p>
            {step === 0
              ? t("Isi tujuan dan jadwal terlebih dahulu.", "Choose who to meet and when first.")
              : step === 1
                ? t(
                    "Pastikan kontak dapat dihubungi untuk konfirmasi.",
                    "Make sure we can reach your contact for confirmation.",
                  )
                : t("Periksa ringkasan sebelum mengirim.", "Review the summary before submitting.")}
          </p>
          <dl>
            <div>
              <dt>{t("Pejabat yang dituju", "Person to visit")}</dt>
              <dd>{selectedOfficer?.name || t("Belum dipilih", "Not selected")}</dd>
            </div>
            <div>
              <dt>{t("Tujuan kunjungan", "Purpose of visit")}</dt>
              <dd>{values.purpose.trim() || t("Belum diisi", "Not entered")}</dd>
            </div>
            <div>
              <dt>{t("Tanggal & waktu", "Date & time")}</dt>
              <dd>
                {values.date
                  ? dateLabel(values.date, language)
                  : t("Belum dipilih", "Not selected")}
                <small>
                  {values.time
                    ? `${values.time} WITA`
                    : t("Waktu belum dipilih", "Time not selected")}
                </small>
              </dd>
            </div>
            <div>
              <dt>{t("Jumlah pengunjung", "Number of visitors")}</dt>
              <dd>
                {1 + (values.visitors?.length || 0)} {t("orang", "people")}
              </dd>
            </div>
          </dl>
          <div className="privacy-note">
            <ShieldCheck size={20} />
            <p>
              {t("Data seperlunya.", "Only necessary data.")}
              <br />
              {t("Identitas asli diperiksa saat kedatangan.", "Original ID is checked on arrival.")}
            </p>
          </div>
        </aside>
      </div>
      <Modal
        returnFocusTo={privacyTrigger.current}
        open={privacy}
        onClose={() => setPrivacy(false)}
        title={t("Pemberitahuan privasi", "Privacy notice")}
        description={t(
          "Data digunakan untuk mengelola dan memverifikasi kunjungan.",
          "Data is used to manage and verify visits.",
        )}
      >
        <p>
          <PrivacyNotice />
        </p>
        <div className="form-actions">
          <Button type="button" onClick={() => setPrivacy(false)}>
            {t("Kembali ke formulir", "Back to form")}
          </Button>
        </div>
      </Modal>
    </section>
  );
}
function ArrowUpIcon() {
  return <ArrowRight size={17} />;
}
