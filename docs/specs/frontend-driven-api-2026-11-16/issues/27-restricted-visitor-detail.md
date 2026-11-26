# Always create visitor details with visit-type validation

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Answer

User confirmed every visit, including Terbatas, has one detail_pengunjung record and a required FK. Validate fields according to visit type: ordinary visits retain representative/contact/identity-type/roster/photo requirements; restricted visits may omit fields not required by their form. Do not insert fake values to satisfy ordinary requirements. No schema migration or application change executed.
