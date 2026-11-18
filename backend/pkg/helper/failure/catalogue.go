package failure

// messages is the user-facing Indonesian text per code. The UI shows message
// verbatim, so it never carries causes, identifiers, SQL or secrets.
var messages = map[string]string{
	// 400
	"invalid_request":         "Permintaan tidak valid. Periksa data lalu coba lagi.",
	"invalid_query_parameter": "Parameter permintaan tidak valid.",
	// 401
	"unauthenticated":     "Sesi berakhir. Silakan masuk kembali.",
	"invalid_credentials": "Email atau kata sandi tidak sesuai.",
	// 403
	"forbidden":    "Akses tidak diizinkan.",
	"csrf_expired": "Formulir sudah kedaluwarsa. Muat ulang halaman lalu coba lagi.",
	// 404 and 405
	"not_found":          "Data tidak ditemukan atau tidak tersedia.",
	"method_not_allowed": "Metode permintaan tidak didukung.",
	// 409
	"version_conflict":         "Data telah berubah. Muat ulang sebelum melanjutkan.",
	"invalid_state":            "Tindakan tidak tersedia untuk status data saat ini.",
	"schedule_conflict":        "Ruangan atau pejabat memiliki jadwal yang bertabrakan.",
	"reschedule_window_closed": "Batas waktu perubahan jadwal telah berakhir.",
	"idempotency_conflict":     "Permintaan ini sudah pernah dikirim dengan data berbeda. Periksa hasil sebelumnya sebelum mengirim ulang.",
	"notice_outdated":          "Pemberitahuan privasi telah diperbarui. Baca dan setujui versi terbaru.",
	"credential_expired":       "Kode atau QR sudah tidak berlaku. Muat ulang untuk mendapatkan yang terbaru.",
	"revision_conflict":        "Data laporan telah berubah. Perbarui sebelum mengekspor.",
	"role_changed":             "Hak akses Anda berubah. Muat ulang sesi.",
	// 413 and 415
	"payload_too_large":      "Ukuran berkas atau permintaan melebihi batas.",
	"unsupported_media_type": "Jenis konten atau berkas tidak didukung.",
	// 422
	"validation_failed": "Periksa isian yang ditandai.",
	"export_too_large":  "Data terlalu banyak untuk diekspor. Persempit filter lalu coba lagi.",
	// 429
	"rate_limited": "Terlalu banyak permintaan. Tunggu sebelum mencoba lagi.",
	// 5xx
	"internal_error":      "Terjadi kesalahan layanan. Coba lagi.",
	"service_unavailable": "Layanan belum tersedia. Coba lagi.",
}

// Message returns the catalogue text for code; ok is false for codes outside the catalogue.
func Message(code string) (message string, ok bool) {
	message, ok = messages[code]
	return message, ok
}
