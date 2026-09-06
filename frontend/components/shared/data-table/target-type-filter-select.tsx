"use client"

import { Filter } from "@/components/icons"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { DataTableFacetedFilter, DataTableFacetedFilterGroup } from "./faceted-filter"
import type { TargetType } from "@/types/target.types"

interface TargetTypeFilterSelectOptionLabels {
  all: string
  domain: string
  ip: string
  cidr: string
}

interface TargetTypeFilterSelectProps {
  value: TargetType | "all"
  onValueChange: (value: TargetType | "all") => void
  labels: TargetTypeFilterSelectOptionLabels
}

type TargetTypeOption = {
  value: TargetType | "all"
  label: string
}

type FacetedTargetTypeOption = {
  value: TargetType
  label: string
}

interface TargetTypeFacetedFilterProps {
  title: string
  values: TargetType[]
  onValuesChange: (values: TargetType[]) => void
  labels: TargetTypeFilterSelectOptionLabels
  clearLabel: string
}

export function TargetTypeFilterSelect({
  value,
  onValueChange,
  labels,
}: TargetTypeFilterSelectProps) {
  const options: TargetTypeOption[] = [
    {
      value: "all",
      label: labels.all,
    },
    {
      value: "domain",
      label: labels.domain,
    },
    {
      value: "ip",
      label: labels.ip,
    },
    {
      value: "cidr",
      label: labels.cidr,
    },
  ]

  return (
    <Select value={value} onValueChange={(nextValue) => onValueChange(nextValue as TargetType | "all")}>
      <SelectTrigger size="sm" className="w-28">
        <Filter className="h-4 w-4" />
        <SelectValue placeholder={labels.all} />
      </SelectTrigger>
      <SelectContent width="content-fit">
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export function TargetTypeFacetedFilter({
  title,
  values,
  onValuesChange,
  labels,
  clearLabel,
}: TargetTypeFacetedFilterProps) {
  const options: FacetedTargetTypeOption[] = [
    {
      value: "domain",
      label: labels.domain,
    },
    {
      value: "ip",
      label: labels.ip,
    },
    {
      value: "cidr",
      label: labels.cidr,
    },
  ]

  return (
    <DataTableFacetedFilterGroup hasSelectedValues={values.length > 0} onReset={() => onValuesChange([])}>
      <DataTableFacetedFilter
        title={title}
        values={values}
        onValuesChange={onValuesChange}
        options={options}
        emptyLabel={labels.all}
        clearLabel={clearLabel}
      />
    </DataTableFacetedFilterGroup>
  )
}
