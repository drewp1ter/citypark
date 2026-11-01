# Task 48: Mobile cart opening and closing animations

## Goal
Slide the mobile cart up on opening and down on closing, with a fading backdrop.

## Depends on
47.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/widgets/cart-summary/ui/cart-sheet.tsx`
- `src/widgets/cart-summary/ui/cart-bar.tsx`
- `src/app/styles/globals.css`

## Files to create
- `docs/tasks/48-mobile-cart-motion.md`
- `scripts/qa/flows/cart-motion.mjs`

## Files to change
- `src/widgets/cart-summary/ui/cart-sheet.tsx`: scoped CSS class.
- `src/app/styles/globals.css`: sheet/backdrop transitions.
- `docs/tasks/PROGRESS.md`: completion status.

## Spec
- Use native dialog discrete display/overlay transitions to slide the sheet 100% down and fade its backdrop over 260 ms.
- Close button, backdrop, Escape and checkout links retain native behavior without delayed navigation or JS timers.
- Restore focus and support repeated opening/closing.
- Reduced motion uses immediate opening/closing.
- Browsers lacking discrete overlay transitions retain a slide-in animation and immediate native close.
- Preserve Mini App insets and desktop behavior.

## Tests to write
Production-browser flow: normal/reduced motion in both themes on mobile, opening and closing transitions, repeated opening, Escape, backdrop, focus return and checkout navigation. No backend writes.

## Done when
- `npm run check`
- `npm run build`
- `node scripts/qa/flows/cart-motion.mjs` against the production build.

## Out of scope
Other dialogs, cart calculations and backend changes.
