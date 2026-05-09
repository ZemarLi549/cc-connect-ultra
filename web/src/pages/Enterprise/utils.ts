export function parseCSV(input: string) {
  return input
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

export function joinCSV(values?: string[]) {
  return values?.join(', ') || '';
}

export function formatTime(value?: string) {
  if (!value) return '-';
  return new Date(value).toLocaleString();
}

export function safeJSONStringify(value: unknown, fallback: string) {
  if (value === undefined || value === null) {
    return fallback;
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return fallback;
  }
}

export function parseJSONArray<T = any>(input: string, label: string): T[] {
  const trimmed = input.trim();
  if (!trimmed) return [];
  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    throw new Error(`${label} 必须是合法 JSON 数组`);
  }
  if (!Array.isArray(parsed)) {
    throw new Error(`${label} 必须是 JSON 数组`);
  }
  return parsed as T[];
}

export function parseJSONObject<T extends Record<string, any>>(input: string, label: string): T {
  const trimmed = input.trim();
  if (!trimmed) return {} as T;
  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    throw new Error(`${label} 必须是合法 JSON 对象`);
  }
  if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
    throw new Error(`${label} 必须是 JSON 对象`);
  }
  return parsed as T;
}
