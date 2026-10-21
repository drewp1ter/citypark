import type { MetadataRoute } from "next";

// The order pages are for visitors only. TODO: add `sitemap` when the production domain is known.
export default function robots(): MetadataRoute.Robots {
  return { rules: { userAgent: "*", allow: "/", disallow: ["/checkout"] } };
}
