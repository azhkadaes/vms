\set ON_ERROR_STOP on
BEGIN;
DO $vms$
BEGIN
  IF current_database() <> 'vms_stage1' THEN
    RAISE EXCEPTION 'Synthetic seed is restricted to vms_stage1';
  END IF;
  IF EXISTS (SELECT FROM vms.jabatan) OR EXISTS (SELECT FROM vms.pejabat) OR EXISTS (SELECT FROM vms.ruangan) THEN
    RAISE EXCEPTION 'Catalog already contains data; inspect it instead of reseeding';
  END IF;
END
$vms$;
INSERT INTO vms.jabatan(nama,pejabat_vip) VALUES ('STAGE1 TEST - VIP position',TRUE);
INSERT INTO vms.pejabat(id_jabatan,nama,status_aktif)
SELECT id,'STAGE1 TEST - Active official',TRUE FROM vms.jabatan WHERE pejabat_vip;
INSERT INTO vms.pejabat(id_jabatan,nama,status_aktif)
SELECT id,'STAGE1 TEST - Inactive official',FALSE FROM vms.jabatan WHERE pejabat_vip;
-- A non-VIP official is the only kind a PA may be assigned to (A-16).
INSERT INTO vms.jabatan(nama,pejabat_vip) VALUES ('STAGE1 TEST - Standard position',FALSE);
INSERT INTO vms.pejabat(id_jabatan,nama,status_aktif)
SELECT id,'STAGE1 TEST - Standard official',TRUE FROM vms.jabatan WHERE NOT pejabat_vip;
-- vms.ruangan has no ruangan column since the per-table rebuild; lokasi stays private to the public catalog.
INSERT INTO vms.ruangan(nama,lokasi,status_aktif)
VALUES ('STAGE1 TEST - Active room','private-test-location',TRUE),
       ('STAGE1 TEST - Inactive room',NULL,FALSE);
COMMIT;
