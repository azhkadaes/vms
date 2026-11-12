import { preview, useCatalogs } from "@vms/data";
import { Button } from "@vms/ui";
import { useLanguage } from "../i18n";

export default function PrivacyNotice() {
  const catalogs = useCatalogs();
  const { language, t } = useLanguage();
  const text = preview
    ? t(
        "Nama rombongan dan kontak digunakan untuk penjadwalan, pemberitahuan dan verifikasi kunjungan. Bawa identitas asli ke resepsionis. Gunakan data fiktif pada pratinjau ini: dokumen dan email tidak dikirim, dan data contoh hilang saat layanan pengembangan dimulai ulang. Pemberitahuan privasi resmi harus diterbitkan organisasi sebelum layanan produksi digunakan.",
        "Visitor names and contact details are used to schedule, notify and verify visits. Bring your original ID to reception. Use fictional details in this preview: documents and email are not sent, and example data disappears when the development service restarts. The organization must publish its official privacy notice before production use.",
      )
    : catalogs.data?.privacyNotice?.[language] ||
      t(
        "Pemberitahuan privasi belum tersedia. Coba lagi sebelum mengirim permohonan.",
        "The privacy notice is unavailable. Try again before submitting a request.",
      );
  return (
    <>
      <span style={{ whiteSpace: "pre-line" }}>{text}</span>
      {!preview && !catalogs.data?.privacyNotice && (
        <Button
          type="button"
          kind="ghost"
          disabled={catalogs.isFetching}
          onClick={() => void catalogs.refetch()}
        >
          {t("Coba lagi", "Try again")}
        </Button>
      )}
    </>
  );
}
