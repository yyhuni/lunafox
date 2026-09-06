"use client"

import * as React from "react"
import { IconSearch } from "@/components/icons"
import { useTranslations } from "next-intl"

import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog"

type CommandContextValue = {
  activeValue: string | null
  itemCount: number
  listId: string
  search: string
  shouldFilter: boolean
  setActiveValue: (value: string | null) => void
  setSearch: (value: string) => void
  updateItemVisibility: (id: string, visible: boolean) => void
}

const CommandContext = React.createContext<CommandContextValue | null>(null)

function useCommandContext(component: string) {
  const context = React.useContext(CommandContext)
  if (!context) {
    throw new Error(`${component} must be used within Command`)
  }
  return context
}

function getItemValue(value: string | undefined, children: React.ReactNode) {
  if (value) return value
  if (typeof children === "string" || typeof children === "number") return String(children)
  return ""
}

function matchesCommandSearch(value: string, keywords: string[] | undefined, search: string) {
  if (!search) return true
  const haystack = [value, ...(keywords ?? [])].join(" ").toLowerCase()
  return haystack.includes(search.toLowerCase())
}

type CommandProps = React.HTMLAttributes<HTMLDivElement> & {
  shouldFilter?: boolean
}

const Command = React.forwardRef<HTMLDivElement, CommandProps>(
  ({ className, shouldFilter = true, onKeyDown, ...props }, ref) => {
    const rootRef = React.useRef<HTMLDivElement | null>(null)
    const [activeValue, setActiveValue] = React.useState<string | null>(null)
    const [search, setSearch] = React.useState("")
    const [visibleItems, setVisibleItems] = React.useState(() => new Map<string, boolean>())
    const listId = React.useId()

    const setRootRef = React.useCallback(
      (node: HTMLDivElement | null) => {
        rootRef.current = node
        if (typeof ref === "function") {
          ref(node)
        } else if (ref) {
          ref.current = node
        }
      },
      [ref]
    )

    const updateItemVisibility = React.useCallback((id: string, visible: boolean) => {
      setVisibleItems((current) => {
        if (current.get(id) === visible) return current
        const next = new Map(current)
        next.set(id, visible)
        return next
      })
    }, [])

    const getSelectableItems = React.useCallback(() => {
      return Array.from(
        rootRef.current?.querySelectorAll<HTMLElement>(
          "[data-command-item]:not([data-disabled=true]):not([hidden])"
        ) ?? []
      )
    }, [])

    const moveActive = React.useCallback(
      (direction: 1 | -1) => {
        const items = getSelectableItems()
        if (items.length === 0) return
        const currentIndex = items.findIndex((item) => item.dataset.value === activeValue)
        const nextIndex = currentIndex === -1 ? (direction === 1 ? 0 : items.length - 1) : Math.min(Math.max(currentIndex + direction, 0), items.length - 1)
        const next = items[nextIndex]
        setActiveValue(next.dataset.value ?? null)
        next.scrollIntoView({ block: "nearest" })
      },
      [activeValue, getSelectableItems]
    )

    const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
      onKeyDown?.(event)
      if (event.defaultPrevented) return

      if (event.key === "ArrowDown") {
        event.preventDefault()
        moveActive(1)
      } else if (event.key === "ArrowUp") {
        event.preventDefault()
        moveActive(-1)
      } else if (event.key === "Home") {
        const first = getSelectableItems()[0]
        if (first) {
          event.preventDefault()
          setActiveValue(first.dataset.value ?? null)
          first.scrollIntoView({ block: "nearest" })
        }
      } else if (event.key === "End") {
        const items = getSelectableItems()
        const last = items.at(-1)
        if (last) {
          event.preventDefault()
          setActiveValue(last.dataset.value ?? null)
          last.scrollIntoView({ block: "nearest" })
        }
      } else if (event.key === "Enter" && activeValue) {
        const activeItem = getSelectableItems().find((item) => item.dataset.value === activeValue)
        if (activeItem) {
          event.preventDefault()
          activeItem.click()
        }
      }
    }

    const itemCount = React.useMemo(
      () => Array.from(visibleItems.values()).filter(Boolean).length,
      [visibleItems]
    )

    const contextValue = React.useMemo<CommandContextValue>(
      () => ({
        activeValue,
        itemCount,
        listId,
        search,
        shouldFilter,
        setActiveValue,
        setSearch,
        updateItemVisibility,
      }),
      [activeValue, itemCount, listId, search, shouldFilter, updateItemVisibility]
    )

    return (
      <CommandContext.Provider value={contextValue}>
        <div
          ref={setRootRef}
          className={cn(
            "flex h-full w-full flex-col overflow-hidden rounded-md bg-popover text-popover-foreground",
            className
          )}
          role="combobox"
          aria-controls={listId}
          aria-expanded="true"
          aria-haspopup="listbox"
          data-command-root=""
          onKeyDown={handleKeyDown}
          {...props}
        />
      </CommandContext.Provider>
    )
  }
)
Command.displayName = "Command"

type CommandDialogProps = Omit<React.ComponentProps<typeof Dialog>, "children"> & {
  children?: React.ReactNode
  contentClassName?: string
}

const CommandDialog = ({ children, contentClassName, ...props }: CommandDialogProps) => {
  const t = useTranslations("common.ui")

  return (
    <Dialog {...props}>
      <DialogContent className={cn("overflow-hidden p-0 sm:max-w-[500px]", contentClassName)}>
        <DialogTitle className="sr-only">{t("commandDialog")}</DialogTitle>
        <Command className={cn("[&_[data-command-group-heading]]:px-2 [&_[data-command-group-heading]]:text-muted-foreground [&_[data-command-group]:not([hidden])_~[data-command-group]]:pt-0 [&_[data-command-group]]:px-2 [&_[data-command-input-wrapper]_svg]:h-5 [&_[data-command-input-wrapper]_svg]:w-5 [&_[data-command-input]]:h-12 [&_[data-command-item]_svg]:h-5 [&_[data-command-item]_svg]:w-5 [&_[data-command-item]]:px-2 [&_[data-command-item]]:py-3", `[&_[data-command-group-heading]]:${textRole.metadataLabel}`)}>
          {children}
        </Command>
      </DialogContent>
    </Dialog>
  )
}

const CommandInput = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  ({ className, onChange, value, defaultValue, ...props }, ref) => {
    const command = useCommandContext("CommandInput")

    React.useEffect(() => {
      if (value !== undefined) command.setSearch(String(value))
    }, [command, value])

    const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
      command.setSearch(event.currentTarget.value)
      onChange?.(event)
    }

    return (
      <div className="flex h-10 items-center gap-2 border-b px-3" data-command-input-wrapper="">
        <span className="flex size-4 shrink-0 items-center justify-center">
          <IconSearch className="size-4 opacity-50" />
        </span>
        <input
          ref={ref}
          className={cn(
            "flex h-10 w-full min-w-0 bg-transparent outline-none placeholder:text-muted-foreground focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50",
            textRole.body,
            className
          )}
          value={value}
          defaultValue={defaultValue}
          data-command-input=""
          onChange={handleChange}
          {...props}
        />
      </div>
    )
  }
)
CommandInput.displayName = "CommandInput"

const CommandList = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, id, ...props }, ref) => {
    const command = useCommandContext("CommandList")

    return (
      <div
        ref={ref}
        id={id ?? command.listId}
        className={cn("max-h-[300px] overflow-x-hidden overflow-y-auto", className)}
        role="listbox"
        data-command-list=""
        {...props}
      />
    )
  }
)
CommandList.displayName = "CommandList"

const CommandEmpty = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => {
    const command = useCommandContext("CommandEmpty")

    return (
      <div
        ref={ref}
        className={cn("py-6 text-center", textRole.bodySubtle, className)}
        hidden={command.itemCount > 0}
        data-command-empty=""
        {...props}
      />
    )
  }
)
CommandEmpty.displayName = "CommandEmpty"

type CommandGroupProps = React.HTMLAttributes<HTMLDivElement> & {
  heading?: React.ReactNode
}

const CommandGroup = React.forwardRef<HTMLDivElement, CommandGroupProps>(
  ({ className, heading, children, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        "overflow-hidden p-1 text-foreground [&_[data-command-group-heading]]:px-2 [&_[data-command-group-heading]]:py-1.5 [&_[data-command-group-heading]]:text-muted-foreground",
        `[&_[data-command-group-heading]]:${textRole.metadataLabel}`,
        className
      )}
      role="group"
      data-command-group=""
      {...props}
    >
      {heading ? <div data-command-group-heading="">{heading}</div> : null}
      {children}
    </div>
  )
)
CommandGroup.displayName = "CommandGroup"

const CommandSeparator = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("-mx-1 h-px bg-border", className)} role="separator" data-command-separator="" {...props} />
  )
)
CommandSeparator.displayName = "CommandSeparator"

type CommandItemProps = Omit<React.HTMLAttributes<HTMLDivElement>, "onSelect"> & {
  value?: string
  keywords?: string[]
  disabled?: boolean
  onSelect?: (value: string) => void
}

const CommandItem = React.forwardRef<HTMLDivElement, CommandItemProps>(
  ({ className, value, keywords, disabled = false, onClick, onSelect, children, hidden, ...props }, ref) => {
    const id = React.useId()
    const command = useCommandContext("CommandItem")
    const { updateItemVisibility } = command
    const itemValue = getItemValue(value, children)
    const visible = !hidden && (!command.shouldFilter || matchesCommandSearch(itemValue, keywords, command.search))
    const isActive = command.activeValue === itemValue

    React.useLayoutEffect(() => {
      updateItemVisibility(id, visible)
      return () => updateItemVisibility(id, false)
    }, [id, updateItemVisibility, visible])

    const handleClick = (event: React.MouseEvent<HTMLDivElement>) => {
      if (disabled) {
        event.preventDefault()
        return
      }
      onClick?.(event)
      if (!event.defaultPrevented) onSelect?.(itemValue)
    }

    return (
      <div
        ref={ref}
        className={cn(
          "radius-control relative flex cursor-default select-none items-center px-2 py-1.5 outline-none focus-visible:outline-none data-[active=true]:bg-accent data-[active=true]:text-accent-foreground data-[disabled=true]:pointer-events-none data-[disabled=true]:opacity-50 aria-selected:bg-accent aria-selected:text-accent-foreground",
          textRole.body,
          className
        )}
        role="option"
        hidden={!visible}
        aria-disabled={disabled || undefined}
        aria-selected={isActive}
        data-active={isActive ? "true" : undefined}
        data-command-item=""
        data-disabled={disabled ? "true" : undefined}
        data-value={itemValue}
        onClick={handleClick}
        onPointerMove={() => {
          if (!disabled) command.setActiveValue(itemValue)
        }}
        {...props}
      >
        {children}
      </div>
    )
  }
)
CommandItem.displayName = "CommandItem"

const CommandShortcut = ({ className, ...props }: React.HTMLAttributes<HTMLSpanElement>) => {
  return <span className={cn("ml-auto", textRole.metadataLabel, className)} {...props} />
}
CommandShortcut.displayName = "CommandShortcut"

export {
  Command,
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandSeparator,
  CommandShortcut,
}
