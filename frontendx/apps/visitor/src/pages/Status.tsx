import { ApiError, preview, repository, visitTiming, type PublicBooking } from "@vms/data";
import { emailSchema } from "@vms/data/validation";
import { FieldMessage, Button, EmailInput, ErrorNotice, Modal, RequiredFieldsHint, RequiredMark } from "@vms/ui";
import { InputOTP } from "@vms/ui/input-otp";
import { localizeMessage } from "@vms/ui/language";
import { ArrowLeft, ArrowRight, Ban, RefreshCw, ShieldCheck } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Link, useLocation } from "react-router";
import PrivacyNotice from "../features/PrivacyNotice";
import PublicVisitHistory from "../features/PublicVisitHistory";
import { useCancellationWindow } from "../features/useCancellationWindow";
import VisitTicket from "../features/VisitTicket";
import { useLanguage } from "../i18n";
export default function Status({
  embedded = false,
  headingLevel = 1,
}: {
  embedded?: boolean;
  headingLevel?: 1 | 2;
}) {
  const { language, t } = useLanguage();
  const IntroHeading = headingLevel === 1 ? "h1" : "h2";
  const SectionHeading = headingLevel === 1 ? "h2" : "h3";
  const state = useLocation().state as { code?: string; email?: string } | null;
  const [code, setCode] = useState(state?.code || "");
  const [email, setEmail] = useState(state?.email || "");
  const [emailError, setEmailError] = useState("");
  const emailInput = useRef<HTMLInputElement>(null);
  const [booking, setBooking] = useState<PublicBooking | null>(null);
  const [error, setError] = useState<unknown>();
  const [rejectedLookup, setRejectedLookup] = useState<{ code: string; email: string }>();
  const [busy, setBusy] = useState(false);
  const [cancel, setCancel] = useState(false);
  const dialogTrigger = useRef<HTMLElement | null>(null);
  const resultHeading = useRef<HTMLHeadingElement>(null);
  const resultToolbar = useRef<HTMLDivElement>(null);
  const focusResult = useRef(false);
  const focusLookup = useRef(false);
  const restoreUpdateFocus = useRef(false);
  const cancellationAvailable = useCancellationWindow(booking);
  const codeRejected = rejectedLookup?.code === code && rejectedLookup.email === email;
  function clearLookupRejection() {
    if (rejectedLookup) {
      setRejectedLookup(undefined);
      setError(null);
    }
  }
  useEffect(() => {
    if (booking && focusResult.current) {
      focusResult.current = false;
      resultHeading.current?.focus();
    } else if (!booking && focusLookup.current) {
      focusLookup.current = false;
      document.getElementById("status-code")?.focus();
    }
  }, [booking]);
  useEffect(() => {
    if (busy || !restoreUpdateFocus.current) return;
    restoreUpdateFocus.current = false;
    if (document.activeElement === document.body)
      resultToolbar.current?.querySelector("button")?.focus();
  }, [busy]);
  async function load(focusOnResult = false) {
    if (busy) return;
    const parsedEmail = emailSchema.safeParse(email);
    if (!parsedEmail.success) {
      setEmailError(parsedEmail.error.issues[0].message);
      emailInput.current?.focus();
      return;
    }
    restoreUpdateFocus.current =
      !focusOnResult && document.activeElement === resultToolbar.current?.querySelector("button");
    setBusy(true);
    setError(null);
    setRejectedLookup(undefined);
    try {
      const result = await repository.lookup(code, parsedEmail.data);
      focusResult.current = focusOnResult;
      setBooking(result);
    } catch (err) {
      if (
        focusOnResult &&
        err instanceof ApiError &&
        (err.status === 404 || (preview && err.status === 401))
      )
        setRejectedLookup({ code, email });
      setError(err);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className={`status-container${embedded ? " status-embedded" : ""}`}>
      {!embedded && (
        <Link to="/" className="text-link">
          <ArrowLeft size={16} />
          {t("Kembali ke beranda", "Back to home")}
        </Link>
      )}
      <div className="booking-page-intro">
        <IntroHeading>
          {booking
            ? t("Kunjungan Anda.", "Your visit.")
            : t("Sudah punya rencana?", "Already have a plan?")}
          <br />
          {booking
            ? t("Semua ada di sini.", "Find it all here.")
            : t("Mari periksa statusnya.", "Check its status.")}
        </IntroHeading>
        <p>
          {booking
            ? t(
                "Periksa status dan rincian kunjungan Anda di bawah ini.",
                "Review your visit status and details below.",
              )
            : t(
                "Masukkan email pendaftaran dan kode kunjungan untuk melihat detail kunjungan.",
                "Enter your registration email and visit code to see the visit details.",
              )}
        </p>
      </div>
      <div className="panel status-panel">
        {booking ? (
          <>
            <div className="status-heading" ref={resultToolbar}>
              <SectionHeading ref={resultHeading} tabIndex={-1} className="status-heading__label">
                {t("Rincian kunjungan", "Visit summary")}
              </SectionHeading>
              <Button kind="ghost" disabled={busy} onClick={() => void load()}>
                <RefreshCw size={16} />
                {busy ? t("Memperbarui…", "Updating…") : t("Perbarui", "Update")}
              </Button>
            </div>
            <VisitTicket booking={booking} headingLevel={headingLevel === 1 ? 3 : 4} />
            {booking.rescheduled && (
              <p className="visit-ticket__rescheduled">
                {t(
                  "Jadwal kunjungan telah diubah. Periksa kembali tanggal dan waktu di atas.",
                  "This visit was rescheduled. Check the date and time above.",
                )}
              </p>
            )}
            {booking.status === "rejected" && booking.rejectionReason && (
              <section
                className="decision-note section-gap"
                aria-label={t("Alasan penolakan", "Reason for rejection")}
              >
                <SectionHeading>{t("Alasan penolakan", "Reason for rejection")}</SectionHeading>
                <p>{booking.rejectionReason}</p>
              </section>
            )}
            {booking.status === "cancelled" && booking.cancellationReason && (
              <section
                className="decision-note section-gap"
                aria-label={t("Alasan pembatalan", "Reason for cancellation")}
              >
                <SectionHeading>{t("Alasan pembatalan", "Reason for cancellation")}</SectionHeading>
                <p>{booking.cancellationReason}</p>
              </section>
            )}
            <div className="hint section-gap">
              <ShieldCheck size={18} />
              {visitTiming(booking).displayStatus === "missed"
                ? t(
                    "Waktu kedatangan telah berakhir. Buat permohonan baru untuk kunjungan berikutnya.",
                    "The arrival window has ended. Submit a new request for another visit.",
                  )
                : booking.status === "pending"
                  ? t(
                      "Permohonan sedang ditinjau. Tunggu persetujuan sebelum datang.",
                      "Your request is under review. Wait for approval before arriving.",
                    )
                  : booking.status === "approved"
                    ? t(
                        "Kunjungan disetujui. Bawa identitas asli dan kode kunjungan atau QR saat kedatangan.",
                        "Your visit is approved. Bring your original ID and visit code or QR code when you arrive.",
                      )
                    : booking.status === "checked_in"
                      ? t(
                          "Kunjungan sedang berlangsung. Sampaikan kode kunjungan kepada petugas saat keluar.",
                          "Your visit is in progress. Give your visit code to staff when leaving.",
                        )
                      : t(
                          "Lihat riwayat untuk informasi keputusan dan perubahan kunjungan.",
                          "See the history for decisions and visit changes.",
                        )}
            </div>
            <PublicVisitHistory
              events={booking.history}
              headingLevel={headingLevel === 1 ? 2 : 3}
            />
            {["pending", "approved"].includes(booking.status) && !cancellationAvailable && (
              <p id="cancel-cutoff" className="hint section-gap" role="status">
                {t(
                  "Pembatalan mandiri ditutup pada waktu mulai kunjungan. Hubungi petugas untuk bantuan.",
                  "Self-service cancellation closes at the visit start time. Contact staff for help.",
                )}
              </p>
            )}
            <div className="toolbar">
              {["pending", "approved"].includes(booking.status) && (
                <Button
                  kind="secondary"
                  disabled={!cancellationAvailable}
                  aria-describedby={!cancellationAvailable ? "cancel-cutoff" : undefined}
                  onClick={(event) => {
                    dialogTrigger.current = event.currentTarget;
                    setCancel(true);
                  }}
                >
                  <Ban size={16} />
                  {t("Batalkan kunjungan", "Cancel visit")}
                </Button>
              )}
              <Button
                kind="ghost"
                onClick={() => {
                  focusLookup.current = true;
                  setBooking(null);
                  setError(null);
                }}
              >
                {t("Periksa kunjungan lain", "Check another visit")}
              </Button>
            </div>
          </>
        ) : (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (!busy) void load(true);
            }}
          >
            <>
              <SectionHeading>{t("Temukan kunjungan Anda", "Find your visit")}</SectionHeading>
              <p className="form-subtitle">
                {t(
                  "Masukkan kode kunjungan dan email saat pendaftaran.",
                  "Enter your visit code and registration email.",
                )}
              </p>
              <RequiredFieldsHint />
              <div className="field status-code-field">
                <label htmlFor="status-code">
                  {t("Kode kunjungan", "Visit code")} <RequiredMark />
                </label>
                <InputOTP
                  className="status-code-input"
                  id="status-code"
                  required
                  readOnly={busy}
                  value={code}
                  onValueChange={(next) => {
                    setCode(next);
                    clearLookupRejection();
                  }}
                  aria-label={t("8 digit kode kunjungan", "8-digit visit code")}
                  aria-invalid={codeRejected}
                  aria-describedby={codeRejected ? "status-lookup-error" : undefined}
                />
              </div>
              <div className="field section-gap status-email-field">
                <label htmlFor="status-email">
                  {t("Email pendaftaran", "Registration email")} <RequiredMark />
                </label>
                <EmailInput
                  id="status-email"
                  ref={emailInput}
                  required
                  value={email}
                  onValueChange={(nextEmail) => {
                    setEmail(nextEmail);
                    clearLookupRejection();
                    if (emailError && emailSchema.safeParse(nextEmail).success) setEmailError("");
                  }}
                  aria-invalid={!!emailError}
                  aria-describedby={emailError ? "status-email-error" : undefined}
                />
                {emailError && (
                  <FieldMessage tone="error" id="status-email-error">
                    {localizeMessage(emailError, language)}
                  </FieldMessage>
                )}
              </div>
            </>
            <div id="status-lookup-error">
              <ErrorNotice error={error} />
            </div>
            <div className="form-actions">
              <Button disabled={busy}>
                {busy ? t("Memeriksa…", "Checking…") : t("Lihat kunjungan", "View visit")}
                <ArrowRight size={17} />
              </Button>
            </div>
          </form>
        )}
        {preview && (
          <div className="demo-help">
            {t("Contoh:", "Example:")} 88000002 · visitor@example.test
          </div>
        )}
        <ErrorNotice error={booking ? error : null} />
      </div>
      <div className="privacy-copy" id="privacy">
        <ShieldCheck size={24} />
        <div>
          <SectionHeading>
            {t("Privasi Anda, bagian dari pelayanan kami.", "Your privacy is part of our service.")}
          </SectionHeading>
          <p>
            <PrivacyNotice />
          </p>
        </div>
      </div>
      <Modal
        returnFocusTo={
          cancellationAvailable
            ? dialogTrigger.current
            : resultToolbar.current?.querySelector("button")
        }
        open={cancel}
        onClose={() => !busy && setCancel(false)}
        title={t("Batalkan kunjungan?", "Cancel visit?")}
        description={t(
          "Kunjungan yang dibatalkan tidak dapat digunakan untuk masuk.",
          "A canceled visit cannot be used for entry.",
        )}
      >
        <ErrorNotice error={error} />
        <div className="form-actions">
          <Button kind="secondary" onClick={() => setCancel(false)}>
            {t("Tetap simpan kunjungan", "Keep visit")}
          </Button>
          <Button
            kind="danger"
            disabled={busy || !cancellationAvailable}
            onClick={async () => {
              if (!booking || busy || !cancellationAvailable) return;
              setBusy(true);
              setError(null);
              try {
                setBooking(
                  await repository.cancel(booking.id, {
                    code: booking.code,
                    email: email.trim(),
                    version: booking.version,
                  }),
                );
                setCancel(false);
              } catch (err) {
                setError(err);
                if (err instanceof ApiError && err.status === 409) {
                  try {
                    setBooking(await repository.lookup(booking.code, email.trim()));
                  } catch {
                    /* Preserve the cancellation error if refresh also fails. */
                  }
                }
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy ? t("Membatalkan…", "Canceling…") : t("Ya, batalkan", "Yes, cancel")}
          </Button>
        </div>
      </Modal>
    </div>
  );
}
