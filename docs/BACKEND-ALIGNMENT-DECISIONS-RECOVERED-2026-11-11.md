# Recovered VMS database-alignment decisions

Recovered 11 November 2026 from the original user selections in chat `01a1aa13-86cc-7d93-9527-80f486954239` (Verify database design compliance), using the app's read-only chat tool. This is a decision record for the new backend planning task, not proof of current implementation or new permission to migrate existing data. Decisions 1–32 establish the earlier baseline; 33 concerns legacy migration and is deferred for the fresh-database-first build.

The original diagram attachment was located at `C:/Users/ASUS/AppData/Local/Temp/codex-clipboard-5ad7e512-35e5-4f86-afcd-af4abe1e4699.png`. Its 20 business entity labels are File Storage, Pengunjung, Detail Kunjungan, Pejabat, Jabatan, Pegawai Pejabat, Pegawai, Ruangan, Role Pegawai, Audit Log, Kunjungan, Role, Role Policy, Policy, Integration Keys, QR Kunjungan, Feature, Permission, Policy Permission and Action. Field-level diagram transcription/verification remains necessary before physical DDL is frozen; locating the image is not a claim that every field/cardinality has been independently validated.

| Original decision | User's selected answer | Source turn |
| --- | --- | --- |
| 1: Which ID type should the 20 business entities use? This covers record IDs, not identity-document numbers or booking codes. | Follow the diagram: integer IDs (Recommended) | `01a1aae9-65cb-7c52-ae3c-582f2bdf17a5` |
| 2: How should visitor and visit details be stored? | Follow the diagram: separate Pengunjung, Kunjungan, and Detail Kunjungan tables (Recommended) | `01a1aaf1-a1fa-7c62-8980-caded3164046` |
| 3: Which booking-code length should we follow? | Keep the current eight-digit code as an approved exception | `01a1aaf2-d09d-7540-97a4-237d5613f78c` |
| 4: How should role permissions be stored? | Follow the diagram: policies and linked permission tables (Recommended) | `01a1ae5b-0931-7b23-9e45-d9d07fe779ec` |
| 5: Which employee-role assignment model should we use? | Use Role Pegawai and allow multiple roles per employee | `01a1ae60-27f4-7b80-907e-1e608656bdae` |
| 6: How should an employee’s multiple roles work during use? | Use one selected active role at a time, with a role switcher (Recommended) | `01a1ae60-c0f6-7ce1-83d0-d9dc3752099e` |
| 7: Which structure should we use for officials’ positions and VIP status? | Follow the diagram: Pejabat links to Jabatan, which determines VIP status (Recommended) | `01a1ae61-6b9b-7322-ada2-3d6d32117054` |
| 8: How should visits reference rooms? This choice covers room storage and relationships; room-assignment timing comes next. | Follow the diagram: Ruangan records linked by integer foreign key, with the diagram’s room fields (Recommended) | `01a1ae62-7c14-7c61-a7f1-20fe8be7adcb` |
| 9: When must a room be assigned? | Preserve current rules, including staff-created approved visits without a room (approved exception) | `01a1ae64-45a7-7751-b217-0c29b0ccba74` |
| 10: Should ordinary bookings collect the representative’s identity-document number? | Keep identity type only; leave the number uncollected and nullable (Recommended exception) | `01a1ae65-25af-7743-8f40-0dfb1fe0b470` |
| 11: Should ordinary bookings require a supporting file? | Keep supporting files optional, with a nullable file link (Recommended exception) | `01a1ae67-4890-7170-9e72-8fc31545d63c` |
| 12: Where should the optional supporting-file relationship be stored? | Keep files linked directly to each visit, as an approved exception | `01a1ae68-481f-7ff3-ae92-3b6e17619c2c` |
| 13: Should restricted visits require a visitor record and target official? | Keep visitor identity and target official optional for restricted visits (Recommended exception) | `01a1ae69-af79-7872-8295-4efb06e45e04` |
| 14: Should bookings retain the names of accompanying visitors? | Keep the named roster and derive the visitor count from it (Recommended extension) | `01a1ae6b-8302-7a03-ac34-a50f24c82086` |
| 15: Which audit-log detail should we retain? | Follow the diagram: descriptions, IP addresses, and filtered before/after values (Recommended) | `01a1ae6d-1f72-7da2-9ef6-39ff5c27c3bb` |
| 16: How should audit logs record public and background-system actions? | Use one log with actor type and a nullable employee link for public/system events (Recommended exception) | `01a1ae6e-96c8-7091-8b0b-47b260338d05` |
| 17: How should QR expiry be stored? | Keep the exact expiry date and time, including timezone (Recommended exception) | `01a1ae70-64d4-7cd2-8f0a-da3228e06567` |
| 18: Which employee-profile fields should Pegawai contain? | Restore the diagram’s jabatan, kedeputian, direktorat, phone, and NIP fields as optional administrative fields (Recommended) | `01a1ae71-685e-7f81-b2c2-c750ec755b40` |
| 19: How should the visit purpose be recorded? | Keep the current free-text purpose field (Recommended exception) | `01a1ae71-d74e-7761-8569-e49e96ba0c70` |
| 20: Which visit-schedule storage model should we use? | Keep the current start timestamp plus duration as an approved exception | `01a1ae73-f56d-7612-b6b9-81de58f3d10e` |
| 21: Where should check-in and checkout details be stored? | Keep separate attendance-event records and derive visit attendance details (Recommended exception) | `01a1ae76-4efd-77d3-8edb-3184da3c804b` |
| 22: Which integration-storage model should we use? | Keep the current provider settings and encrypted-secrets model (Recommended exception) | `01a1ae78-5b73-7cf0-a294-a662c34d16de` |
| 23: Where should the additional operational tables live? Both choices retain them to support existing backend functions. | Put operational tables in a separate support schema (Recommended) | `01a1ae7b-b253-77f3-b958-aa865da4518f` |
| 24: How should visit status be stored? Both choices preserve the current six statuses and lifecycle rules. | Follow the diagram: use a database ENUM with the current six statuses (Recommended) | `01a1ae7c-b0ab-7672-b83e-0f21a5c8dd54` |
| 25: How should File Storage record file size? | Keep file size as an integer number of bytes (Recommended exception) | `01a1ae7d-1593-7832-a2f9-e8cd85e53fd0` |
| 26: How should we interpret the duplicated catatan fields in Detail Kunjungan? | Use two distinct note fields; define their purposes and visibility next | `01a1ae83-bdc1-7b00-98ab-da6e75de2dec` |
| 27: What should the two visit-note fields mean, and who should see them? | Visitor-visible instructions + a separate staff-only coordination note | `01a1ae85-6697-74e1-a4bb-327067ab4861` |
| 28: Should each permission area—such as Kunjungan or Pegawai—have just a name, or also a description? | Name only, such as Kunjungan (Recommended) | `01a1ae88-ac08-7b33-b0a0-3b915afc8dc8` |
| 29: How should staff-to-official assignments be removed? | Follow the diagram: retain assignment records and mark them inactive using status_aktif (Recommended) | `01a1ae8b-1bcb-7690-924c-c5499b6575bc` |
| 30: How should QR revocation be recorded? | Follow the diagram: store a nullable revocation timestamp, dicabut_pada (Recommended) | `01a1ae8b-d22e-7bc0-b4f6-9fc384a65d4e` |
| 31: How should File Storage identify the stored file’s location? | Keep deriving the private file location from its file ID, as an approved exception | `01a1ae8d-c2ca-7ba0-be60-33f8774304ae` |
| 32: How should integer File Storage IDs coexist with derived private filenames? | Use an integer database ID plus a separate UUID object ID for private filenames (Recommended extension) | `01a1ae8f-fd2d-7df1-8757-405107800cc3` |
| 33: How should we migrate QR credentials that were already revoked? The old database stored only a revoked flag, so their original revocation time is unknown. Both options keep them unusable. | Archive revoked credentials; import only unrevoked credentials (Recommended) | `01a1be2a-8137-7ba0-8d07-5d965c8b77af` |

Later UI selections, recovered from the same chat: mandatory selection before workspace access for employees with multiple roles; switch-role dropdown in the sidebar; discard-or-stay confirmation for unsaved changes; searchable checkbox/tag role picker; optional profile fields in an expandable section; both note fields on staff creation; visitor instructions on visitor details and notification emails; note edits only during approval/rescheduling/restricted editing. Current frontend code and contract addendum independently support these requirements. Detailed UI implementation is outside this planning task.


Field reading update: the transparent source diagram has now been rendered on white for inspection, allowing all 20 entity field lists to be transcribed in the [schema evidence register](BACKEND-SCHEMA-EVIDENCE-2026-11-11.md). This improves source evidence; semantic/cardinality/constraint reconciliation remains unfinished and is not approval of DDL.
