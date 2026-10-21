import { z } from "zod";
import { cartReducer, type CartAction } from "./cart-reducer";
import type { CartLine } from "./types";

// A tiny external store for useSyncExternalStore: the cart lives in localStorage, so it survives reloads
// and is shared between tabs. Every component reads the same state, so no provider is needed.

const STORAGE_KEY = "citypark-cart";
const EMPTY: readonly CartLine[] = [];

const linesSchema = z.array(
  z.object({ dishId: z.string(), variantId: z.string(), quantity: z.number().int().positive() }),
);

const listeners = new Set<() => void>();

// getSnapshot must return the same array until the cart really changes, so the parsed value is cached by its raw string.
let cachedRaw: string | null = null;
let cachedLines: readonly CartLine[] = EMPTY;
// Used when localStorage is blocked (private mode, full quota): the cart then lives until the page is reloaded.
let memoryLines: readonly CartLine[] | null = null;

function parse(raw: string | null): readonly CartLine[] {
  if (!raw) return EMPTY;
  try {
    const result = linesSchema.safeParse(JSON.parse(raw));
    return result.success && result.data.length > 0 ? result.data : EMPTY;
  } catch {
    return EMPTY;
  }
}

export function getCartSnapshot(): readonly CartLine[] {
  if (memoryLines) return memoryLines;
  let raw: string | null;
  try {
    raw = localStorage.getItem(STORAGE_KEY);
  } catch {
    return cachedLines;
  }
  if (raw !== cachedRaw) {
    cachedRaw = raw;
    cachedLines = parse(raw);
  }
  return cachedLines;
}

/** Server and first client render: the cart is always empty, so hydration matches. */
export function getServerCartSnapshot(): readonly CartLine[] {
  return EMPTY;
}

export function subscribeToCart(listener: () => void): () => void {
  listeners.add(listener);
  // "storage" fires in other tabs when the cart changes there.
  const onStorage = (event: StorageEvent) => {
    if (event.key === STORAGE_KEY || event.key === null) listener();
  };
  window.addEventListener("storage", onStorage);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", onStorage);
  };
}

export function dispatchCart(action: CartAction): void {
  const next = cartReducer(getCartSnapshot(), action);
  const raw = next.length > 0 ? JSON.stringify(next) : null;

  try {
    if (raw) localStorage.setItem(STORAGE_KEY, raw);
    else localStorage.removeItem(STORAGE_KEY);
    memoryLines = null;
    cachedRaw = raw;
    cachedLines = next.length > 0 ? next : EMPTY;
  } catch {
    memoryLines = next;
  }

  listeners.forEach((listener) => listener());
}
