# Task 66: Yandex Webmaster runbook

## Goal
Give the owner concrete instructions for registering the production site later, including verification, indexing and Alice visibility checks.

## Depends on
62; preparation through 65 is complete.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/SEARCH_DISCOVERY.md`, `.env.example`, `app/robots.ts`.

## Files to create
- `docs/YANDEX_WEBMASTER.md`: owner runbook and pending operational checklist.

## Files to change
- `docs/SEARCH_DISCOVERY.md`: link the runbook and record documentation completion.
- `docs/tasks/PROGRESS.md`: task entry and validation results.

## Spec
Write instructions for https://cityparkvip.ru, the canonical production origin. Explain production/staging indexing flags and required public checks. Document manual Yandex account sign-in, adding the correct HTTPS/non-www site, downloading the genuine HTML verification file, deploying it unchanged from public/, checking the response and confirming ownership. Retain the file across releases. Mention DNS TXT and meta-tag alternatives without implementing them. Document sitemap submission, troubleshooting, indexing inspection and the Alice visibility report, with links to current official Yandex documentation. Use exact existing frontend routes/configuration. Do not create a placeholder verification file/token, change deployment settings, register the site or submit it to Webmaster. Mark every operational step pending; distinguish runbook completion from future deployment/registration acceptance (67).

## Tests to write
Documentation only; no new executable logic or test fixtures. Review local links and commands against repository configuration and official Yandex documentation.

## API documentation
No API changes.

## Done when
`npm run check`, `npm run build` and `git diff --check` pass. The runbook covers ownership verification, production noindex troubleshooting, sitemap submission, indexing and Alice visibility; all operational statuses remain pending.

## Out of scope
Live deployment, account registration, adding the real verification artifact, DNS changes, Webmaster submissions and deployed-site acceptance (67).

## Results
Official Yandex references and repository paths/configuration reviewed; local Markdown links resolve. `npm run check`, `npm run build` and `git diff --check` passed. Existing dynamic-rendering diagnostics remain during build. All operational checklist entries remain Pending.
