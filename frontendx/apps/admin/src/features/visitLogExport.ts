import { labels } from "@vms/data";
import { selectedFilters } from "@vms/data/report-filter";
import { visitKindLabels, type VisitLogFilter, type VisitLogSnapshot } from "@vms/data/visit-log";

export function visitLogPeriod(filter: VisitLogFilter) {
  if (!filter.from && !filter.to) return "Semua waktu";
  return `${filter.from || "Awal pencatatan"} s.d. ${filter.to || "Terbaru"}`;
}
export function visitLogTimestamp(value: string | null) {
  return value
    ? new Intl.DateTimeFormat("id-ID", {
        timeZone: "Asia/Makassar",
        dateStyle: "short",
        timeStyle: "short",
      }).format(new Date(value))
    : "—";
}
export function visitLogCsv(snapshot: VisitLogSnapshot) {
  const rows = [
    [
      "Referensi",
      "Pengunjung",
      "Instansi",
      "Jenis",
      "Tanggal",
      "Waktu (WITA)",
      "Durasi (menit)",
      "Tujuan",
      "Ruangan",
      "Keperluan",
      "Jumlah pengunjung",
      "Status",
      "Check in (WITA)",
      "Check out (WITA)",
    ],
    ...snapshot.items.map((b) => [
      b.reference,
      b.name,
      b.organization,
      visitKindLabels[b.kind],
      b.date,
      b.time,
      String(b.duration),
      b.host,
      b.room,
      b.purpose,
      b.visitors === null ? "Tidak dicatat" : String(b.visitors),
      labels[b.status],
      visitLogTimestamp(b.checkedInAt),
      visitLogTimestamp(b.checkedOutAt),
    ]),
  ];
  // Quote every value and neutralize spreadsheet formulas, including whitespace-prefixed ones.
  return (
    "\uFEFF" +
    rows
      .map((row) =>
        row
          .map((value) => {
            const safe =
              /^[\s\u0000-\u001f]*[=+\-@]/.test(value) || /^[\t\r\n]/.test(value)
                ? `'${value}`
                : value;
            return `"${safe.replaceAll('"', '""')}"`;
          })
          .join(","),
      )
      .join("\r\n")
  );
}
export async function visitLogPdf(snapshot: VisitLogSnapshot) {
  const [{ jsPDF }, { autoTable }] = await Promise.all([
    import("jspdf"),
    import("jspdf-autotable"),
  ]);
  const doc = new jsPDF({ orientation: "landscape", unit: "mm", format: "a4" });
  const width = doc.internal.pageSize.getWidth();
  const height = doc.internal.pageSize.getHeight();
  doc.setProperties({ title: "Laporan Log Kunjungan", author: "Otorita Ibu Kota Nusantara" });
  const filter = snapshot.filter;
  const statusNames = selectedFilters(filter.status, "all")
    .filter((status) => status !== "all")
    .map((status) => labels[status]);
  const kindNames = selectedFilters(filter.kind, "all")
    .filter((kind) => kind !== "all")
    .map((kind) => visitKindLabels[kind]);
  const destinations = selectedFilters(filter.officer, "")
    .map((id) => snapshot.items.find((b) => b.officerId === id)?.host || id);
  const rooms = selectedFilters(filter.room, "");
  const selected = [
    `Periode: ${visitLogPeriod(filter)}`,
    `Status: ${statusNames.join(", ") || "Semua"}`,
    `Jenis: ${kindNames.join(", ") || "Semua"}`,
    filter.q && `Pencarian: ${filter.q}`,
    destinations.length && `Tujuan: ${destinations.join(", ")}`,
    rooms.length && `Ruangan: ${rooms.join(", ")}`,
    `Urutan: ${{ date: "Jadwal", name: "Pengunjung", organization: "Instansi", host: "Tujuan", status: "Status", kind: "Jenis", visitors: "Jumlah pengunjung" }[filter.sort]} (${filter.dir === "asc" ? "naik" : "turun"})`,
  ]
    .filter(Boolean)
    .join("  |  ");
  doc.setFontSize(18);
  doc.setTextColor(25, 45, 66);
  doc.text("Laporan Log Kunjungan", 12, 17);
  doc.setFontSize(9);
  doc.text("Otorita Ibu Kota Nusantara · Visitor Management System", 12, 24);
  doc.setFontSize(8);
  const filterLines = doc.splitTextToSize(selected, width - 24) as string[];
  doc.text(filterLines, 12, 32);
  const summaryY = 34 + filterLines.length * 4;
  const s = snapshot.summary;
  doc.text(
    `Total: ${s.total} kunjungan   |   Pengunjung tercatat: ${s.visitors} orang   |   Sedang berkunjung: ${s.inside}   |   Selesai: ${s.completed}   |   Pending: ${s.pending}`,
    12,
    summaryY,
  );
  autoTable(doc, {
    startY: summaryY + 5,
    margin: { top: 15, right: 12, bottom: 19, left: 12 },
    head: [
      [
        "No. / Referensi",
        "Pengunjung / Instansi",
        "Jenis",
        "Jadwal WITA",
        "Tujuan / Ruangan",
        "Keperluan",
        "Orang",
        "Status",
        "Kehadiran WITA",
      ],
    ],
    body: snapshot.items.map((b, i) => [
      `${i + 1}. ${b.reference}`,
      `${b.name}\n${b.organization || "—"}`,
      visitKindLabels[b.kind],
      `${b.date}\n${b.time} (${b.duration} mnt)`,
      `${b.host || "—"}\n${b.room || "—"}`,
      b.purpose || "—",
      b.visitors === null ? "—" : String(b.visitors),
      labels[b.status],
      `Masuk: ${visitLogTimestamp(b.checkedInAt)}\nKeluar: ${visitLogTimestamp(b.checkedOutAt)}`,
    ]),
    styles: {
      font: "helvetica",
      fontSize: 7,
      cellPadding: 2.5,
      overflow: "linebreak",
      valign: "top",
    },
    headStyles: { fillColor: [29, 57, 83], textColor: 255, fontStyle: "bold" },
    alternateRowStyles: { fillColor: [245, 248, 250] },
    columnStyles: {
      0: { cellWidth: 26 },
      1: { cellWidth: 38 },
      2: { cellWidth: 18 },
      3: { cellWidth: 29 },
      4: { cellWidth: 37 },
      5: { cellWidth: 34 },
      6: { cellWidth: 13 },
      7: { cellWidth: 24 },
    },
    rowPageBreak: "avoid",
  });
  const pages = doc.getNumberOfPages();
  for (let page = 1; page <= pages; page++) {
    doc.setPage(page);
    doc.setFontSize(7);
    doc.setTextColor(90);
    doc.text(
      `Dibuat ${visitLogTimestamp(snapshot.generatedAt)} WITA · Jumlah pengunjung terbatas tidak dicatat.`,
      12,
      height - 10,
    );
    doc.text(`Halaman ${page} / ${pages}`, width - 12, height - 10, { align: "right" });
  }
  return doc.output("blob");
}

export async function downloadVisitLog(
  snapshot: VisitLogSnapshot,
  format: "csv" | "pdf",
  isCurrent = () => true,
) {
  const blob =
    format === "csv"
      ? new Blob([visitLogCsv(snapshot)], { type: "text/csv;charset=utf-8" })
      : await visitLogPdf(snapshot);
  if (!isCurrent()) return;
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `log-kunjungan-${snapshot.filter.from || "semua"}-${snapshot.filter.to || "waktu"}.${format}`;
  document.body.append(link);
  link.click();
  link.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}
