# Register City Park with Yandex Webmaster

Prepared on 2026-11-05 for **https://cityparkvip.ru**. Registration is an owner action to perform later. This document does not record a completed deployment or registration. Follow the stages below in order when the new frontend is ready to launch.

## 1. Prepare the production deployment

Deploy the intended frontend and confirm that its hotel, menu, contacts and FAQ pages show the correct public business information. Use HTTPS without www: this matches `productionOrigin` in `src/shared/config/search.ts`. Route alternative HTTP/www addresses to the intended production address without loops. If the intended domain changes, update canonical configuration before registering it.

Set `SITE_INDEXING_ENABLED=true` in the production build and runtime environment, then rebuild and deploy. Keep it false on staging. In the returned public-page HTML, verify that the robots meta tag permits indexing; private account, checkout and Mini App pages must still emit noindex. Check reverse-proxy/CDN headers for an overriding `X-Robots-Tag: noindex`. Changing the environment alone is not proof that the deployed HTML is correct.

Expected public endpoints:

| URL | Expected result |
|---|---|
| https://cityparkvip.ru/ | The new homepage, HTTP 200, production canonical, indexable HTML |
| https://cityparkvip.ru/contacts | Current contacts, FAQ and Hotel/Restaurant JSON-LD |
| https://cityparkvip.ru/robots.txt | Crawl allowed; `Sitemap: https://cityparkvip.ru/sitemap.xml` |
| https://cityparkvip.ru/sitemap.xml | HTTP 200 and XML containing public URLs on the production domain |

For a quick response check, run these read-only commands:

```bash
curl --silent --show-error --dump-header - --output /dev/null https://cityparkvip.ru/
curl --fail --silent --show-error https://cityparkvip.ru/robots.txt
curl --fail --silent --show-error https://cityparkvip.ru/sitemap.xml
```

Use browser **View page source** to inspect the robots and canonical tags. Do not infer indexing eligibility from `robots.txt` alone: this project's robots configuration allows crawling even when page metadata says noindex.

For fuller checks during task 67, run from `citypark-modern/` with `MENU_API_URL` pointing to the backend used by that production deployment and reachable from your machine:

```bash
BASE_URL=https://cityparkvip.ru node scripts/qa/sitemap-http.mjs
BASE_URL=https://cityparkvip.ru node scripts/qa/public-content-http.mjs
BASE_URL=https://cityparkvip.ru node scripts/qa/business-json-ld-http.mjs
BASE_URL=https://cityparkvip.ru node scripts/qa/customer-questions.mjs
```

These checks read public/backend data; the FAQ check also saves screenshots locally. A local development backend is not a valid comparison source for production. Preserve any pending checks if the production backend cannot be reached.

## 2. Add the site manually

Sign in at [Yandex Webmaster](https://webmaster.yandex.ru/) with the account that should own the site. Select **Добавить сайт**, enter **https://cityparkvip.ru**, then add it. Check the scheme and hostname before proceeding. Follow Yandex's [getting-started instructions](https://yandex.ru/support/webmaster/ru/service/quick-start) if the interface has changed.

## 3. Verify ownership with the genuine HTML file

On the access-rights screen, select the HTML-file method. Download the file Yandex supplies, preserving its exact filename and contents. Put that actual file directly in **citypark-modern/public/**; do not wrap it in a React page or add site markup. No verification token or sample file has been added by this task.

Build and deploy that file with the frontend. The public URL is the production origin followed by `/` and its filename. If deployment uses Next.js standalone output, ensure the deployment also ships `public/`; the standalone server bundle alone is insufficient to deliver that file.

From `citypark-modern/`, compare the deployed response with the local artifact:

```bash
read -r -p 'Exact downloaded verification filename: ' verification_file
curl --fail --silent --show-error --output /tmp/citypark-yandex-verification.html --write-out '%{http_code}\n' "https://cityparkvip.ru/$verification_file"
cmp "public/$verification_file" /tmp/citypark-yandex-verification.html
```

Expected: HTTP **200**, and `cmp` exits successfully without differences. A redirect or a normal site page is not the expected verification artifact. Return to Webmaster and confirm ownership. Keep the file in source/deployment artifacts across later releases because ownership can be checked again. DNS TXT and a homepage meta tag are alternatives if the file method cannot be used; follow the exact values Yandex provides. See [ownership verification](https://yandex.ru/support/webmaster/ru/service/rights.html).

## 4. Submit the sitemap

Select the verified site, open **Индексирование → Файлы Sitemap**, and add **https://cityparkvip.ru/sitemap.xml**. Inspect the processing status and any reported errors. Updates to the existing sitemap do not require deleting and re-adding its URL. See [Yandex's sitemap instructions](https://yandex.ru/support/webmaster/ru/indexing-options/sitemap.html).

Our sitemap contains the public static pages and current room/news details. It excludes account, authentication, checkout/payment, Mini App and API routes. Its size changes with backend content; do not use the previous local URL count as a production requirement. The sitemap endpoint deliberately fails if required backend lists fail instead of publishing an incomplete successful list.

## 5. Inspect indexing

Use **Индексирование → Проверка страницы** for the homepage, contacts, menu, hotel and representative room/news details. Use **Страницы в поиске** to inspect inclusion and exclusion reasons. For newly deployed or corrected public pages, use **Переобход страниц** where appropriate. These tools are listed in Yandex's [indexing help](https://yandex.ru/support/webmaster/ru/service/searchable).

Check actionable failures before requesting another crawl:

| Symptom | Check |
|---|---|
| Ownership fails | Exact file bytes/name, public HTTP 200, correct host, public/ present in deployment, proxy/CDN behavior |
| Robot cannot reach the site | Hosting/firewall/bot restrictions; use Webmaster's server-response tool, and check IPv4/IPv6 if both are configured |
| Public pages excluded by noindex | Production build/runtime flag, emitted HTML and proxy/CDN X-Robots-Tag |
| Sitemap error or non-200 | Deployment routing, XML response and the configured backend's room/news endpoints |
| Different canonical chosen | Production origin and host redirects; menu category query URLs currently canonicalize to /menu by design |
| Old information remains | Confirm the deployed content first, then inspect crawl status and request recrawl |

Do not request indexing of private routes. Registration, verification and sitemap processing do not guarantee search inclusion. [Yandex explicitly distinguishes registration from appearing in search](https://yandex.ru/support/webmaster/ru/service/quick-start).

## 6. Record Alice visibility

Open **Эффективность → Видимость сайта в Алисе AI**. Record the selected date range, available mention/share metrics and example queries. Keep a dated screenshot for comparison after later content changes. If data is absent, record that status rather than claiming zero citations. The report covers a defined set of search queries; it is not a complete inventory of all Alice answers. See [Alice visibility documentation](https://yandex.ru/support/webmaster/ru/service/alice-answers).

This is monitoring, not a separate registration or a guaranteed recommendation. Useful manual examples to investigate later are «Сити Парк Белореченск адрес», «Сити Парк Белореченск парковка» and «Сити Парк Белореченск доставка». Record the date, query and cited source if a relevant answer appears.

## Operational checklist — all pending

Task 66 completes this runbook only. Fill in evidence during deployment/registration and task 67; do not tick a step because its instructions exist.

| Step | Status | Evidence to record |
|---|---|---|
| Intended frontend deployed on production HTTPS | Pending | Deployment commit/date and URL |
| Public indexability and private noindex checked | Pending | HTML/header check results |
| Webmaster site added under owner's account | Pending | Exact site URL and date |
| Genuine verification file deployed | Pending | Filename, HTTP result and byte comparison |
| Ownership verified | Pending | Webmaster status/date |
| Sitemap submitted and processed | Pending | Status/date and errors, if any |
| Representative pages inspected | Pending | URLs and inclusion/exclusion reasons |
| Alice visibility baseline recorded | Pending | Date range and available metrics/examples, or no-data status |

Related: [search preparation and results](SEARCH_DISCOVERY.md), [customer content gaps](CUSTOMER_QUESTIONS.md).
