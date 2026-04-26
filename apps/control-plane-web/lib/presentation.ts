export function formatControlPlaneError(error: unknown) {
  if (error instanceof Error && error.message) {
    return error.message
      .replace(/[_-]+/g, ' ')
      .replace(/\s+/g, ' ')
      .trim();
  }
  return 'request failed';
}

export function joinList(values: string[]) {
  return values.join(', ');
}

export function splitList(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}
