export class ApiError extends Error {
  status: number;
  fieldErrors?: Record<string, string[]>;
  code?: string;
  retryAfter?: number;
  constructor(status: number, message: string, fieldErrors?: Record<string, string[]>) {
    super(message);
    this.status = status;
    this.fieldErrors = fieldErrors;
  }
}

export function isTemporaryNetworkError(error: unknown) {
  return (
    error instanceof TypeError ||
    (error instanceof Error && ["TimeoutError", "NetworkError"].includes(error.name))
  );
}

export function isDefinitiveRejection(error: unknown) {
  return (
    error instanceof ApiError && error.status >= 400 && error.status < 500 && error.status !== 408
  );
}
