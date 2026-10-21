import { addDays, diffDays, todayInMoscow } from "@/shared/lib";

export const roomTypes = ["all", "single", "double", "balcony"] as const;
export type RoomType = (typeof roomTypes)[number];

const guestOptions = ["1", "2", "3-4"] as const;
const ISO_DATE = /^\d{4}-\d{2}-\d{2}$/;

type RawParams = Record<string, string | string[] | undefined>;
const first = (value: string | string[] | undefined) => (Array.isArray(value) ? value[0] : value);

export type RoomsQuery = {
  /** Stay to price and check availability for. Today and tomorrow when the address has no dates. */
  checkIn: string;
  checkOut: string;
  nights: number;
  guests: (typeof guestOptions)[number];
  type: RoomType;
  /** Only rooms that are free for the stay. */
  freeOnly: boolean;
  /** What the address bar actually contained, to keep it when a filter chip is clicked. */
  given: { checkIn?: string; checkOut?: string; guests?: string };
};

/** Reads /rooms?checkIn=…&checkOut=…&guests=…&type=…&free=1. Anything invalid is ignored, never an error. */
export function parseRoomsQuery(raw: RawParams): RoomsQuery {
  const rawIn = first(raw.checkIn);
  const rawOut = first(raw.checkOut);
  const rawGuests = first(raw.guests);
  const rawType = first(raw.type);

  const today = todayInMoscow();
  const givenIn = rawIn && ISO_DATE.test(rawIn) && rawIn >= today ? rawIn : undefined;
  const checkIn = givenIn ?? today;
  const givenOut = rawOut && ISO_DATE.test(rawOut) && rawOut > checkIn ? rawOut : undefined;
  const checkOut = givenOut ?? addDays(checkIn, 1);
  const guests = guestOptions.find((option) => option === rawGuests);

  return {
    checkIn,
    checkOut,
    nights: Math.max(1, diffDays(checkIn, checkOut)),
    guests: guests ?? "2",
    type: roomTypes.find((type) => type === rawType) ?? "all",
    freeOnly: first(raw.free) === "1",
    given: { checkIn: givenIn, checkOut: givenOut, guests },
  };
}

/** /rooms address that keeps the dates and guests and changes the filter. */
export function roomsHref(query: RoomsQuery, change: { type?: RoomType; freeOnly?: boolean }): string {
  const type = change.type ?? query.type;
  const freeOnly = change.freeOnly ?? query.freeOnly;

  const params = new URLSearchParams();
  if (query.given.checkIn) params.set("checkIn", query.given.checkIn);
  if (query.given.checkOut) params.set("checkOut", query.given.checkOut);
  if (query.given.guests) params.set("guests", query.given.guests);
  if (type !== "all") params.set("type", type);
  if (freeOnly) params.set("free", "1");

  const search = params.toString();
  return search ? `/rooms?${search}` : "/rooms";
}
