-- +migrate Up
ALTER TABLE vms.pegawai
    ALTER COLUMN nomor_hp TYPE TEXT[]
        USING CASE WHEN nomor_hp IS NULL OR btrim(nomor_hp) = '' THEN ARRAY[]::TEXT[] ELSE ARRAY[nomor_hp::TEXT] END,
    ALTER COLUMN nomor_hp SET DEFAULT ARRAY[]::TEXT[],
    ALTER COLUMN nomor_hp SET NOT NULL;

-- +migrate Down
ALTER TABLE vms.pegawai ADD CONSTRAINT pegawai_single_phone CHECK (cardinality(nomor_hp) <= 1);
ALTER TABLE vms.pegawai DROP CONSTRAINT pegawai_single_phone;
ALTER TABLE vms.pegawai ALTER COLUMN nomor_hp DROP DEFAULT;
ALTER TABLE vms.pegawai
    ALTER COLUMN nomor_hp DROP NOT NULL,
    ALTER COLUMN nomor_hp TYPE VARCHAR(255) USING nomor_hp[1];
