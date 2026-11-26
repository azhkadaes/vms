import { createContext, useContext, type ReactNode } from "react";

export type UiLanguage = "id" | "en";
const LanguageContext = createContext<UiLanguage>("id");

export function UiLanguageProvider({ language, children }: { language: UiLanguage; children: ReactNode }) {
  return <LanguageContext.Provider value={language}>{children}</LanguageContext.Provider>;
}

export function useUiLanguage() {
  const language = useContext(LanguageContext);
  return {
    language,
    t: (indonesian: string, english: string) => language === "en" ? english : indonesian,
  };
}

const englishErrors: Record<string, string> = {
  "Pembatalan mandiri hanya tersedia sebelum waktu mulai kunjungan. Hubungi petugas untuk bantuan.": "Self-service cancellation is only available before the visit start time. Contact staff for help.",
  "Pilih pejabat aktif yang tersedia.": "Choose an available active official.",
  "Periksa data permohonan.": "Check the request details.",
  "Lengkapi rombongan pengunjung.": "Complete the visitor group.",
  "Rombongan maksimal 20 orang.": "The visitor group can contain up to 20 people.",
  "Nama perwakilan harus menjadi pengunjung pertama.": "The representative must be the first visitor.",
  "Pilih nama dokumen yang valid.": "Choose a valid document filename.",
  "Nama dokumen maksimal 255 karakter.": "Document filenames can contain up to 255 characters.",
  "Pilih nama foto yang valid.": "Choose a valid photo filename.",
  "Nama foto maksimal 255 karakter.": "Photo filenames can contain up to 255 characters.",
  "Gunakan foto JPEG atau PNG.": "Use a JPEG or PNG photo.",
  "Unggah foto wajah perwakilan.": "Upload a face photo of the representative.",
  "Ukuran foto maksimal 5 MB dan tidak boleh kosong.": "The photo must be non-empty and no larger than 5 MB.",
  "Email pengajuan sebelumnya tidak cocok. Periksa email dan coba lagi.": "The previous submission email does not match. Check the email and try again.",
  "Hasil pengiriman belum dapat dipastikan.": "The submission result could not be confirmed.",
  "Browser tidak dapat menyimpan referensi pengiriman. Izinkan penyimpanan sesi lalu coba lagi.": "The browser could not save the submission reference. Allow session storage and try again.",
  "Periksa nama email dan domain Anda.": "Check the email name and domain.",
  "Pilih tujuan kunjungan.": "Choose who to visit.",
  "Pilih tanggal yang valid, hari ini atau sesudahnya.": "Choose a valid date, today or later.",
  "Pilih waktu mulai antara 08.00–16.00 WITA.": "Choose a start time between 08:00 and 16:00 WITA.",
  "Pilih waktu mulai antara 08.00–16.59 WITA.": "Choose a start time between 08:00 and 16:59 WITA.",
  "Jelaskan keperluan minimal 5 karakter.": "Describe the purpose in at least 5 characters.",
  "Waktu kunjungan harus di masa depan (WITA).": "The visit start time must be in the future (WITA).",
  "Masukkan nama lengkap.": "Enter a full name.",
  "Masukkan instansi atau asal.": "Enter an organization or affiliation.",
  "Gunakan 8–20 digit angka untuk nomor telepon.": "Use 8–20 digits for the phone number.",
  "Lengkapi nama pengunjung.": "Enter the visitor's name.",
  "Baca dan konfirmasi pemberitahuan privasi.": "Read and confirm the privacy notice.",
  "Gunakan PDF, JPEG, atau PNG.": "Use a PDF, JPEG, or PNG file.",
  "Gunakan berkas PDF.": "Use a PDF file.",
  "Ukuran dokumen maksimal 5 MB dan tidak boleh kosong.": "The document must be non-empty and no larger than 5 MB.",
  "Kunjungan tidak tersedia.": "Visit unavailable.",
};

export function localizeMessage(message: string | undefined, language: UiLanguage) {
  if (!message) return message;
  if (/^(signal timed out|the operation was aborted due to timeout.*)$/i.test(message.trim())) {
    return language === "en" ? "The request timed out. Check your connection." : "Permintaan terlalu lama. Periksa koneksi Anda.";
  }
  if (language === "id") return message;
  return englishErrors[message] || message;
}
