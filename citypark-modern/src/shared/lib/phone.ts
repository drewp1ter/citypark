const MAX_DIGITS = 11;

/** Only the digits of a Russian phone, starting with 7: "+7 918 311-97-91" -> "79183119791". */
export function phoneDigits(value: string): string {
  let digits = value.replace(/\D/g, "");
  if (digits.startsWith("8")) digits = `7${digits.slice(1)}`;
  else if (digits && !digits.startsWith("7")) digits = `7${digits}`;
  return digits.slice(0, MAX_DIGITS);
}

/** A full Russian number: 11 digits, country code 7. */
export function isValidRuPhone(value: string): boolean {
  return phoneDigits(value).length === MAX_DIGITS;
}

/**
 * Formats what the user types as "+7 918 311-97-91" while they type.
 *
 * TODO(you): this version always formats the whole string, so if someone edits a digit in the MIDDLE of the number
 * the cursor jumps to the end. Decide how much that matters: leave it, or keep the caret position
 * (PhoneField passes the input element to this function's caller, so you have `selectionStart` available there).
 */
export function formatPhoneInput(value: string): string {
  const digits = phoneDigits(value);
  if (!digits) return "";

  const rest = digits.slice(1); // after the country code
  const parts = [rest.slice(0, 3), rest.slice(3, 6), rest.slice(6, 8), rest.slice(8, 10)];
  let formatted = "+7";
  if (parts[0]) formatted += ` ${parts[0]}`;
  if (parts[1]) formatted += ` ${parts[1]}`;
  if (parts[2]) formatted += `-${parts[2]}`;
  if (parts[3]) formatted += `-${parts[3]}`;
  return formatted;
}
