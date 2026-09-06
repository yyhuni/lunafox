"use client";
import * as React from "react";
import { useLocale, useTranslations } from "next-intl";
import { ChevronDownIcon } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger, } from "@/components/ui/popover";
interface DateTimePickerProps {
    value?: Date;
    onChange?: (date: Date | undefined) => void;
    label?: string;
    placeholder?: string;
    minDate?: Date;
}
export function DateTimePicker({ value, onChange, label, placeholder, minDate, }: DateTimePickerProps) {
    const locale = useLocale();
    const t = useTranslations("common.dateTimePicker");
    const [open, setOpen] = React.useState(false);
    const [date, setDate] = React.useState<Date | undefined>(value);
    const [time, setTime] = React.useState<string>(value ? `${String(value.getHours()).padStart(2, "0")}:${String(value.getMinutes()).padStart(2, "0")}` : "02:00");
    // Keep internal UI state in sync when parent-controlled value changes.
    React.useEffect(() => {
        if (!value) {
            setDate(undefined);
            return;
        }
        setDate(value);
        setTime(`${String(value.getHours()).padStart(2, "0")}:${String(value.getMinutes()).padStart(2, "0")}`);
    }, [value]);
    // Merge date and time
    const updateDateTime = React.useCallback((newDate: Date | undefined, newTime: string) => {
        if (!newDate) {
            onChange?.(undefined);
            return;
        }
        const [hours, minutes] = newTime.split(":").map(Number);
        const dateTime = new Date(newDate);
        dateTime.setHours(hours || 0, minutes || 0, 0, 0);
        onChange?.(dateTime);
    }, [onChange]);
    const handleDateChange = (newDate: Date | undefined) => {
        setDate(newDate);
        updateDateTime(newDate, time);
        setOpen(false);
    };
    const handleTimeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setTime(e.target.value);
        updateDateTime(date, e.target.value);
    };
    const resolvedLabel = label ?? t("executionTime");
    const resolvedPlaceholder = placeholder ?? t("selectDateTime");
    const resolvedDisplayDate = date
        ? date.toLocaleDateString(locale, {
            year: "numeric",
            month: "2-digit",
            day: "2-digit",
        })
        : resolvedPlaceholder;
    return (<div className="flex flex-col gap-3">
      {resolvedLabel && (<Label className="px-1">{resolvedLabel}</Label>)}
      <div className="flex gap-3">
        {/* Date selection */}
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger render={<Button variant="outline" className="flex-1 font-normal justify-between"/>}>
              {resolvedDisplayDate}
              <ChevronDownIcon className="h-4 opacity-50 w-4"/>
            </PopoverTrigger>
          <PopoverContent className="overflow-hidden p-0 w-auto" align="start">
            <Calendar mode="single" selected={date} captionLayout="dropdown" onSelect={handleDateChange} disabled={minDate ? { before: minDate } : undefined}/>
          </PopoverContent>
        </Popover>

        {/* Time selection */}
        <Input type="time" name="executionTime" autoComplete="off" value={time} onChange={handleTimeChange} className="[&::-webkit-calendar-picker-indicator]:appearance-none [&::-webkit-calendar-picker-indicator]:hidden appearance-none bg-background w-28"/>
      </div>
    </div>);
}
