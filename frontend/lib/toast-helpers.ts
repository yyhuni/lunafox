/**
 * Toast message helper function
 * 
 * Provide i18n-aware toast message display function:
 * - success(): display success message
 * - error(): display error message
 * - errorFromCode(): Display error message based on error code
 * - loading(): display loading message
 * - dismiss(): close the specified toast
 * 
 * How to use:
 * ```tsx
 * function MyComponent() {
 *   const toastMessages = useToastMessages();
 *   
 *   const handleDelete = async () => {
 *     try {
 *       await deleteItem(id);
 *       toastMessages.success('toast.item.delete.success', { name: item.name });
 *     } catch (error) {
 *       toastMessages.errorFromCode(getErrorCode(error));
 *     }
 *   };
 * }
 * ```
 */

import { toast } from 'sonner';
import { useTranslations } from 'next-intl';
import { getErrorI18nKey, DEFAULT_ERROR_KEY } from './error-code-map';

/**
 * Toast message parameter type
 * Supports interpolation variables of string and numeric types
 */
export type ToastParams = Record<string, string | number>;

/**
 * Toast message Hook return type
 */
export interface ToastMessages {
  /**
   * Show success message
   * @param key - i18n message key
   * @param params - interpolation parameters
   * @param toastId - optional toast ID (for replacement or closing)
   */
  success: (key: string, params?: ToastParams, toastId?: string) => void;
  
  /**
   * Show error message
   * @param key - i18n message key
   * @param params - interpolation parameters
   * @param toastId - optional toast ID (for replacement or closing)
   */
  error: (key: string, params?: ToastParams, toastId?: string) => void;
  
  /**
   * Display error message based on error code
   * @param code - the error code returned by the backend
   * @param fallbackKey - fallback key for unknown error codes (default 'errors.unknown')
   * @param toastId - optional toast ID (for replacement or closing)
   */
  errorFromCode: (code: string | null, fallbackKey?: string, toastId?: string) => void;
  
  /**
   * Show loading message
   * @param key - i18n message key
   * @param params - interpolation parameters
   * @param toastId - toast ID (for subsequent closing)
   */
  loading: (key: string, params?: ToastParams, toastId?: string) => void;
  
  /**
   * Show warning message
   * @param key - i18n message key
   * @param params - interpolation parameters
   * @param toastId - optional toast ID (for replacement or closing)
   */
  warning: (key: string, params?: ToastParams, toastId?: string) => void;
  
  /**
   * Close the specified toast
   * @param toastId - toast ID
   */
  dismiss: (toastId: string) => void;
}

/**
 * i18n aware Toast message Hook
 * 
 * Provides a unified toast message display interface and automatically handles i18n translation and parameter interpolation.
 * 
 * @returns ToastMessages object, including success, error, errorFromCode, loading, warning, and dismiss methods
 * 
 * @example
 * ```tsx
 * function DeleteButton({ item }) {
 *   const toastMessages = useToastMessages();
 *   const { mutate: deleteItem } = useDeleteItem();
 *   
 *   const handleDelete = () => {
 *     toastMessages.loading('toast.item.delete.loading', {}, 'delete-item');
 *     
 *     deleteItem(item.id, {
 *       onSuccess: () => {
 *         toastMessages.dismiss('delete-item');
 *         toastMessages.success('toast.item.delete.success', { name: item.name });
 *       },
 *       onError: (error) => {
 *         toastMessages.dismiss('delete-item');
 *         toastMessages.errorFromCode(getErrorCode(error.response?.data));
 *       }
 *     });
 *   };
 *   
 *   return <button onClick={handleDelete}>Delete</button>;
 * }
 * ```
 */
export function useToastMessages(): ToastMessages {
  // Uses the root namespace, allowing access to all translation keys
  const t = useTranslations();
  
  return {
    success: (key: string, params?: ToastParams, toastId?: string) => {
      const message = t(key, params);
      if (toastId) {
        toast.success(message, { id: toastId });
      } else {
        toast.success(message);
      }
    },
    
    error: (key: string, params?: ToastParams, toastId?: string) => {
      const message = t(key, params);
      if (toastId) {
        toast.error(message, { id: toastId });
      } else {
        toast.error(message);
      }
    },
    
    errorFromCode: (code: string | null, fallbackKey = DEFAULT_ERROR_KEY, toastId?: string) => {
      const errorKey = code ? getErrorI18nKey(code) : fallbackKey;
      const message = t(errorKey);
      if (toastId) {
        toast.error(message, { id: toastId });
      } else {
        toast.error(message);
      }
    },
    
    loading: (key: string, params?: ToastParams, toastId?: string) => {
      const message = t(key, params);
      if (toastId) {
        toast.loading(message, { id: toastId });
      } else {
        toast.loading(message);
      }
    },
    
    warning: (key: string, params?: ToastParams, toastId?: string) => {
      const message = t(key, params);
      if (toastId) {
        toast.warning(message, { id: toastId });
      } else {
        toast.warning(message);
      }
    },
    
    dismiss: (toastId: string) => {
      toast.dismiss(toastId);
    },
  };
}

/**
 * Non-Hook version of toast helper function
 * 
 * Used for scenarios not in React components (such as API interceptors).
 * NOTE: These functions do not support i18n and can only display raw strings.
 * 
 * @example
 * ```ts
 * // used in API interceptor
 * apiClient.interceptors.response.use(
 *   (response) => response,
 *   (error) => {
 *     if (error.response?.status === 401) {
 *       showToast.error('Session expired, please login again');
 *     }
 *     return Promise.reject(error);
 *   }
 * );
 * ```
 */
export const showToast = {
  success: (message: string, toastId?: string) => {
    if (toastId) {
      toast.success(message, { id: toastId });
    } else {
      toast.success(message);
    }
  },
  
  error: (message: string, toastId?: string) => {
    if (toastId) {
      toast.error(message, { id: toastId });
    } else {
      toast.error(message);
    }
  },
  
  loading: (message: string, toastId?: string) => {
    if (toastId) {
      toast.loading(message, { id: toastId });
    } else {
      toast.loading(message);
    }
  },
  
  warning: (message: string, toastId?: string) => {
    if (toastId) {
      toast.warning(message, { id: toastId });
    } else {
      toast.warning(message);
    }
  },
  
  dismiss: (toastId: string) => {
    toast.dismiss(toastId);
  },
};
