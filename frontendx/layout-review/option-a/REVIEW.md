# Option A implementation — 3 November 2026

Approved choice: reuse the existing cards when the booking content area becomes too narrow.

| Severity | Location | Before | After | Why |
| --- | --- | --- | --- | --- |
| HIGH — resolved in tested coverage | [Bookings.tsx:268](D:/oikn/VMS/frontendx/apps/admin/src/pages/Bookings.tsx:268), [styles.css:574](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:574) | The viewport breakpoint kept tables and scrolling status tabs visible inside a narrow content area. At 768px the PA table needed 699px inside 498px; status and detail actions were off-screen. | The booking panel is a named inline-size container. At a content width of 900px or less, the existing cards and labeled status selector replace the table and status tabs. Search and sort wrap within the same container. Wider panels retain the existing table. | Adaptation follows the actual available space, including sidebar changes. Status and detail actions stay within each card. |

The 900px boundary accommodates the existing complete status-tab row as well as the table. It applies to both Janji temu and Kunjungan hari ini. Room information remains available in visit details, as in the existing phone cards.

Validation passed:

- TypeScript type check and admin production build.
- 96 browser combinations: PA, Protocol Officer and Receptionist; booking and today pages; widths 1440, 1280, 1150, 1024, 900, 768, 390 and 320px; sidebar expanded and collapsed.
- One list presentation and one status-filter presentation visible in each combination. No document horizontal overflow, no horizontal overflow in visible table/tab regions, and no horizontally clipped card actions.
- Sidebar keyboard expansion/collapse at the same 1150px viewport switches between cards and table without reloading.
- Status filtering, name sorting, search and visit-detail opening work through cards for all three roles and both sidebar starting states. Pagination moves to the next page and back where sample data supplies multiple pages.
- Populated browser checks use an isolated sample-data service. No production API or database verification is implied.

Evidence: [tablet cards](booking-768-expanded.png), [desktop table](booking-1280-expanded.png), [phone](booking-320-expanded.png), [measurements](measurements.json).

Other review items await the user's choices: visible sort labels, mobile input typography, management-card grouping, and RTL alignment. Native 200% browser zoom and physical-device behavior remain **Not verified**.

**Approve** for the tested option A layout coverage. This does not approve the untested states or the remaining review items.
