import type { PublicVisitAction, PublicVisitEvent } from "@vms/data";
import { useLanguage } from "../i18n";

const labels: Record<PublicVisitAction, [string, string]> = {
  received: ["Permohonan diterima", "Request received"],
  approved: ["Kunjungan disetujui", "Visit approved"],
  rejected: ["Permohonan ditolak", "Request declined"],
  rescheduled: ["Jadwal kunjungan diubah", "Visit rescheduled"],
  cancelled: ["Kunjungan dibatalkan", "Visit canceled"],
  checked_in: ["Kedatangan dicatat", "Arrival recorded"],
  checked_out: ["Kepulangan dicatat", "Departure recorded"],
  updated: ["Kunjungan diperbarui", "Visit updated"],
};

export default function PublicVisitHistory({
  events,
  headingLevel = 2,
}: {
  events: PublicVisitEvent[];
  headingLevel?: 2 | 3;
}) {
  const { language, t } = useLanguage();
  const Heading = headingLevel === 2 ? "h2" : "h3";
  const format = new Intl.DateTimeFormat(language === "en" ? "en-US" : "id-ID", {
    dateStyle: "medium",
    timeStyle: "short",
    hourCycle: "h23",
    timeZone: "Asia/Makassar",
  });
  return (
    <section className="public-visit-history section-gap" aria-labelledby="visit-history-title">
      <Heading id="visit-history-title">{t("Riwayat kunjungan", "Visit history")}</Heading>
      {events.length ? (
        <ol className="timeline">
          {events.map((event, index) => {
            const label = labels[event.action] || labels.updated;
            const date = new Date(event.at);
            return (
              <li key={`${event.at}-${index}`}>
                {t(...label)}
                {event.action === "rescheduled" && event.publicMessage && (
                  <p>{event.publicMessage}</p>
                )}
                <small>
                  {Number.isFinite(date.getTime()) ? (
                    <time dateTime={event.at}>{format.format(date)} WITA</time>
                  ) : (
                    t("Waktu belum tersedia", "Time unavailable")
                  )}
                </small>
              </li>
            );
          })}
        </ol>
      ) : (
        <p>{t("Riwayat belum tersedia.", "History is not available yet.")}</p>
      )}
    </section>
  );
}
