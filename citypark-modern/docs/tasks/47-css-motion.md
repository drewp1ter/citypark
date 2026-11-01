# Task 47: CSS animations and hover effects

## Goal
Add subtle CSS motion to the existing public frontend.

## Depends on
46.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/app/styles/globals.css`
- `src/shared/ui/button.tsx`
- `src/shared/ui/card.tsx`

## Files to create
- `docs/tasks/47-css-motion.md`

## Files to change
- `src/app/styles/globals.css`: shared motion styles and pointer/reduced-motion guards.
- `src/shared/ui/button.tsx`: control class and disabled appearance.
- `src/shared/ui/icon-button.tsx`: control class.
- `src/shared/ui/card.tsx`: photo card class.
- `docs/tasks/PROGRESS.md`: task status.

## Spec
- Animate headings on entry and native dialogs on opening.
- Give enabled shared buttons hover lift, opacity feedback and pressed feedback.
- Zoom images inside actionable article/link cards on fine-pointer hover.
- Keep static panels steady, existing tokens, content and layout intact.
- Respect reduced motion; avoid sticky hover effects on touch devices.
- Add no runtime dependencies or client components.

## Tests to write
No application logic changes. Verify generated CSS in Chromium with normal/reduced motion and fine/coarse pointers, including disabled controls and card image behavior.

## Done when
- `npm run check`
- `npm run build`
- Browser checks of motion, reduced motion, touch hover and disabled controls pass.

## Out of scope
Backend changes, new design tokens, scroll observers and layout changes.
