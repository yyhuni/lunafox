import {
  useMutation,
  useQueryClient,
  type InvalidateQueryFilters,
  type MutationFunction,
  type QueryClient,
  type UseMutationResult,
} from '@tanstack/react-query'
import { useToastMessages, type ToastMessages, type ToastParams } from '@/lib/toast-helpers'
import { getErrorCode, getErrorResponseData } from '@/lib/response-parser'

type InvalidateDescriptor<TData, TVariables> =
  | InvalidateQueryFilters
  | ((context: { data: TData; variables: TVariables }) => InvalidateQueryFilters)

type LoadingToastDescriptor<TVariables> = {
  key: string
  params?: ToastParams | ((variables: TVariables) => ToastParams | undefined)
  id: string | ((variables: TVariables) => string)
}

type MutationRetry = false | number | ((failureCount: number, error: unknown) => boolean)

type MutationContext<TContext> = {
  toastLifecycle?: OperationToastLifecycle
  userContext?: TContext
}

type OperationToastLifecycle = {
  id: string
  terminalShown: boolean
}

type SuccessHandlerContext<TData, TVariables, TContext> = {
  data: TData
  variables: TVariables
  context: TContext | undefined
  queryClient: QueryClient
  toast: ToastMessages
}

type ErrorHandlerContext<TVariables, TContext> = {
  error: unknown
  variables: TVariables
  context: TContext | undefined
  queryClient: QueryClient
  toast: ToastMessages
}

type SettledHandlerContext<TData, TVariables, TContext> = {
  data: TData | undefined
  error: unknown | null
  variables: TVariables
  context: TContext | undefined
  queryClient: QueryClient
  toast: ToastMessages
}

export type UseResourceMutationOptions<TData, TVariables, TContext = unknown> = {
  mutationFn: MutationFunction<TData, TVariables>
  retry?: MutationRetry
  loadingToast?: LoadingToastDescriptor<TVariables>
  invalidate?: Array<InvalidateDescriptor<TData, TVariables>>
  errorFallbackKey?: string
  skipDefaultErrorHandler?: boolean
  onMutate?: (
    variables: TVariables,
    context: { queryClient: QueryClient; toast: ToastMessages }
  ) => Promise<TContext | void> | TContext | void
  onSuccess?: (
    context: SuccessHandlerContext<TData, TVariables, TContext>
  ) => Promise<void> | void
  onError?: (context: ErrorHandlerContext<TVariables, TContext>) => Promise<void> | void
  onSettled?: (
    context: SettledHandlerContext<TData, TVariables, TContext>
  ) => Promise<void> | void
}

function resolveLoadingToastId<TVariables>(
  loadingToast: LoadingToastDescriptor<TVariables> | undefined,
  variables: TVariables
): string | undefined {
  if (!loadingToast) {
    return undefined
  }

  return typeof loadingToast.id === 'function' ? loadingToast.id(variables) : loadingToast.id
}

function resolveLoadingToastParams<TVariables>(
  loadingToast: LoadingToastDescriptor<TVariables> | undefined,
  variables: TVariables
): ToastParams | undefined {
  if (!loadingToast?.params) {
    return undefined
  }

  return typeof loadingToast.params === 'function'
    ? loadingToast.params(variables)
    : loadingToast.params
}

function createOperationToastLifecycle<TVariables>(
  loadingToast: LoadingToastDescriptor<TVariables> | undefined,
  variables: TVariables
): OperationToastLifecycle | undefined {
  if (!loadingToast) {
    return undefined
  }

  const id = resolveLoadingToastId(loadingToast, variables)
  if (!id) {
    throw new Error('loadingToast.id must resolve to a non-empty value')
  }

  return { id, terminalShown: false }
}

function createOperationToastMessages(
  toast: ToastMessages,
  lifecycle: OperationToastLifecycle | undefined
): ToastMessages {
  if (!lifecycle) {
    return toast
  }

  const resolveTerminalId = (toastId?: string) => toastId ?? lifecycle.id
  const markTerminalIfCurrent = (toastId: string) => {
    if (toastId === lifecycle.id) {
      lifecycle.terminalShown = true
    }
  }

  return {
    success: (key, params, toastId) => {
      const resolvedId = resolveTerminalId(toastId)
      markTerminalIfCurrent(resolvedId)
      toast.success(key, params, resolvedId)
    },
    error: (key, params, toastId) => {
      const resolvedId = resolveTerminalId(toastId)
      markTerminalIfCurrent(resolvedId)
      toast.error(key, params, resolvedId)
    },
    errorFromCode: (code, fallbackKey, toastId) => {
      const resolvedId = resolveTerminalId(toastId)
      markTerminalIfCurrent(resolvedId)
      toast.errorFromCode(code, fallbackKey, resolvedId)
    },
    loading: (key, params, toastId) => {
      toast.loading(key, params, toastId ?? lifecycle.id)
    },
    warning: (key, params, toastId) => {
      const resolvedId = resolveTerminalId(toastId)
      markTerminalIfCurrent(resolvedId)
      toast.warning(key, params, resolvedId)
    },
    dismiss: (toastId) => {
      if (toastId === lifecycle.id) {
        lifecycle.terminalShown = true
      }
      toast.dismiss(toastId)
    },
  }
}

function dismissUnsettledLoadingToast(
  toast: ToastMessages,
  lifecycle: OperationToastLifecycle | undefined
) {
  if (lifecycle && !lifecycle.terminalShown) {
    toast.dismiss(lifecycle.id)
    lifecycle.terminalShown = true
  }
}

export function useResourceMutation<TData, TVariables, TContext = unknown>(
  options: UseResourceMutationOptions<TData, TVariables, TContext>
): UseMutationResult<TData, unknown, TVariables, MutationContext<TContext>> {
  const queryClient = useQueryClient()
  const toast = useToastMessages()

  return useMutation<TData, unknown, TVariables, MutationContext<TContext>>({
    mutationFn: options.mutationFn,
    retry: options.retry,
    onMutate: async (variables) => {
      const toastLifecycle = createOperationToastLifecycle(options.loadingToast, variables)

      if (options.loadingToast && toastLifecycle) {
        const loadingToastParams = resolveLoadingToastParams(options.loadingToast, variables)
        toast.loading(options.loadingToast.key, loadingToastParams, toastLifecycle.id)
      }

      const userContext = await options.onMutate?.(variables, {
        queryClient,
        toast: createOperationToastMessages(toast, toastLifecycle),
      })

      return {
        toastLifecycle,
        userContext: userContext as TContext | undefined,
      }
    },
    onSuccess: async (data, variables, mutationContext) => {
      if (options.invalidate) {
        const invalidateFilters = options.invalidate.map((descriptor) =>
          typeof descriptor === 'function'
            ? descriptor({ data, variables })
            : descriptor
        )
        await Promise.all(
          invalidateFilters.map((invalidateFilter) =>
            queryClient.invalidateQueries(invalidateFilter)
          )
        )
      }

      await options.onSuccess?.({
        data,
        variables,
        context: mutationContext?.userContext,
        queryClient,
        toast: createOperationToastMessages(toast, mutationContext?.toastLifecycle),
      })
    },
    onError: async (error, variables, mutationContext) => {
      const toastLifecycle = mutationContext?.toastLifecycle ??
        createOperationToastLifecycle(options.loadingToast, variables)
      const operationToast = createOperationToastMessages(toast, toastLifecycle)

      try {
        if (options.onError) {
          await options.onError({
            error,
            variables,
            context: mutationContext?.userContext,
            queryClient,
            toast: operationToast,
          })
          return
        }

        if (options.skipDefaultErrorHandler) {
          return
        }

        operationToast.errorFromCode(getErrorCode(getErrorResponseData(error)), options.errorFallbackKey)
      } finally {
        dismissUnsettledLoadingToast(toast, toastLifecycle)
      }
    },
    onSettled: async (data, error, variables, mutationContext) => {
      try {
        await options.onSettled?.({
          data,
          error,
          variables,
          context: mutationContext?.userContext,
          queryClient,
          toast: createOperationToastMessages(toast, mutationContext?.toastLifecycle),
        })
      } finally {
        dismissUnsettledLoadingToast(toast, mutationContext?.toastLifecycle)
      }
    },
  })
}
