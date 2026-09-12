/**
 * Phone canonicalization — exact port of the backend `internal/phone`
 * package. All numbers are stored and compared as 01XXXXXXXXX (11 digits).
 */
const nonDigit = /\D/g;

export function canonicalizePhone(input: string): string {
  let digits = input.trim().replace(nonDigit, "");
  if (digits.startsWith("8801")) {
    digits = "0" + digits.slice(3);
  } else if (digits.startsWith("880")) {
    digits = digits.slice(3);
  }
  if (digits.length === 11 && digits.startsWith("01")) {
    return digits;
  }
  return "";
}

export function isValidPhone(input: string): boolean {
  return canonicalizePhone(input) !== "";
}

/** Display form: 01XXX-XXXXXX */
export function formatPhoneDisplay(canonical: string): string {
  if (canonical.length !== 11) return canonical;
  return `${canonical.slice(0, 5)}-${canonical.slice(5)}`;
}
