import { useState, useEffect, useCallback } from "react"
import { useRouter, useSearchParams, usePathname } from "next/navigation"

export function useUrlState<T extends string>(key: string, defaultValue: T): [T, (value: T) => void] {
  const router = useRouter()
  const pathname = usePathname()
  const searchParams = useSearchParams()
  
  // Initialize from URL or default
  const [state, setState] = useState<T>(() => {
    const param = searchParams.get(key)
    return (param as T) || defaultValue
  })

  useEffect(() => {
    const param = searchParams.get(key)
    if (param !== state && param !== null) {
      setState(param as T)
    }
  }, [searchParams, key, state])

  const setValue = useCallback((value: T) => {
    setState(value)
    
    // Update URL
    const params = new URLSearchParams(searchParams)
    if (value === defaultValue) {
      params.delete(key)
    } else {
      params.set(key, value)
    }
    
    router.replace(`${pathname}?${params.toString()}`, { scroll: false })
  }, [defaultValue, key, pathname, router, searchParams])

  return [state, setValue]
}
