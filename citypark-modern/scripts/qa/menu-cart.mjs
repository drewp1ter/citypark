// Cart lines for the QA scripts, built from real products of the backend (the ids are not known in advance).
// Usage: const cart = await seedCart([1, 2]);  // two lines: 1 and 2 portions
// Reads MENU_API_URL (default http://localhost:3000/api/v1).
export async function seedCart(quantities) {
  const api = (process.env.MENU_API_URL ?? "http://localhost:3000/api/v1").replace(/\/+$/, "");
  const get = async (path) => {
    const response = await fetch(api + path);
    if (!response.ok) throw new Error(`GET ${path}: ${response.status}`);
    return response.json();
  };
  // GET /products also returns products of hidden categories; the site does not show them.
  const visible = new Set((await get("/categories")).map((c) => c.id));
  const products = (await get("/products?pageSize=500")).filter(
    (p) => p.isAvailableForOrder && p.price > 0 && visible.has(p.categoryId),
  );
  if (products.length < quantities.length) throw new Error(`Need ${quantities.length} orderable products, got ${products.length}`);
  return quantities.map((quantity, i) => ({ dishId: String(products[i].id), variantId: "default", quantity }));
}
