/** Russian plural form: plural(1, ["блюдо", "блюда", "блюд"]) = "блюдо", 4 -> "блюда", 5 -> "блюд". */
export function plural(count: number, forms: readonly [one: string, few: string, many: string]): string {
  const mod10 = count % 10;
  const mod100 = count % 100;
  if (mod10 === 1 && mod100 !== 11) return forms[0];
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return forms[1];
  return forms[2];
}
