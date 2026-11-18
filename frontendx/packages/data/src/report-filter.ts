import { z } from "zod";

export function multipleFilter<T extends z.ZodType<string>, U extends z.ZodType<string>>(
  single: T,
  option: U,
) {
  return z.union([
    single,
    z.array(option).min(2).refine((values) => new Set(values).size === values.length),
  ]);
}

export function selectedFilters<T extends string>(value: T | T[], empty: string): T[] {
  return Array.isArray(value) ? value : value === empty ? [] : [value];
}

export function matchesFilter<T extends string>(value: T, filter: T | T[], empty: string) {
  return Array.isArray(filter) ? filter.includes(value) : filter === empty || filter === value;
}

export function readFilter(
  fields: Pick<URLSearchParams, "getAll"> | Pick<FormData, "getAll">,
  name: string,
  fallback: string | string[],
) {
  const values = fields.getAll(name);
  return values.length > 1 ? values : (values[0] ?? fallback);
}

export function reportFilterParams(fields: Record<string, string | string[]>) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(fields))
    for (const item of Array.isArray(value) ? value : [value]) params.append(key, item);
  return params;
}
