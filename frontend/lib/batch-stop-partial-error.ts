export const batchStopRequestLimit = 100;

export type BatchStopPartialCompletion = {
  stoppedCount: number;
  skippedCount: number;
  revokedTaskCount: number;
  remainingCount: number;
  completedIds: readonly number[];
  cause: unknown;
};

export class BatchStopPartialCompletionError extends Error {
  readonly stoppedCount: number;
  readonly skippedCount: number;
  readonly revokedTaskCount: number;
  readonly remainingCount: number;
  readonly completedIds: readonly number[];

  constructor(details: BatchStopPartialCompletion) {
    super("batchStop committed earlier requests before a later request failed", {
      cause: details.cause,
    });
    this.name = "BatchStopPartialCompletionError";
    this.stoppedCount = details.stoppedCount;
    this.skippedCount = details.skippedCount;
    this.revokedTaskCount = details.revokedTaskCount;
    this.remainingCount = details.remainingCount;
    this.completedIds = details.completedIds;
  }
}

export function isBatchStopPartialCompletionError(
  error: unknown,
): error is BatchStopPartialCompletionError {
  return error instanceof BatchStopPartialCompletionError;
}
