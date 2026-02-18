import type { ComponentProps, HTMLAttributes } from 'react';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

export type StatusProps = ComponentProps<typeof Badge> & {
  status: 'online' | 'offline' | 'maintenance' | 'degraded';
};

// State color configuration - using semantic CSS variables
const statusStyles = {
  online: 'bg-[var(--success)]/10 text-[var(--success)] border-[var(--success)]/20',
  offline: 'bg-[var(--error)]/10 text-[var(--error)] border-[var(--error)]/20',
  maintenance: 'bg-[var(--muted-foreground)]/10 text-[var(--muted-foreground)] border-[var(--muted-foreground)]/20',
  degraded: 'bg-[var(--warning)]/10 text-[var(--warning)] border-[var(--warning)]/20',
};

export const Status = ({ className, status, ...props }: StatusProps) => (
  <Badge
    className={cn('flex items-center gap-2', 'group', status, statusStyles[status], className)}
    variant="outline"
    {...props}
  />
);

export type StatusIndicatorProps = HTMLAttributes<HTMLSpanElement>;

export const StatusIndicator = ({
  className,
  ...props
}: StatusIndicatorProps) => (
  <span className={cn('relative flex h-2 w-2', className)} {...props}>
    <span
      className={cn(
        'absolute inline-flex h-full w-full animate-ping rounded-full opacity-75',
        'group-[.online]:bg-[var(--success)]',
        'group-[.offline]:bg-[var(--error)]',
        'group-[.maintenance]:bg-[var(--muted-foreground)]',
        'group-[.degraded]:bg-[var(--warning)]'
      )}
    />
    <span
      className={cn(
        'relative inline-flex h-2 w-2 rounded-full',
        'group-[.online]:bg-[var(--success)]',
        'group-[.offline]:bg-[var(--error)]',
        'group-[.maintenance]:bg-[var(--muted-foreground)]',
        'group-[.degraded]:bg-[var(--warning)]'
      )}
    />
  </span>
);

export type StatusLabelProps = HTMLAttributes<HTMLSpanElement>;

export const StatusLabel = ({
  className,
  children,
  ...props
}: StatusLabelProps) => (
  <span className={cn(className)} {...props}>
    {children ?? (
      <>
        <span className="hidden group-[.online]:block">Online</span>
        <span className="hidden group-[.offline]:block">Offline</span>
        <span className="hidden group-[.maintenance]:block">Maintenance</span>
        <span className="hidden group-[.degraded]:block">Degraded</span>
      </>
    )}
  </span>
);
