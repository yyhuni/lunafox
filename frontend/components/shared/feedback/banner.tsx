'use client';

import { semanticIcons, type Icon } from "@/components/icons";
import {
  type ComponentProps,
  createContext,
  type HTMLAttributes,
  type MouseEventHandler,
  useContext,
  useState,
} from 'react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

type BannerContextProps = {
  show: boolean;
  setShow: (show: boolean) => void;
};

export const BannerContext = createContext<BannerContextProps>({
  show: true,
  setShow: () => {},
});

export type BannerProps = HTMLAttributes<HTMLDivElement> & {
  visible?: boolean;
  defaultVisible?: boolean;
  onClose?: () => void;
  inset?: boolean;
};

function useControllableBannerState({
  defaultVisible,
  onClose,
  visible,
}: Pick<BannerProps, "defaultVisible" | "onClose" | "visible">) {
  const [uncontrolledVisible, setUncontrolledVisible] = useState(defaultVisible ?? true);
  const isControlled = visible !== undefined;
  const show = isControlled ? visible : uncontrolledVisible;

  const setShow = (nextVisible: boolean) => {
    if (!isControlled) {
      setUncontrolledVisible(nextVisible);
    }

    if (!nextVisible && show) {
      onClose?.();
    }
  };

  return [show, setShow] as const;
}

export const Banner = ({
  children,
  visible,
  defaultVisible = true,
  onClose,
  className,
  inset = false,
  ...props
}: BannerProps) => {
  const [show, setShow] = useControllableBannerState({ defaultVisible, onClose, visible });

  if (!show) {
    return null;
  }

  return (
    <BannerContext.Provider value={{ show, setShow }}>
      <div
        className={cn(
          'flex w-full items-center justify-between gap-2 bg-primary px-4 py-2 text-primary-foreground',
          inset && 'rounded-lg',
          className
        )}
        {...props}
      >
        {children}
      </div>
    </BannerContext.Provider>
  );
};

export type BannerIconProps = HTMLAttributes<HTMLDivElement> & {
  icon: Icon;
};

export const BannerIcon = ({
  icon: Icon,
  className,
  ...props
}: BannerIconProps) => (
  <div
    className={cn(
      'rounded-full border border-background/20 bg-background/10 p-1 shadow-sm',
      className
    )}
    {...props}
  >
    <Icon size={16} />
  </div>
);

export type BannerTitleProps = HTMLAttributes<HTMLParagraphElement>;

export const BannerTitle = ({ className, ...props }: BannerTitleProps) => (
  <p className={cn('flex-1 text-sm', className)} {...props} />
);

export type BannerActionProps = ComponentProps<typeof Button>;

export const BannerAction = ({
  variant = 'outline',
  size = 'sm',
  className,
  ...props
}: BannerActionProps) => (
  <Button
    className={cn(
      'shrink-0 bg-transparent hover:bg-background/10 hover:text-background',
      className
    )}
    size={size}
    variant={variant}
    {...props}
  />
);

export type BannerCloseProps = ComponentProps<typeof Button>;

export const BannerClose = ({
  variant = 'ghost',
  size = 'icon',
  onClick,
  className,
  ...props
}: BannerCloseProps) => {
  const { setShow } = useContext(BannerContext);

  const handleClick: MouseEventHandler<HTMLButtonElement> = (e) => {
    setShow(false);
    onClick?.(e);
  };

  return (
    <Button
      className={cn(
        'shrink-0 bg-transparent hover:bg-background/10 hover:text-background',
        className
      )}
      onClick={handleClick}
      size={size}
      variant={variant}
      {...props}
    >
      <semanticIcons.action.cancel size={18} />
    </Button>
  );
};
