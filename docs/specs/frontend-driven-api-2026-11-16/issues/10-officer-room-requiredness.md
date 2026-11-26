# Require an officer and allow an optional room for every visit

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Question

Does the officer requirement include restricted visits, and is room optional even at approval?

## Answer

17 November 2026: user explicitly confirmed all visits require an officer, including Terbatas, and room is always optional, including approval. This supersedes the observed optional restricted officer and mandatory ordinary approval room rules. Keep detail_kunjungan.id_pejabat NOT NULL; allow id_ruangan NULL. Future frontend validation must require restricted officer and allow blank room for ordinary approval/rescheduling. No application/database edits authorized.
