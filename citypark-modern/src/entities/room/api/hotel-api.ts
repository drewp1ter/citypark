import { serverEnv } from "@/shared/config";

export type ApiHotelRooms = {
  rooms: { vip: number; singleBalcony: number; single: number; doubleTwin: number; doubleEuro: number };
  free: number;
  updatedAt: string;
};

/** GET /hotel/rooms. `null` when the backend has no data yet (404). */
export async function fetchHotelRooms(): Promise<ApiHotelRooms | null> {
  const response = await fetch(serverEnv().MENU_API_URL + "/hotel/rooms", { next: { revalidate: 60 } });
  if (response.status === 404) return null;
  if (!response.ok) throw new Error(`GET /hotel/rooms: ${response.status}`);
  return (await response.json()) as ApiHotelRooms;
}
