import { dateLabel, type StaffBooking } from "@vms/data";
import { useId, type ReactNode } from "react";
import { useUiLanguage } from "./language";

export type RestrictedIdentitySections = { visit?: ReactNode; visitor?: ReactNode };

export function RestrictedIdentity({
  booking: b,
  sections,
}: {
  booking: StaffBooking;
  sections?: RestrictedIdentitySections;
}) {
  const { language, t } = useUiLanguage();
  const headingId = useId();
  const schedule = (
    <>
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
      {b.code && (
        <div>
          <dt>{t("Kode booking", "Booking code")}</dt>
          <dd className="tabular">{b.code}</dd>
        </div>
      )}
    </>
  );
  const officer = (
    <div>
      <dt>{t("Pejabat yang dituju", "Officer to meet")}</dt>
      <dd>{b.host || t("Tidak diisi", "Not supplied")}</dd>
    </div>
  );
  const visitor = (
    <>
      <div>
        <dt>{t("Nama", "Name")}</dt>
        <dd>{b.name || t("Tidak diisi", "Not supplied")}</dd>
      </div>
      {b.restrictedView !== "arrival" && (
        <div>
          <dt>{t("Instansi", "Organization")}</dt>
          <dd>{b.organization || t("Tidak diisi", "Not supplied")}</dd>
        </div>
      )}
    </>
  );
  const serviceId = (
    <div>
      <dt>
        {b.restrictedView === "full"
          ? "No ID Dinas"
          : t("No ID Dinas · 4 karakter terakhir", "Service ID · last 4 characters")}
      </dt>
      <dd>{b.serviceId || b.serviceIdLast4 || t("Tidak diisi", "Not supplied")}</dd>
    </div>
  );
  return (
    <section aria-label={t("Rincian kunjungan terbatas", "Restricted visit details")}>
      <h3>{b.reference || t("Kunjungan terbatas", "Restricted visit")}</h3>
      {sections && b.restrictedView === "full" ? (
        <>
          <section className="section-gap" aria-labelledby={`${headingId}-visit`}>
            <div className="restricted-detail-heading">
              <h4 id={`${headingId}-visit`}>{t("Detail kunjungan", "Visit details")}</h4>
              {sections.visit}
            </div>
            <dl className="detail-grid">
              {schedule}
              {officer}
            </dl>
          </section>
          <section className="section-gap" aria-labelledby={`${headingId}-visitor`}>
            <div className="restricted-detail-heading">
              <h4 id={`${headingId}-visitor`}>{t("Detail pengunjung", "Visitor details")}</h4>
              {sections.visitor}
            </div>
            <dl className="detail-grid">
              {visitor}
              {serviceId}
            </dl>
          </section>
        </>
      ) : (
        <dl className="detail-grid section-gap">
          {schedule}
          {b.restrictedView !== "reception" && (
            <>
              {visitor}
              {officer}
              {serviceId}
            </>
          )}
        </dl>
      )}
      {b.restrictedView === "reception" && (
        <p className="muted section-gap">
          {t(
            "Identitas tidak ditampilkan di rincian umum. Gunakan kode atau QR untuk verifikasi kedatangan.",
            "Identity is hidden in general details. Use a code or QR for arrival verification.",
          )}
        </p>
      )}
    </section>
  );
}
