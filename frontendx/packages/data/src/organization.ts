// Official organization reference, checked 2026-11-15:
// https://ikn.go.id/id/struktur-organisasi
// Perka OIKN 1/2022, articles 33, 43, 53, 63, 73, 83 and 93:
// https://ikn.go.id/storage/regulasi/20220909-peraturan-kepala-otorita-ikn-12022-otk-ikn.pdf
// These units do not create officials or grant assignment authority.
export const organizationStructure = [
  {
    code: "PP",
    deputy: "Kedeputian Bidang Perencanaan dan Pertanahan",
    directorates: [
      "Direktorat Perencanaan Makro",
      "Direktorat Perencanaan Mikro",
      "Direktorat Pertanahan",
    ],
  },
  {
    code: "PDP",
    deputy: "Kedeputian Bidang Pengendalian Pembangunan",
    directorates: [
      "Direktorat Pengendalian Penyelenggaraan Pemerintahan dan Perizinan Pembangunan",
      "Direktorat Pengawasan, Pemantauan, dan Evaluasi",
      "Direktorat Ketentraman dan Ketertiban Umum",
    ],
  },
  {
    code: "SBPM",
    deputy: "Kedeputian Bidang Sosial, Budaya, dan Pemberdayaan Masyarakat",
    directorates: [
      "Direktorat Pelayanan Dasar",
      "Direktorat Pemberdayaan Masyarakat",
      "Direktorat Kebudayaan, Pariwisata, dan Ekonomi Kreatif",
    ],
  },
  {
    code: "THD",
    deputy: "Kedeputian Bidang Transformasi Hijau dan Digital",
    directorates: [
      "Direktorat Pengembangan Ekosistem Digital",
      "Direktorat Transformasi Hijau",
      "Direktorat Data dan Kecerdasan Buatan",
    ],
  },
  {
    code: "LHSDA",
    deputy: "Kedeputian Bidang Lingkungan Hidup dan Sumber Daya Alam",
    directorates: [
      "Direktorat Lingkungan Hidup dan Penanggulangan Bencana",
      "Direktorat Pengembangan Pemanfaatan Kehutanan dan Sumber Daya Air",
      "Direktorat Ketahanan Pangan",
    ],
  },
  {
    code: "PI",
    deputy: "Kedeputian Bidang Pendanaan dan Investasi",
    directorates: [
      "Direktorat Investasi dan Kemudahan Berusaha",
      "Direktorat Pendanaan",
      "Direktorat Pembiayaan",
    ],
  },
  {
    code: "SP",
    deputy: "Kedeputian Bidang Sarana dan Prasarana",
    directorates: [
      "Direktorat Sarana Prasarana Dasar",
      "Direktorat Sarana Prasarana Sosial",
      "Direktorat Pengelolaan Gedung, Kawasan, dan Perkotaan",
    ],
  },
] as const;

export type OfficerOrganization = { directorate: string; deputy: string };
type OfficerWithOrganization = {
  name: string;
  jobTitle?: string;
  directorate?: string;
  deputy?: string;
};

function unitName(value: string) {
  return value
    .trim()
    .toLocaleLowerCase("id-ID")
    .replace(/^direktur\s+/, "direktorat ")
    .replace(/[,\s]+/g, " ");
}

export function officerOrganization(officer?: OfficerWithOrganization): OfficerOrganization {
  if (!officer) return { directorate: "", deputy: "" };
  // Explicit catalog metadata is authoritative. Match only an exact known unit/title
  // for older catalogs; never infer a person's unit from an ID or partial name.
  const candidate = officer.directorate || officer.jobTitle || officer.name;
  const group = organizationStructure.find((group) =>
    group.directorates.some((directorate) => unitName(directorate) === unitName(candidate)),
  );
  const directorate = group?.directorates.find(
    (directorate) => unitName(directorate) === unitName(candidate),
  );
  return {
    directorate: officer.directorate?.trim() || directorate || "",
    deputy: officer.deputy?.trim() || group?.deputy || "",
  };
}
