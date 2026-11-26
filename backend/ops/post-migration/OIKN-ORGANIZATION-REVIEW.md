# OIKN organization data migration review

The [SQL draft](seed-oikn-organization-2026-11.sql) covers **41 pejabat appointments for 40 named people**, with full titles and links to **7 existing kedeputian** and **20 existing directorates** where applicable. The organization page identifies the groups, the regulations establish the reporting lines, and each proposed appointment includes a link to its official profile. The roster cards on the main page were unavailable during verification. [OIKN organization page](https://ikn.go.id/id/struktur-organisasi), [Perka OIKN 1/2022](https://ikn.go.id/storage/regulasi/20220909-peraturan-kepala-otorita-ikn-12022-otk-ikn.pdf), [OIKN strategic plan](https://ikn.go.id/storage/regulasi/20251010-perka-oikn-11-2025-tentang-rencana-strategis-otorita-ikn-tahun-2025-2029.pdf)

## Items to confirm

| Position or rule | Proposed handling | Evidence |
| --- | --- | --- |
| Wakil Kepala Otorita Ibu Kota Nusantara | Omitted. The indexed official profile has a blank name. | [OIKN profile](https://ikn.go.id/id/organization/wakil-kepala-otorita-ibu-kota-nusantara) |
| Staf Khusus Bidang Komunikasi Publik | Omitted. OIKN reported Troy Pantouw's death in July 2026. The indexed profile shows a blank name. | [Official announcement](https://ikn.go.id/id/posts/otorita-ikn-berduka-cita-atas-berpulangnya-staf-khusus-kepala-otorita-ikn-bidang-komunikasi-publik) |
| Direktur Pengelolaan Gedung, Kawasan, dan Perkotaan | Omitted until the current holder is confirmed. The profile lists Heri Nugroho. A July 2026 release names Cakra Nagara as acting director there. His own profile lists him as Direktur Sarana Prasarana Dasar, the position included for him in the draft. | [Heri profile](https://ikn.go.id/id/organization/direktur-pengelolaan-gedung-kawasan-dan-perkotaan), [July release](https://ikn.go.id/id/posts/otorita-ikn-hadirkan-hunian-pekerja-konstruksi-yang-aman-nyaman-dan-responsif-gender), [Cakra profile](https://ikn.go.id/id/organization/direktur-sarana-prasarana-dasar) |
| Kepala Bagian Protokol | Omitted. The regulation lists the unit. The current holder needs confirmation. | [Perka OIKN 1/2022, Pasal 17](https://ikn.go.id/storage/regulasi/20220909-peraturan-kepala-otorita-ikn-12022-otk-ikn.pdf) |
| Acting appointments | Preserve the published acting title for Ambar Tri Bawono, Ratih Febriana, and Yati Suharyani. Yati also appears under her separately published Kepala Bagian Rumah Tangga appointment. VMS stores a separate pejabat row per position, so she will appear twice. | [Ambar profile](https://ikn.go.id/id/organization/direktur-data-dan-kecerdasan-buatan), [Ratih profile](https://ikn.go.id/en/organization/plt-direktur-hukum), [Yati acting profile](https://ikn.go.id/id/organization/plt-kepala-biro-umum-dan-pengadaan-barangjasa), [Yati Rumah Tangga profile](https://ikn.go.id/id/organization/kepala-bagian-rumah-tangga) |
| VIP routing | Kepala Otorita Ibu Kota Nusantara is the only VIP position. Every other position is standard. | [VMS physical schema](../../../docs/contracts/PHYSICAL-SCHEMA-2026-11-18.md) |

The VMS hierarchy links a directorate to one kedeputian. The three legal directorates sit under Unit Kerja Hukum dan Kepatuhan, which is outside the seven kedeputian. Their positions and named officials are included with no kedeputian or directorate foreign key. The same nullable organization links cover leadership, Secretariat, and the listed special staff. Additional special-staff appointments without current OIKN confirmation are outside this draft. [VMS organization mapping](../../../docs/contracts/EMPLOYEE-ORGANIZATION.md)

## Review and execution

The seed starts from named pejabat appointments. It uses the full published title for `vms.jabatan.nama` and inserts a matching jabatan only when needed for the required `vms.pejabat.id_jabatan` foreign key. It resolves unit IDs from the existing `vms.kedeputian` and `vms.direktorat` rows. The fields on `vms.pegawai` describe the employee. An official may have zero or multiple assigned employees, so the seed uses the official appointment and the organization hierarchy. The source links are listed below. Use the default review mode in PostgreSQL 18 `psql` to print the mapping on a disposable database. The preview works without the VMS schema.

```powershell
psql -X -d <review_database> -f ops/post-migration/seed-oikn-organization-2026-11.sql
```

After approval, apply the schema migrations and run the [post-migration checks](README.md). Then connect to the same VMS database as `vms_migrator` and pass both `-v apply=true` and `-v expected_database=<exact VMS database name>` to the SQL file. Use the configured host and port. Applying the file inserts missing pejabat and their required positions and preserves exact existing matches. Missing or conflicting units, positions, VIP values or current holders abort the transaction. Existing rows outside the reviewed mapping, including any synthetic catalog entries, are preserved.

## Sources by position

| Jabatan | Source |
| --- | --- |
| Kepala Otorita Ibu Kota Nusantara | [Official source](https://ikn.go.id/id/organization/kepala-otorita-ibu-kota-nusantara) |
| Staf Khusus Kepala Otorita IKN Bidang Keamanan dan Keselamatan Publik | [Official source](https://ikn.go.id/id/organization/staf-khusus-kepala-otorita-ikn-bidang-keamanan-dan-keselamatan-publik) |
| Staf Khusus Kepala Otorita Ibu Kota Nusantara Bidang Manajemen Kebijakan dan Strategi Konstruksi | [Official source](https://ikn.go.id/id/organization/staf-khusus-kepala-otorita-ibu-kota-nusantara-bidang-manajemen-dan-strategi-konstruksi) |
| Sekretaris | [Official source](https://ikn.go.id/id/organization/sekretaris) |
| Kepala Biro Perencanaan, Organisasi, dan Kerja Sama | [Official source](https://ikn.go.id/id/organization/kepala-biro-perencanaan-organisasi-dan-kerja-sama) |
| Kepala Biro Sumber Daya Manusia dan Hubungan Masyarakat | [Official source](https://ikn.go.id/id/organization/kepala-biro-sumber-daya-manusia-dan-hubungan-masyarakat) |
| (Plt.) Kepala Biro Umum dan Pengadaan Barang/Jasa | [Official source](https://ikn.go.id/id/organization/plt-kepala-biro-umum-dan-pengadaan-barangjasa) |
| Kepala Biro Keuangan, Barang Milik Negara, dan Aset Dalam Penguasaan | [Official source](https://ikn.go.id/id/organization/kepala-biro-keuangan-barang-milik-negara-dan-aset-dalam-penguasaan) |
| Kepala Unit Kerja Hukum dan Kepatuhan | [Official source](https://ikn.go.id/id/organization/kepala-unit-kerja-hukum-dan-kepatuhan) |
| (Plt.) Direktur Hukum | [Official source](https://ikn.go.id/en/organization/plt-direktur-hukum) |
| Direktur Kepatuhan | [Official source](https://ikn.go.id/id/organization/direktur-kepatuhan) |
| Direktur Pengawasan dan Audit Internal | [Official source](https://ikn.go.id/id/organization/direktur-pengawasan-dan-audit-internal) |
| Deputi Bidang Perencanaan dan Pertanahan | [Official source](https://ikn.go.id/id/organization/deputi-bidang-perencanaan-dan-pertanahan) |
| Direktur Perencanaan Makro | [Official source](https://ikn.go.id/id/organization/direktur-perencanaan-makro) |
| Direktur Perencanaan Mikro | [Official source](https://ikn.go.id/id/organization/direktur-perencanaan-mikro) |
| Direktur Pertanahan | [Official source](https://ikn.go.id/id/organization/direktur-pertanahan) |
| Deputi Bidang Pengendalian Pembangunan | [Official source](https://ikn.go.id/id/organization/deputi-bidang-pengendalian-pembangunan) |
| Direktur Pengendalian Penyelenggaraan Pemerintahan dan Perizinan Pembangunan | [Official source](https://ikn.go.id/id/organization/direktur-pengendalian-penyelenggaraan-pemerintahan-dan-perizinan-pembangunan) |
| Direktur Pengawasan, Pemantauan, dan Evaluasi | [Official source](https://ikn.go.id/id/organization/direktur-pengawasan-pemantauan-dan-evaluasi) |
| Direktur Ketentraman dan Ketertiban Umum | [Official source](https://ikn.go.id/id/organization/direktur-ketentraman-dan-ketertiban-umum) |
| Deputi Bidang Sosial, Budaya, dan Pemberdayaan Masyarakat | [Official source](https://ikn.go.id/id/organization/deputi-bidang-sosial-budaya-dan-pemberdayaan-masyarakat) |
| Direktur Pelayanan Dasar | [Official source](https://ikn.go.id/id/organization/direktur-pelayanan-dasar) |
| Direktur Pemberdayaan Masyarakat | [Official source](https://ikn.go.id/id/organization/direktur-pemberdayaan-masyarakat) |
| Direktur Kebudayaan, Pariwisata, dan Ekonomi Kreatif | [Official source](https://ikn.go.id/id/organization/direktur-kebudayaan-pariwisata-dan-ekonomi-kreatif) |
| Deputi Bidang Transformasi Hijau dan Digital | [Official source](https://ikn.go.id/id/organization/deputi-bidang-transformasi-hijau-dan-digital) |
| Direktur Pengembangan Ekosistem Digital | [Official source](https://ikn.go.id/id/organization/direktur-pengembangan-ekosistem-digital) |
| Direktur Transformasi Hijau | [Official source](https://ikn.go.id/id/organization/direktur-transformasi-hijau) |
| (Plt.) Direktur Data dan Kecerdasan Buatan | [Official source](https://ikn.go.id/id/organization/direktur-data-dan-kecerdasan-buatan) |
| Deputi Bidang Lingkungan Hidup dan Sumber Daya Alam | [Official source](https://ikn.go.id/id/organization/deputi-bidang-lingkungan-hidup-dan-sumber-daya-alam) |
| Direktur Lingkungan Hidup dan Penanggulangan Bencana | [Official source](https://ikn.go.id/id/organization/direktur-lingkungan-hidup-dan-penanggulangan-bencana) |
| Direktur Pengembangan Pemanfaatan Kehutanan dan Sumber Daya Air | [Official source](https://ikn.go.id/id/organization/direktur-pengembangan-pemanfaatan-kehutanan-dan-sumber-daya-air) |
| Direktur Ketahanan Pangan | [Official source](https://ikn.go.id/id/organization/direktur-ketahanan-pangan) |
| Deputi Bidang Pendanaan dan Investasi | [Official source](https://ikn.go.id/id/organization/deputi-bidang-pendanaan-dan-investasi) |
| Direktur Investasi dan Kemudahan Berusaha | [Official source](https://ikn.go.id/id/organization/direktur-investasi-dan-kemudahan-berusaha) |
| Direktur Pendanaan | [Official source](https://ikn.go.id/id/organization/direktur-pendanaan) |
| Direktur Pembiayaan | [Official source](https://ikn.go.id/id/organization/direktur-pembiayaan) |
| Deputi Bidang Sarana dan Prasarana | [Official source](https://ikn.go.id/id/organization/deputi-bidang-sarana-dan-prasarana) |
| Direktur Sarana Prasarana Dasar | [Official source](https://ikn.go.id/id/organization/direktur-sarana-prasarana-dasar) |
| Direktur Sarana Prasarana Sosial | [Official source](https://ikn.go.id/id/organization/direktur-sarana-prasarana-sosial) |
| Kepala Bagian Tata Usaha | [Official source](https://ikn.go.id/id/organization/kepala-bagian-tata-usaha) |
| Kepala Bagian Rumah Tangga | [Official source](https://ikn.go.id/id/organization/kepala-bagian-rumah-tangga) |
