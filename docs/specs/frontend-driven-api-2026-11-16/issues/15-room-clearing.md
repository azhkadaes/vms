# Preserve omitted rooms and clear explicit nulls

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Question

How do omitted roomId and explicit null differ in creation and existing-visit mutations?

## Answer

17 November 2026: user confirmed omitted roomId preserves the existing room on edit; explicit null clears it. On creation, omitted or null means no room. Apply consistently to existing-visit schedule commands, including approval, rescheduling and restricted update. A supplied ID replaces the selection after validation. No application changes authorized.
