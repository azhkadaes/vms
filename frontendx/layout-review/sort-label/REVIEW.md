# Visible sort label — 3 November 2026

Implemented the approved option A on Janji temu and Kunjungan hari ini.

| Severity | Location | Before | After | Why |
| --- | --- | --- | --- | --- |
| MEDIUM — resolved | [Bookings.tsx:308](D:/oikn/VMS/frontendx/apps/admin/src/pages/Bookings.tsx:308), [styles.css:854](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:854) | The default sort selection had no visible text or label. | A visible “Urutkan” label is associated with the select; the default option reads “Urutan bawaan”. The toolbar aligns the search, select and VIP control along their lower edge. The label stays visible after selecting another order. | Users can identify the control's purpose and current state in both table and card layouts. |

Validation: existing sort browser test updated and passed, including restoring default order and sort persistence on reload. TypeScript and admin production build passed. An additional browser inspection passed 36 combinations across PA, Protocol Officer and Receptionist, both list pages, and widths 1440, 1280, 1024, 768, 390 and 320px. Label placement, horizontal overflow, same-row lower-edge alignment, default text, and selecting/restoring sort were checked. Browser data was isolated sample data.

[Desktop preview](booking-1440.png) · [Tablet preview](booking-768.png) · [Phone preview](booking-320.png) · [Measurements](measurements.json)

Native browser zoom, physical devices, and the remaining input-typography, management-card spacing and RTL findings were not verified or changed in this step.

**Approve** for the inspected sort-label coverage.
