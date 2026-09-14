"use client";
import type { ComponentPropsWithoutRef, ReactElement, ReactNode } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Sheet, SheetClose, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger, } from "@/components/ui/sheet";
import { semanticIcons } from "@/components/icons";
import { EdgePanelHeader } from "@/components/shared/edge-panel-header";
import {
    COMPACT_FORM_OVERLAY_INSET_CLASS,
    COMPACT_FORM_OVERLAY_SECTION_GAP_CLASS,
} from "@/components/shared/layout/page-shell-density";
import { formDrawerContentClassName } from "@/lib/ui/overlay-styles";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
interface FormDrawerFrameProps {
    title: ReactNode;
    description?: ReactNode;
    children: ReactNode;
    icon?: ReactNode;
    footer?: ReactNode;
    closeControl: ReactNode;
    titleMode?: "sheet" | "panel";
    className?: string;
    headerClassName?: string;
    bodyClassName?: string;
    footerClassName?: string;
    formProps?: ComponentPropsWithoutRef<"form">;
}
interface FormDrawerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    title: ReactNode;
    description?: ReactNode;
    children: ReactNode;
    trigger?: ReactElement;
    icon?: ReactNode;
    footer?: ReactNode;
    closeDisabled?: boolean;
    className?: string;
    headerClassName?: string;
    bodyClassName?: string;
    footerClassName?: string;
    formProps?: ComponentPropsWithoutRef<"form">;
}
interface FormDrawerPanelProps {
    title: ReactNode;
    description?: ReactNode;
    children: ReactNode;
    onClose: () => void;
    icon?: ReactNode;
    footer?: ReactNode;
    closeDisabled?: boolean;
    className?: string;
    headerClassName?: string;
    bodyClassName?: string;
    footerClassName?: string;
    formProps?: ComponentPropsWithoutRef<"form">;
}
function FormDrawerFrame({ title, description, children, icon, footer, closeControl, titleMode = "sheet", className, headerClassName, bodyClassName, footerClassName, formProps, }: FormDrawerFrameProps) {
    const { className: formClassName, ...restFormProps } = formProps ?? {};
    const titleNode = titleMode === "sheet" ? (<SheetTitle className={textRole.sectionTitle}>{title}</SheetTitle>) : (<h2 className={textRole.sectionTitle}>{title}</h2>);
    const descriptionNode = description ? (titleMode === "sheet" ? (<SheetDescription className={cn("max-w-2xl", textRole.helperText)}>
              {description}
            </SheetDescription>) : (<p className={cn("max-w-2xl", textRole.helperText)}>
              {description}
            </p>)) : null;
    return (<form className={cn("flex h-full min-h-0 w-full flex-col", className, formClassName)} {...restFormProps}>
      <SheetHeader className={cn("shrink-0 border-b bg-card text-left", COMPACT_FORM_OVERLAY_INSET_CLASS, headerClassName)}>
        <EdgePanelHeader variant="form" title={titleNode} description={descriptionNode} leading={icon} actions={closeControl} />
      </SheetHeader>

      <div className={cn("grid flex-1 content-start overflow-y-auto", COMPACT_FORM_OVERLAY_SECTION_GAP_CLASS, COMPACT_FORM_OVERLAY_INSET_CLASS, bodyClassName)}>
        {children}
      </div>

      {footer ? (<div className={cn("border-t", COMPACT_FORM_OVERLAY_INSET_CLASS, footerClassName)}>
          {footer}
        </div>) : null}
    </form>);
}
export function FormDrawer({ open, onOpenChange, title, description, children, trigger, icon, footer, closeDisabled, className, headerClassName, bodyClassName, footerClassName, formProps, }: FormDrawerProps) {
    const tActions = useTranslations("common.actions");
    return (<Sheet open={open} onOpenChange={onOpenChange}>
      {trigger ? (<SheetTrigger render={trigger}></SheetTrigger>) : null}

      <SheetContent side="right" showCloseButton={false} className={cn(formDrawerContentClassName, className)}>
        <FormDrawerFrame title={title} description={description} icon={icon} footer={footer} headerClassName={headerClassName} bodyClassName={bodyClassName} footerClassName={footerClassName} formProps={formProps} closeControl={(<SheetClose render={<Button className="overlay-close-control" type="button" variant="ghost" size="icon-sm" aria-label={tActions("close")} disabled={closeDisabled}/>}>
                <semanticIcons.action.cancel />
              </SheetClose>)}>
          {children}
        </FormDrawerFrame>
      </SheetContent>
    </Sheet>);
}
export function FormDrawerPanel({ title, description, children, onClose, icon, footer, closeDisabled, className, headerClassName, bodyClassName, footerClassName, formProps, }: FormDrawerPanelProps) {
    const tActions = useTranslations("common.actions");
    return (<FormDrawerFrame title={title} description={description} icon={icon} footer={footer} className={className} headerClassName={headerClassName} bodyClassName={bodyClassName} footerClassName={footerClassName} formProps={formProps} titleMode="panel" closeControl={(<Button className="overlay-close-control" type="button" variant="ghost" size="icon-sm" aria-label={tActions("close")} disabled={closeDisabled} onClick={onClose}>
          <semanticIcons.action.cancel />
        </Button>)}>
      {children}
    </FormDrawerFrame>);
}
