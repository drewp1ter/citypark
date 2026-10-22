import { rooms } from "../model/rooms.data";
import type { Room, RoomAvailability, StayQuery } from "../model/types";

// Reads go through these functions only. Today they return static data;
// later they call the booking engine / CMS and nothing else has to change.

export async function getRooms(): Promise<readonly Room[]> {
  return rooms;
}

/** Availability by room id for the given stay. Sample numbers from the design until a booking engine is connected. */
export async function getRoomAvailability(query?: StayQuery): Promise<Record<string, RoomAvailability>> {
  void query; // TODO: ask the booking engine for these dates
  return {
    vip: { state: "available" },
    "single-balcony": { state: "available" },
    single: { state: "available", left: 2 },
    "double-twin": { state: "available", left: 1 },
    "double-euro": { state: "unavailable" },
  };
}

/** Rooms free tonight, summed over room types. `null` when availability is unknown. */
export async function getFreeRoomsCount(): Promise<number | null> {
  const availability = await getRoomAvailability();
  return Object.values(availability).reduce(
    (sum, room) => sum + (room.state === "available" ? (room.left ?? 1) : 0),
    0,
  );
}
