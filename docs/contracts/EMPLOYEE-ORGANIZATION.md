# Employee official assignment and organization

User correction, 15 November 2026: an employee has at most one current pejabat assignment. Multiple roles remain supported in the employee editor. This supersedes the historical D94 limit of 100 assigned officials; it does not add employee write endpoints to the read-only Phase 1 API.

- Keep `assignedOfficerIds` as an array for wire compatibility, with zero or one ID on writes. Reject larger arrays without truncating or choosing the first ID. Old multi-assignment records can still be read so an operator can choose a replacement explicitly.
- The searchable pejabat control is single-select. Selecting another official replaces the current selection. PA directory scope still depends on the explicit official assignment, never on an organization label.
- The catalog optionally returns `jobTitle`, `directorate`, and `deputy`. Those come from the official's Jabatan and linked organizational units. Missing metadata stays unknown; do not guess a person's unit from their name or ID.
- For an assigned PA, show the official's directorate and deputy automatically and submit these values as profile metadata. They are read-only in this flow. Other employee roles retain their existing optional profile fields.
- Since the per-table rebuild of 18 November 2026, four migrations carry this: `202611180002_create_vms_kedeputian.sql` and `202611180003_create_vms_direktorat.sql` create the Kedeputian-to-Direktorat reference hierarchy, `202611180004_create_vms_jabatan.sql` links each Jabatan to at most one unit, and `202611180009_create_vms_pegawai_pejabat.sql` enforces one active Pegawai-Pejabat row per employee with a partial unique index. Inactive historical rows are retained.
- The reference seed adds verified unit names only. It invents no official, person or assignment.

Verified sources:

- [OIKN organization structure](https://ikn.go.id/id/struktur-organisasi).
- [Perka OIKN 1/2022](https://ikn.go.id/storage/regulasi/20220909-peraturan-kepala-otorita-ikn-12022-otk-ikn.pdf), articles 33, 43, 53, 63, 73, 83 and 93: seven deputy groups and their 21 directorates.
- [Transformasi Hijau dan Digital](https://satudata.ikn.go.id/organization/kedeputian-bidang-transformasi-hijau-dan-digital) and [Sosial, Budaya, dan Pemberdayaan Masyarakat](https://satudata.ikn.go.id/organization/kedeputian-bidang-sosial-budaya-dan-pemberdayaan-masyarakat) cross-check the two existing preview destinations.

Units outside these deputy/directorate groups, including leadership, Secretariat and legal/compliance, are not forced into a fabricated directorate or deputy. Their catalog metadata remains optional until correctly linked.
