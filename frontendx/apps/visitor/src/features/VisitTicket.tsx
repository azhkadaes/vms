import type { PublicBooking } from "@vms/data";
import { dateLabel, visitTiming } from "@vms/data";
import { Loading } from "@vms/ui";
import { Ban, CircleX, Clock3, LogIn, LogOut, ShieldCheck } from "lucide-react";
import { lazy, Suspense } from "react";
import { useLanguage } from "../i18n";

const AdmissionQR = lazy(() => import("./AdmissionQR"));

export default function VisitTicket({
  booking,
  headingLevel = 3,
}: {
  booking: PublicBooking;
  headingLevel?: 3 | 4;
}) {
  const { language, t } = useLanguage();
  const status = visitTiming(booking).displayStatus;
  const approved = status === "approved";
  const PersonHeading = headingLevel === 3 ? "h3" : "h4";
  const statusText = {
    pending: t("Menunggu persetujuan", "Awaiting approval"),
    approved: t("Kunjungan disetujui", "Visit approved"),
    rejected: t("Permohonan ditolak", "Request declined"),
    cancelled: t("Kunjungan dibatalkan", "Visit canceled"),
    checked_in: t("Sedang berkunjung", "Visit in progress"),
    checked_out: t("Kunjungan selesai", "Visit completed"),
    missed: t("Kunjungan terlewat", "Visit missed"),
  }[status];
  const StatusIcon = {
    pending: Clock3,
    approved: ShieldCheck,
    rejected: CircleX,
    cancelled: Ban,
    checked_in: LogIn,
    checked_out: LogOut,
    missed: Clock3,
  }[status];

  return (
    <article
      className={`visit-ticket visit-ticket--${status}`}
      aria-label={t("Detail kunjungan", "Visit details")}
    >
      <div className="visit-ticket__main">
        <div className="visit-ticket__topline">
          <span className="visit-ticket__brand">
            Otorita IKN <span aria-hidden="true">/</span> {t("Kunjungan", "Visit")}
          </span>
          <span className="visit-ticket__status" role="status">
            <StatusIcon size={16} aria-hidden="true" />
            {statusText}
          </span>
        </div>
        <div className="visit-ticket__person">
          <span className="visit-ticket__eyebrow">
            {t("Perwakilan pengunjung", "Visitor representative")}
          </span>
          <PersonHeading>
            {booking.name || t("Nama belum diisi", "Name not provided")}
          </PersonHeading>
          {booking.organization && <p>{booking.organization}</p>}
        </div>
        <div className="visit-ticket__schedule">
          <div>
            <span className="visit-ticket__eyebrow">{t("Tanggal kunjungan", "Visit date")}</span>
            <strong>{dateLabel(booking.date, language)}</strong>
          </div>
          <div>
            <span className="visit-ticket__eyebrow">{t("Waktu · WITA", "Time · WITA")}</span>
            <strong>
              {booking.time}{" "}
              <span className="visit-ticket__duration">
                · {booking.duration} {t("menit", "min")}
              </span>
            </strong>
          </div>
        </div>
        <dl className="visit-ticket__details">
          <div>
            <dt>{t("Bertemu", "Meeting")}</dt>
            <dd>{booking.host || t("Belum ditentukan", "Not assigned")}</dd>
          </div>
          <div>
            <dt>{t("Tempat", "Room")}</dt>
            <dd>
              {booking.status === "pending"
                ? t("Dikonfirmasi setelah persetujuan", "Confirmed after approval")
                : booking.room || t("Belum ditentukan", "Not assigned")}
            </dd>
          </div>
          <div>
            <dt>{t("Jumlah pengunjung", "Visitors")}</dt>
            <dd>{booking.visitors.length || t("Belum diisi", "Not provided")}</dd>
          </div>
          <div>
            <dt>{t("Keperluan", "Purpose")}</dt>
            <dd>{booking.purpose || t("Tidak diisi", "Not provided")}</dd>
          </div>
          {booking.visitorInstructions && (
            <div>
              <dt>{t("Instruksi pengunjung", "Visitor instructions")}</dt>
              <dd className="visit-note-text">{booking.visitorInstructions}</dd>
            </div>
          )}
        </dl>
      </div>
      <div className="visit-ticket__stub">
        <div className="visit-ticket__code">
          <span className="visit-ticket__eyebrow">{t("Kode kunjungan", "Visit code")}</span>
          <strong>{booking.code}</strong>
        </div>
        {approved ? (
          <Suspense fallback={<Loading />}>
            <AdmissionQR booking={booking} ticket />
          </Suspense>
        ) : (
          <p className="visit-ticket__eligibility">
            {booking.status === "pending"
              ? t(
                  "Permohonan ini belum menjadi izin masuk. QR tersedia setelah disetujui.",
                  "This request does not grant entry. A QR code becomes available after approval.",
                )
              : t(
                  "QR masuk tidak tersedia untuk status kunjungan ini.",
                  "An entry QR code is unavailable for this visit status.",
                )}
          </p>
        )}
      </div>
    </article>
  );
}
