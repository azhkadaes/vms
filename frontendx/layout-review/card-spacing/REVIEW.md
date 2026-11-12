# Management-card spacing — 3 November 2026

Implemented the approved option A on Peran & akses and Integrasi.

| Severity | Location | Before | After | Why |
| --- | --- | --- | --- | --- |
| LOW — resolved in inspected layouts | [styles.css:1813](D:/oikn/VMS/frontendx/apps/admin/src/styles.css:1813), [RoleManagement.tsx:134](D:/oikn/VMS/frontendx/apps/admin/src/features/RoleManagement.tsx:134), [IntegrationManagement.tsx:209](D:/oikn/VMS/frontendx/apps/admin/src/features/IntegrationManagement.tsx:209) | Cards touched their panel-heading border and preview notice. Description, permissions/status, and action buttons had no separating space. | The card grid has 24px block padding and inline insets matching the panel headings: 26px on desktop, 20px at widths up to 640px. Card paragraphs have a 12px gap, actions have 20px clearance, and protected-role hints have 16px clearance. Preview notices share the grid's leading edge. | Information and actions read as separate groups while retaining existing cards and reading order. |

Validation passed: TypeScript type check, admin production build, and 12 browser page/width combinations across the two pages at 1440, 1280, 1024, 768, 390 and 320px. Measurements checked paragraph/action gaps, heading/card clearance, matching card/notice insets, and no card or document horizontal overflow. Both role and integration editing dialogs opened. No settings were submitted. Desktop roles and phone roles/integrations were visually inspected using local demo data.

[Desktop roles](./roles-1440.png) · [Phone roles](./roles-320.png) · [Phone integrations](./integration-320.png) · [Measurements](./measurements.json)

Protected-role hints, error/empty states, production data, physical devices, and native 200% browser zoom were not exercised in this step. The protected-role hint spacing is source coverage only.

**Approve** for the inspected card-grouping coverage. RTL adaptability remains awaiting the user's scope choice.
