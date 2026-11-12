import { Button, Loading } from "./shell";
export { AppBoundary, Brand, Button, Loading, PreviewNotice, RouteFocus } from "./shell";
import {
  useEffect,
  useRef,
  useState,
  forwardRef,
  type InputHTMLAttributes,
  type ReactNode,
} from "react";
import * as Dialog from "@radix-ui/react-dialog";
import {
  X,
  RefreshCw,
  TriangleAlert,
  Check,
  Clock3,
  CircleX,
  Ban,
  LogIn,
  LogOut,
  ShieldCheck,
} from "lucide-react";
import {
  labels,
  preview,
  dateLabel,
  emailDomains,
  splitEmail,
  joinEmail,
  isRestricted,
  bookingName,
  partyLabel,
  ApiError,
  type Booking,
  type Status,
  type StaffBooking,
  type DisplayStatus,
  type VisitTiming,
} from "@vms/data";
import { RestrictedIdentity } from "./restricted-identity";
import { localizeMessage, useUiLanguage } from "./language";
export function RequiredMark() {
  return (
    <span className="required-mark" aria-hidden="true">
      *
    </span>
  );
}
export function RequiredFieldsHint() {
  const { t } = useUiLanguage();
  return (
    <p className="required-fields-hint">
      {t("Kolom bertanda", "Fields marked")} <RequiredMark />
      <span className="visually-hidden">{t("bintang", "asterisk")}</span>{" "}
      {t("wajib diisi.", "are required.")}
    </p>
  );
}
type EmailInputProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  "type" | "value" | "onChange"
> & {
  id: string;
  value: string;
  onValueChange: (value: string) => void;
  onDomainPick?: () => void;
};
export const EmailInput = forwardRef<HTMLInputElement, EmailInputProps>(function EmailInput(
  { id, value, onValueChange, onDomainPick, ...props },
  ref,
) {
  const { t } = useUiLanguage();
  const { username, domain } = splitEmail(value);
  const [customOpen, setCustomOpen] = useState(false);
  const customDomainInput = useRef<HTMLInputElement>(null);
  const domainSelect = useRef<HTMLSelectElement>(null);
  const focusDomainAfterSwitch = useRef(false);
  const knownDomain = emailDomains.some((item) => item === domain.toLowerCase());
  const custom = customOpen || (!!domain && !knownDomain);
  const hintId = `${id}-hint`;
  const describedBy = [hintId, props["aria-describedby"]].filter(Boolean).join(" ");
  useEffect(() => {
    if (!focusDomainAfterSwitch.current) return;
    focusDomainAfterSwitch.current = false;
    (custom ? customDomainInput.current : domainSelect.current)?.focus();
  }, [custom]);
  function updateUsername(input: string) {
    if (input.includes("@")) {
      // Accept a complete pasted/autofilled address without appending another suffix.
      const address = splitEmail(input.trim());
      setCustomOpen(
        !!address.domain && !emailDomains.some((item) => item === address.domain.toLowerCase()),
      );
      onValueChange(joinEmail(address.username, address.domain));
    } else {
      onValueChange(joinEmail(input, domain));
    }
  }
  return (
    <div className="email-input">
      <div className="email-address">
        <div className="email-username">
          <input
            {...props}
            id={id}
            ref={ref}
            type="text"
            inputMode="email"
            autoComplete="email"
            autoCapitalize="none"
            autoCorrect="off"
            spellCheck={false}
            placeholder={t("nama.anda", "your.name")}
            pattern="[^@\s]+"
            value={username}
            onChange={(event) => updateUsername(event.target.value)}
            aria-describedby={describedBy}
          />
        </div>
        <span className="email-at" aria-hidden="true">
          @
        </span>
        <div className="email-suffix">
          {custom ? (
            <div className="email-custom-inline">
              <input
                id={`${id}-custom-domain`}
                ref={customDomainInput}
                type="text"
                inputMode="url"
                autoCapitalize="none"
                autoCorrect="off"
                spellCheck={false}
                autoComplete="off"
                placeholder="instansi.go.id"
                aria-label={t("Domain email lain", "Other email domain")}
                required={props.required}
                disabled={props.disabled}
                readOnly={props.readOnly}
                pattern="[^@\s]+\.[^@\s]+"
                value={domain}
                aria-invalid={props["aria-invalid"]}
                aria-describedby={describedBy}
                onChange={(event) => onValueChange(joinEmail(username, event.target.value))}
                onBlur={() => onDomainPick?.()}
              />
              <button
                type="button"
                aria-label={t(
                  "Ubah domain email: pilih dari daftar",
                  "Change email domain: choose from list",
                )}
                disabled={props.disabled || props.readOnly}
                onClick={() => {
                  focusDomainAfterSwitch.current = true;
                  setCustomOpen(false);
                  onValueChange(joinEmail(username, ""));
                }}
              >
                {t("Ubah", "Change")}
              </button>
            </div>
          ) : (
            <select
              id={`${id}-domain`}
              ref={domainSelect}
              required={props.required}
              disabled={props.disabled || props.readOnly}
              aria-invalid={props["aria-invalid"]}
              aria-describedby={describedBy}
              aria-label={t("Domain email", "Email domain")}
              value={custom ? "other" : domain.toLowerCase()}
              onChange={(event) => {
                const nextDomain = event.target.value;
                focusDomainAfterSwitch.current = nextDomain === "other";
                setCustomOpen(nextDomain === "other");
                onValueChange(joinEmail(username, nextDomain === "other" ? "" : nextDomain));
                onDomainPick?.();
              }}
              onBlur={() => onDomainPick?.()}
            >
              <option value="" disabled>
                {t("Pilih domain", "Choose domain")}
              </option>
              {emailDomains.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
              <option value="other">{t("Domain lain…", "Other domain…")}</option>
            </select>
          )}
        </div>
      </div>
      <small id={hintId}>
        {t(
          "Ketik nama sebelum @, lalu pilih domain. Untuk email instansi, pilih Domain lain.",
          "Enter the name before @, then choose a domain. For an organization email, choose Other domain.",
        )}
      </small>
    </div>
  );
});
export function StatusBadge({
  status,
  rescheduled = false,
  timing,
}: {
  status: Status;
  rescheduled?: boolean;
  timing?: VisitTiming;
}) {
  const { language, t } = useUiLanguage();
  const displayStatus: DisplayStatus = timing?.displayStatus || status;
  const Icon = {
    pending: Clock3,
    approved: Check,
    rejected: CircleX,
    cancelled: Ban,
    checked_in: LogIn,
    checked_out: LogOut,
    missed: Clock3,
  }[displayStatus];
  return (
    <span className="status-stack">
      <span className={`badge status-${displayStatus}`}>
        <Icon size={13} />
        {displayStatus === "missed"
          ? t("Terlewat", "Missed")
          : language === "en"
            ? (
                {
                  pending: "Pending",
                  approved: "Approved",
                  rejected: "Rejected",
                  cancelled: "Canceled",
                  checked_in: "Checked in",
                  checked_out: "Checked out",
                } as const
              )[displayStatus]
            : labels[displayStatus]}
      </span>
      {rescheduled && <small className="rescheduled">{t("Jadwal diubah", "Rescheduled")}</small>}
    </span>
  );
}
export function VIP() {
  return (
    <span className="vip">
      <ShieldCheck size={12} />
      VIP
    </span>
  );
}
export function VisitKind({ booking }: { booking: Pick<StaffBooking, "visitKind" | "vip"> }) {
  const { t } = useUiLanguage();
  return isRestricted(booking) ? (
    <span className="vip restricted-badge">
      <ShieldCheck size={12} />
      {t("Terbatas", "Restricted")}
    </span>
  ) : booking.vip ? (
    <VIP />
  ) : null;
}
export function ErrorNotice({ error }: { error: unknown }) {
  const { language } = useUiLanguage();
  return error ? (
    <div className="error-notice" role="alert">
      <TriangleAlert size={18} />
      <span>
        {localizeMessage(error instanceof Error ? error.message : String(error), language)}
      </span>
    </div>
  ) : null;
}
export function QueryState({
  pending,
  error,
  retry,
  hasData = false,
  children,
}: {
  pending: boolean;
  error?: unknown;
  retry: () => unknown;
  hasData?: boolean;
  children: ReactNode;
}) {
  return pending ? (
    <Loading />
  ) : error && (!hasData || (error instanceof ApiError && [401, 403].includes(error.status))) ? (
    <div className="empty">
      <div className="query-error">
        <ErrorNotice error={error} />
        <Button kind="ghost" className="query-error-retry" onClick={() => retry()}>
          <RefreshCw size={16} />
          Coba lagi
        </Button>
      </div>
    </div>
  ) : (
    children
  );
}
export function Empty({
  title = "Belum ada kunjungan",
  text = "Kunjungan yang sesuai akan muncul di sini.",
}: {
  title?: string;
  text?: string;
}) {
  return (
    <div className="empty">
      <Clock3 size={28} />
      <h3>{title}</h3>
      <p>{text}</p>
    </div>
  );
}
export function Modal({
  open,
  onClose,
  title,
  description,
  children,
  returnFocusTo,
  className,
  dismissible = true,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: ReactNode;
  returnFocusTo?: HTMLElement | null;
  className?: string;
  dismissible?: boolean;
}) {
  const { t } = useUiLanguage();
  const returnFocus = useRef<HTMLElement | null>(null);
  return (
    <Dialog.Root open={open} onOpenChange={(v) => !v && dismissible && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className="modal-overlay" />
        <Dialog.Content
          className={className ? `modal ${className}` : "modal"}
          onInteractOutside={(event) => {
            if (!dismissible) event.preventDefault();
          }}
          onEscapeKeyDown={(event) => {
            if (!dismissible) event.preventDefault();
            if (
              event.target instanceof Element &&
              event.target.closest(".date-picker-popover, .time-picker-popover")
            ) {
              event.preventDefault();
            }
          }}
          onOpenAutoFocus={() => {
            returnFocus.current =
              document.activeElement instanceof HTMLElement ? document.activeElement : null;
          }}
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            const target = returnFocusTo || returnFocus.current;
            if (target?.isConnected) target.focus();
            else document.querySelector<HTMLElement>("main")?.focus();
          }}
        >
          <div className="modal-scroll">
            <div className="modal-heading">
              <div>
                <Dialog.Title>{title}</Dialog.Title>
                <Dialog.Description>
                  {description ||
                    t(
                      "Periksa informasi sebelum melanjutkan.",
                      "Review the information before continuing.",
                    )}
                </Dialog.Description>
              </div>
              {dismissible && (
                <Dialog.Close asChild>
                  <Button kind="ghost" aria-label={t("Tutup dialog", "Close dialog")}>
                    <X size={20} />
                  </Button>
                </Dialog.Close>
              )}
            </div>
            {children}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
export function VisitIdentity({
  booking: b,
  showDocument = false,
}: {
  booking: Booking | StaffBooking;
  showDocument?: boolean;
}) {
  const { language, t } = useUiLanguage();
  if (isRestricted(b)) return <RestrictedIdentity booking={b} />;
  return (
    <>
      <div className="identity-title">
        <div className="avatar large">
          {bookingName(b)
            .split(" ")
            .map((n) => n[0])
            .slice(0, 2)
            .join("")}
        </div>
        <div>
          <h3>
            {bookingName(b)} <VisitKind booking={b} />
          </h3>
          <p>{b.organization || t("Instansi belum diisi", "Organization not provided")}</p>
        </div>
      </div>
      <dl className="detail-grid">
        {isRestricted(b) && (
          <div>
            <dt>{t("Kode kunjungan", "Visit code")}</dt>
            <dd className="tabular">{b.code}</dd>
          </div>
        )}
        <div>
          <dt>{t("Tujuan", "Destination")}</dt>
          <dd>{b.host || t("Belum ditentukan", "Not assigned")}</dd>
        </div>
        <div>
          <dt>{t("Jadwal · WITA", "Schedule · WITA")}</dt>
          <dd>
            {dateLabel(b.date, language)} · {b.time} ({b.duration} {t("menit", "min")})
          </dd>
        </div>
        <div>
          <dt>{t("Tempat", "Room")}</dt>
          <dd>{b.room || t("Belum ditentukan", "Not assigned")}</dd>
        </div>
        <div>
          <dt>{t("Jenis identitas", "ID type")}</dt>
          <dd>
            {b.identityType
              ? `${b.identityType === "Paspor" ? t("Paspor", "Passport") : b.identityType} · ${t("periksa dokumen asli", "check original document")}`
              : t(
                  "Belum diisi · koordinasikan dengan Protokol",
                  "Not provided · coordinate with Protocol",
                )}
          </dd>
        </div>
        <div>
          <dt>{t("Rombongan", "Visitors")}</dt>
          <dd>{b.visitors?.length ? b.visitors.join(", ") : partyLabel(b)}</dd>
        </div>
        <div>
          <dt>{t("Keperluan", "Purpose")}</dt>
          <dd>{b.purpose || t("Tidak diisi", "Not provided")}</dd>
        </div>
        {showDocument && (
          <div className="span-2">
            <dt>
              {t("Nota Dinas atau Permohonan Kunjungan Resmi", "Official memo or visit request")}
            </dt>
            <dd>
              {b.ndName || t("Tidak dilampirkan", "Not attached")}
              {b.ndName && (
                <small className="muted detail-document-hint">
                  {preview
                    ? t(
                        "Preview: hanya nama berkas, tanpa unggahan atau unduhan nyata.",
                        "Preview: filename only, without real upload or download.",
                      )
                    : b.attachmentAvailable
                      ? t(
                          "Gunakan tombol unduh dokumen di bawah.",
                          "Use the document download button below.",
                        )
                      : t(
                          "Dokumen tidak tersedia dalam akses Anda.",
                          "This document is unavailable within your access.",
                        )}
                </small>
              )}
            </dd>
          </div>
        )}
      </dl>
    </>
  );
}
