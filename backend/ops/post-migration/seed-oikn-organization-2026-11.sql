\set ON_ERROR_STOP on
\if :{?apply}
\else
  \set apply false
\endif

-- November 2026 snapshot. Review by default. See OIKN-ORGANIZATION-REVIEW.md.
\if :apply
BEGIN;
\else
BEGIN READ ONLY;
\endif
SET LOCAL search_path TO pg_catalog;
SET LOCAL lock_timeout TO '5s';
SET LOCAL statement_timeout TO '60s';

-- Inline seed data for the existing VMS catalog tables.
SELECT jsonb_agg(to_jsonb(seed)) AS seed_deputies
FROM (VALUES
    ('PP', 'Kedeputian Bidang Perencanaan dan Pertanahan'),
    ('PDP', 'Kedeputian Bidang Pengendalian Pembangunan'),
    ('SBPM', 'Kedeputian Bidang Sosial, Budaya, dan Pemberdayaan Masyarakat'),
    ('THD', 'Kedeputian Bidang Transformasi Hijau dan Digital'),
    ('LHSDA', 'Kedeputian Bidang Lingkungan Hidup dan Sumber Daya Alam'),
    ('PI', 'Kedeputian Bidang Pendanaan dan Investasi'),
    ('SP', 'Kedeputian Bidang Sarana dan Prasarana')
) AS seed(kode, nama)
\gset

-- Each row is a named pejabat and the full published title for that appointment.
SELECT jsonb_agg(to_jsonb(seed)) AS seed_positions
FROM (VALUES
    ('Kepala Otorita Ibu Kota Nusantara', NULL, NULL, true, 'Dr. (H.C.) Ir. H. M. Basuki Hadimuljono, M.Sc., Ph.D.'),
    ('Staf Khusus Kepala Otorita IKN Bidang Keamanan dan Keselamatan Publik', NULL, NULL, false, 'Irjen Pol. Edgar Diponegoro, S.I.K., M.H.'),
    ('Staf Khusus Kepala Otorita Ibu Kota Nusantara Bidang Manajemen Kebijakan dan Strategi Konstruksi', NULL, NULL, false, 'Dr. Ir. Danis Hidayat Sumadilaga, M.Eng.Sc.'),
    ('Sekretaris', NULL, NULL, false, 'Bimo Adi Nursanthyasto, S.T., M.B.A.'),
    ('Kepala Biro Perencanaan, Organisasi, dan Kerja Sama', NULL, NULL, false, 'IGA Krisna Murti RS, S.Sos., M.M.'),
    ('Kepala Biro Sumber Daya Manusia dan Hubungan Masyarakat', NULL, NULL, false, 'Moh. Zamroni, S.Sos., M.M.'),
    ('(Plt.) Kepala Biro Umum dan Pengadaan Barang/Jasa', NULL, NULL, false, 'Yati Suharyani, S.T.'),
    ('Kepala Biro Keuangan, Barang Milik Negara, dan Aset Dalam Penguasaan', NULL, NULL, false, 'Muji Budda''wah, S.E., M.E.'),
    ('Kepala Unit Kerja Hukum dan Kepatuhan', NULL, NULL, false, 'Agung Dodit Muliawan, Ph.D., Ak., CA., E.R.M.C.P., Cert.D.A., C.Fr.A, W.C.D.S.'),
    ('(Plt.) Direktur Hukum', NULL, NULL, false, 'Ratih Febriana, S.H., M.H.'),
    ('Direktur Kepatuhan', NULL, NULL, false, 'Dr. Firyadi, S.P., M.Si.'),
    ('Direktur Pengawasan dan Audit Internal', NULL, NULL, false, 'Kaeshar Eksa, S.T., M.Eng.'),
    ('Deputi Bidang Perencanaan dan Pertanahan', 'PP', NULL, false, 'Mia Amalia, S.T., M.Si, Ph.D.'),
    ('Direktur Perencanaan Makro', 'PP', 'Direktorat Perencanaan Makro', false, 'Pungky Widiaryanto, S. Hut., M. Sc.'),
    ('Direktur Perencanaan Mikro', 'PP', 'Direktorat Perencanaan Mikro', false, 'Mirwansyah Prawiranegara, S.T., M.Sc.'),
    ('Direktur Pertanahan', 'PP', 'Direktorat Pertanahan', false, 'Ahmad Burhanudin, S.H., M.H.'),
    ('Deputi Bidang Pengendalian Pembangunan', 'PDP', NULL, false, 'Dr. Thomas Umbu Pati Tena Bolodadi, M.Si.'),
    ('Direktur Pengendalian Penyelenggaraan Pemerintahan dan Perizinan Pembangunan', 'PDP', 'Direktorat Pengendalian Penyelenggaraan Pemerintahan dan Perizinan Pembangunan', false, 'Kuswanto, S.STP., M.U.R.P., Ph.D'),
    ('Direktur Pengawasan, Pemantauan, dan Evaluasi', 'PDP', 'Direktorat Pengawasan, Pemantauan, dan Evaluasi', false, 'dr. Silveria Octaviana Bailia, M.M.'),
    ('Direktur Ketentraman dan Ketertiban Umum', 'PDP', 'Direktorat Ketentraman dan Ketertiban Umum', false, 'Brigjen Pol. Fransiscus Barung Mangera, S.I.K., M.Han.'),
    ('Deputi Bidang Sosial, Budaya, dan Pemberdayaan Masyarakat', 'SBPM', NULL, false, 'Dr. Drs. H. Alimuddin, M.Si.'),
    ('Direktur Pelayanan Dasar', 'SBPM', 'Direktorat Pelayanan Dasar', false, 'Dr. Suwito, SKM., M.Kes., M.H.'),
    ('Direktur Pemberdayaan Masyarakat', 'SBPM', 'Direktorat Pemberdayaan Masyarakat', false, 'Dr. Conrita Ermanto, M.Si.'),
    ('Direktur Kebudayaan, Pariwisata, dan Ekonomi Kreatif', 'SBPM', 'Direktorat Kebudayaan, Pariwisata, dan Ekonomi Kreatif', false, 'Dr. Muhsin Palinrungi, S.S., M.A.'),
    ('Deputi Bidang Transformasi Hijau dan Digital', 'THD', NULL, false, 'Dr. Agung Indrajit, S.T., M.Sc.'),
    ('Direktur Pengembangan Ekosistem Digital', 'THD', 'Direktorat Pengembangan Ekosistem Digital', false, 'Tonny Agus Setiono, S.Si.T., M.T.'),
    ('Direktur Transformasi Hijau', 'THD', 'Direktorat Transformasi Hijau', false, 'Agus Gunawan, ST., M.Eng.'),
    ('(Plt.) Direktur Data dan Kecerdasan Buatan', 'THD', 'Direktorat Data dan Kecerdasan Buatan', false, 'Ambar Tri Bawono, S.T., M.Sc.'),
    ('Deputi Bidang Lingkungan Hidup dan Sumber Daya Alam', 'LHSDA', NULL, false, 'Dr. Myrna Asnawati Safitri, S.H., M.S.I.'),
    ('Direktur Lingkungan Hidup dan Penanggulangan Bencana', 'LHSDA', 'Direktorat Lingkungan Hidup dan Penanggulangan Bencana', false, 'Lazuardi Nasution, M.B.A.'),
    ('Direktur Pengembangan Pemanfaatan Kehutanan dan Sumber Daya Air', 'LHSDA', 'Direktorat Pengembangan Pemanfaatan Kehutanan dan Sumber Daya Air', false, 'Dr. Onesimus Patiung, S.Hut., M.P.'),
    ('Direktur Ketahanan Pangan', 'LHSDA', 'Direktorat Ketahanan Pangan', false, 'Dr. P. Setia Lenggono, S.Sos., M.Si.'),
    ('Deputi Bidang Pendanaan dan Investasi', 'PI', NULL, false, 'Sudiro Roi Santoso, S.T., M.T.'),
    ('Direktur Investasi dan Kemudahan Berusaha', 'PI', 'Direktorat Investasi dan Kemudahan Berusaha', false, 'Ferdinand Kana Lo, S.T., M.T.'),
    ('Direktur Pendanaan', 'PI', 'Direktorat Pendanaan', false, 'Insyafiah, S.E., Ak., M.B.A.'),
    ('Direktur Pembiayaan', 'PI', 'Direktorat Pembiayaan', false, 'Muh. Naufal Aminuddin, S.T., M.M.'),
    ('Deputi Bidang Sarana dan Prasarana', 'SP', NULL, false, 'Aswin Grandiarto Sukahar, S.T., M.B.Env.'),
    ('Direktur Sarana Prasarana Dasar', 'SP', 'Direktorat Sarana Prasarana Dasar', false, 'Cakra Nagara, S.T., M.T., M.E.'),
    ('Direktur Sarana Prasarana Sosial', 'SP', 'Direktorat Sarana Prasarana Sosial', false, 'Ir. Agus Ahyar, M.Sc.'),
    ('Kepala Bagian Tata Usaha', NULL, NULL, false, 'Drs. Firmananur, M.Si.'),
    ('Kepala Bagian Rumah Tangga', NULL, NULL, false, 'Yati Suharyani, S.T.')
) AS seed(jabatan, kode_kedeputian, direktorat, pejabat_vip, pejabat)
\gset

\if :apply
SET LOCAL vms_seed.expected_database TO :'expected_database';
DO $check$
BEGIN
    IF current_database() <> current_setting('vms_seed.expected_database')
       OR current_user <> 'vms_migrator'
       OR current_setting('server_version_num')::int / 10000 <> 18 THEN
        RAISE EXCEPTION 'Connect to the specified VMS database as vms_migrator on PostgreSQL 18';
    END IF;
END
$check$;

-- Serialize catalog writes while matching tables without natural-key constraints.
LOCK TABLE vms.kedeputian, vms.direktorat IN SHARE MODE;
LOCK TABLE vms.jabatan, vms.pejabat IN SHARE ROW EXCLUSIVE MODE;

SET LOCAL vms_seed.deputies TO :'seed_deputies';
SET LOCAL vms_seed.positions TO :'seed_positions';

DO $seed$
DECLARE
    seed_positions CONSTANT jsonb := current_setting('vms_seed.positions')::jsonb;
    office record;
    conflicts text;
    position_ids integer[];
    position_id integer;
    directorate_id integer;
    deputy_id integer;
    matching_officials integer;
BEGIN
    SELECT string_agg(source.kode, ', ' ORDER BY source.kode) INTO conflicts
    FROM jsonb_to_recordset(current_setting('vms_seed.deputies')::jsonb) AS source(kode text, nama text)
    LEFT JOIN vms.kedeputian AS existing USING (kode)
    WHERE existing.nama IS DISTINCT FROM source.nama;
    IF conflicts IS NOT NULL THEN
        RAISE EXCEPTION 'Missing or mismatched kedeputian for codes: %', conflicts;
    END IF;

    SELECT string_agg(source.direktorat, ', ' ORDER BY source.direktorat) INTO conflicts
    FROM jsonb_to_recordset(seed_positions) AS source(jabatan text, kode_kedeputian text, direktorat text, pejabat_vip boolean, pejabat text)
    LEFT JOIN vms.direktorat AS directorate ON directorate.nama = source.direktorat
    LEFT JOIN vms.kedeputian AS deputy ON deputy.id = directorate.id_kedeputian
    WHERE source.direktorat IS NOT NULL AND deputy.kode IS DISTINCT FROM source.kode_kedeputian;
    IF conflicts IS NOT NULL THEN
        RAISE EXCEPTION 'Missing direktorat or incorrect kedeputian link: %', conflicts;
    END IF;

    FOR office IN
        SELECT source.*, directorate.id AS target_directorate_id, deputy.id AS target_deputy_id
        FROM jsonb_to_recordset(seed_positions) AS source(jabatan text, kode_kedeputian text, direktorat text, pejabat_vip boolean, pejabat text)
        LEFT JOIN vms.direktorat AS directorate ON directorate.nama = source.direktorat
        LEFT JOIN vms.kedeputian AS deputy
          ON deputy.kode = source.kode_kedeputian AND source.direktorat IS NULL
        ORDER BY source.jabatan
    LOOP
        directorate_id := office.target_directorate_id;
        deputy_id := office.target_deputy_id;

        -- An existing title with or without the acting prefix represents the same office.
        SELECT array_agg(id ORDER BY id) INTO position_ids
        FROM vms.jabatan
        WHERE regexp_replace(lower(btrim(nama)), '^\(plt\.\) +', '') =
              regexp_replace(lower(office.jabatan), '^\(plt\.\) +', '');

        position_id := position_ids[1];
        IF position_id IS NULL THEN
            INSERT INTO vms.jabatan (nama, pejabat_vip, id_direktorat, id_kedeputian)
            VALUES (office.jabatan, office.pejabat_vip, directorate_id, deputy_id)
            RETURNING id INTO position_id;
        ELSIF cardinality(position_ids) <> 1 OR NOT EXISTS (
            SELECT 1 FROM vms.jabatan
            WHERE id = position_id
              AND nama = office.jabatan
              AND pejabat_vip = office.pejabat_vip
              AND id_direktorat IS NOT DISTINCT FROM directorate_id
              AND id_kedeputian IS NOT DISTINCT FROM deputy_id
        ) THEN
            RAISE EXCEPTION 'Review conflicting jabatan % (IDs: %)', office.jabatan, position_ids;
        END IF;

        IF EXISTS (
            SELECT 1 FROM vms.pejabat
            WHERE id_jabatan = position_id AND status_aktif AND nama <> office.pejabat
        ) THEN
            RAISE EXCEPTION 'Review a different active pejabat for %', office.jabatan;
        END IF;

        SELECT count(*) INTO matching_officials
        FROM vms.pejabat
        WHERE id_jabatan = position_id AND nama = office.pejabat;
        IF matching_officials = 0 THEN
            INSERT INTO vms.pejabat (id_jabatan, nama, status_aktif)
            VALUES (position_id, office.pejabat, true);
        ELSIF matching_officials <> 1 OR NOT EXISTS (
            SELECT 1 FROM vms.pejabat
            WHERE id_jabatan = position_id AND nama = office.pejabat AND status_aktif
        ) THEN
            RAISE EXCEPTION 'Review duplicate or inactive pejabat for %', office.jabatan;
        END IF;
    END LOOP;
END
$seed$;
COMMIT;
\else
SELECT deputy.nama AS kedeputian, office.direktorat, office.jabatan, office.pejabat,
       office.pejabat_vip
FROM jsonb_to_recordset(:'seed_positions'::jsonb) AS office(jabatan text, kode_kedeputian text, direktorat text, pejabat_vip boolean, pejabat text)
LEFT JOIN jsonb_to_recordset(:'seed_deputies'::jsonb) AS deputy(kode text, nama text)
  ON deputy.kode = office.kode_kedeputian
ORDER BY deputy.nama NULLS FIRST, office.direktorat NULLS FIRST, office.jabatan;

SELECT jsonb_array_length(:'seed_deputies'::jsonb) AS kedeputian,
       count(direktorat) AS direktorat,
       count(*) AS jabatan,
       count(pejabat) AS pejabat,
       count(DISTINCT pejabat) AS people
FROM jsonb_to_recordset(:'seed_positions'::jsonb) AS office(jabatan text, kode_kedeputian text, direktorat text, pejabat_vip boolean, pejabat text);
ROLLBACK;
\endif
