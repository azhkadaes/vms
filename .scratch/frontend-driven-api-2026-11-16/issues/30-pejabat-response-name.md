# Use pejabat.nama in booking responses

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Answer

The user chose “Pejabat” as the UI label and `pejabat.nama` as the database and booking response name. Express the response as a nested `pejabat` object with a string `nama`. Apply this wherever a booking projection previously exposed the official's name. Keep existing projection restrictions, request `officerId`, separate catalog and report models, and sort keys.
