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
  id?: string | ((variables: TVariables) => string)
}

type MutationContext<TContext> = {
  loadingToastId?: string
  userContext?: TContext
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
  if (!loadingToast?.id) {
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

export function useResourceMutation<TData, TVariables, TContext = unknown>(
  options: UseResourceMutationOptions<TData, TVariables, TContext>
): UseMutationResult<TData, unknown, TVariables, MutationContext<TContext>> {
  const queryClient = useQueryClient()
  const toast = useToastMessages()

  return useMutation<TData, unknown, TVariables, MutationContext<TContext>>({
    mutationFn: options.mutationFn,
    onMutate: async (variables) => {
      let loadingToastId: string | undefined

      if (options.loadingToast) {
        loadingToastId = resolveLoadingToastId(options.loadingToast, variables)
        const loadingToastParams = resolveLoadingToastParams(options.loadingToast, variables)
        toast.loading(options.loadingToast.key, loadingToastParams, loadingToastId)
      }

      const userContext = await options.onMutate?.(variables, {
        queryClient,
        toast,
      })

      return {
        loadingToastId,
        userContext: userContext as TContext | undefined,
      }
    },
    onSuccess: async (data, variables, mutationContext) => {
      if (mutationContext?.loadingToastId) {
        toast.dismiss(mutationContext.loadingToastId)
      }

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
        toast,
      })
    },
    onError: async (error, variables, mutationContext) => {
      if (mutationContext?.loadingToastId) {
        toast.dismiss(mutationContext.loadingToastId)
      }

      if (options.onError) {
        await options.onError({
          error,
          variables,
          context: mutationContext?.userContext,
          queryClient,
          toast,
        })
        return
      }

      if (options.skipDefaultErrorHandler) {
        return
      }

      toast.errorFromCode(getErrorCode(getErrorResponseData(error)), options.errorFallbackKey)
    },
    onSettled: async (data, error, variables, mutationContext) => {
      await options.onSettled?.({
        data,
        error,
        variables,
        context: mutationContext?.userContext,
        queryClient,
        toast,
      })
    },
  })
}
