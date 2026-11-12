export const emailDomains = [
  "gmail.com",
  "outlook.com",
  "yahoo.com",
  "hotmail.com",
  "icloud.com",
  "ikn.go.id",
] as const;

export function splitEmail(input: string) {
  const separator = input.indexOf("@");
  return separator < 0
    ? { username: input, domain: "" }
    : {
        username: input.slice(0, separator),
        domain: input.slice(separator + 1),
      };
}

export function joinEmail(username: string, domain: string) {
  return domain ? `${username}@${domain.toLowerCase()}` : username;
}

export function phoneDigits(input: string) {
  return input.replace(/[^0-9]/g, "");
}
