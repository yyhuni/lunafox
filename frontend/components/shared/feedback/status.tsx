'use client';

import type { ComponentProps, HTMLAttributes } from 'react';
import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import {
  getRuntimeStatusClasses,
} from '@/lib/status-config';
import { cn } from '@/lib/utils';

export type StatusProps = ComponentProps<typeof Badge> & {
  status: 'online' | 'offline' | 'maintenance' | 'degraded';
};

export const Status = ({ className, status, ...props }: StatusProps) => (
  <Badge
    className={cn('flex items-center gap-2', 'group', `status-${status}`, getRuntimeStatusClasses(status), className)}
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
        'group-[.status-online]:bg-success',
        'group-[.status-offline]:bg-error',
        'group-[.status-maintenance]:bg-muted-foreground',
        'group-[.status-degraded]:bg-warning'
      )}
    />
    <span
      className={cn(
        'relative inline-flex h-2 w-2 rounded-full',
        'group-[.status-online]:bg-success',
        'group-[.status-offline]:bg-error',
        'group-[.status-maintenance]:bg-muted-foreground',
        'group-[.status-degraded]:bg-warning'
      )}
    />
  </span>
);

export type StatusLabelProps = HTMLAttributes<HTMLSpanElement>;

export const StatusLabel = ({
  className,
  children,
  ...props
}: StatusLabelProps) => {
  const t = useTranslations('common.status');

  return (
    <span className={cn(className)} {...props}>
      {children ?? (
        <>
          <span className="group-[.status-online]:block hidden">{t('online')}</span>
          <span className="group-[.status-offline]:block hidden">{t('offline')}</span>
          <span className="group-[.status-maintenance]:block hidden">{t('maintenance')}</span>
          <span className="group-[.status-degraded]:block hidden">{t('degraded')}</span>
        </>
      )}
    </span>
  );
};
