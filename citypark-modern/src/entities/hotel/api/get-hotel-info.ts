import { guestInfo, quickFacts, roomAmenities, rules } from "../model/hotel.data";

// Reads go through these functions only; swap the bodies when a CMS appears.

export async function getRoomAmenities() {
  return roomAmenities;
}

export async function getQuickFacts() {
  return quickFacts;
}

export async function getHotelRules() {
  return rules;
}

export async function getGuestInfo() {
  return guestInfo;
}
