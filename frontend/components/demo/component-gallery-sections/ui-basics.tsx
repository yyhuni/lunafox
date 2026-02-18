"use client"

import React from "react"
import { DemoCard, DemoSection } from "@/components/demo/component-gallery-sections/shared"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Checkbox } from "@/components/ui/checkbox"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Switch } from "@/components/ui/switch"
import { Toggle } from "@/components/ui/toggle"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Calendar } from "@/components/ui/calendar"
import { DateTimePicker } from "@/components/ui/datetime-picker"
import { Dropzone, DropzoneContent, DropzoneEmptyState } from "@/components/ui/dropzone"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"

export function UiBasicsSection() {
  const [calendarDate, setCalendarDate] = React.useState<Date | undefined>(new Date())
  const [dateTime, setDateTime] = React.useState<Date | undefined>(new Date())
  const [dropFiles, setDropFiles] = React.useState<File[]>([])
  const [checkboxValue, setCheckboxValue] = React.useState(false)
  const [radioValue, setRadioValue] = React.useState("a")
  const [switchValue, setSwitchValue] = React.useState(true)
  const [toggleValue, setToggleValue] = React.useState(false)
  const [toggleGroupValue, setToggleGroupValue] = React.useState<string[]>(["bold"])
  const [selectValue, setSelectValue] = React.useState("alpha")

  return (
    <DemoSection
      id="ui-basics"
      title="UI 基础组件"
      description="按钮、表单、选择器与基础元素。"
    >
      <div className="grid gap-4 px-4 lg:px-6 md:grid-cols-2 xl:grid-cols-3">
        <DemoCard title="Button" description="基础按钮与状态">
          <div className="flex flex-wrap gap-2">
            <Button>Primary</Button>
            <Button variant="outline">Outline</Button>
            <Button variant="secondary">Secondary</Button>
            <Button variant="destructive">Destructive</Button>
            <Button size="sm">Small</Button>
          </div>
        </DemoCard>

        <DemoCard title="Badge" description="状态与标签">
          <div className="flex flex-wrap gap-2">
            <Badge>Default</Badge>
            <Badge variant="secondary">Secondary</Badge>
            <Badge variant="outline">Outline</Badge>
            <Badge className="bg-[var(--success)]/10 text-[var(--success)] border-[var(--success)]/30" variant="outline">
              Healthy
            </Badge>
          </div>
        </DemoCard>

        <DemoCard title="Input + Label" description="基础输入">
          <div className="space-y-2">
            <Label htmlFor="demo-input">资产名称</Label>
            <Input autoComplete="off" name="demoInput" id="demo-input" placeholder="example.com" />
          </div>
        </DemoCard>

        <DemoCard title="Textarea" description="多行输入">
          <Textarea autoComplete="off" name="demoTextarea" placeholder="输入说明…" rows={4} />
        </DemoCard>

        <DemoCard title="Checkbox" description="多选">
          <div className="flex items-center gap-2">
            <Checkbox
              id="demo-checkbox"
              checked={checkboxValue}
              onCheckedChange={(value) => setCheckboxValue(Boolean(value))}
            />
            <Label htmlFor="demo-checkbox">启用深度扫描</Label>
          </div>
        </DemoCard>

        <DemoCard title="Radio Group" description="单选">
          <RadioGroup value={radioValue} onValueChange={setRadioValue} className="gap-2">
            <div className="flex items-center gap-2">
              <RadioGroupItem value="a" id="radio-a" />
              <Label htmlFor="radio-a">Quick</Label>
            </div>
            <div className="flex items-center gap-2">
              <RadioGroupItem value="b" id="radio-b" />
              <Label htmlFor="radio-b">Deep</Label>
            </div>
          </RadioGroup>
        </DemoCard>

        <DemoCard title="Switch" description="开关">
          <div className="flex items-center gap-3">
            <Switch checked={switchValue} onCheckedChange={setSwitchValue} />
            <span className="text-sm text-muted-foreground">
              {switchValue ? "实时监控开启" : "实时监控关闭"}
            </span>
          </div>
        </DemoCard>

        <DemoCard title="Toggle / ToggleGroup" description="强调型切换">
          <div className="flex flex-col gap-3">
            <Toggle pressed={toggleValue} onPressedChange={setToggleValue}>
              单个 Toggle
            </Toggle>
            <ToggleGroup
              type="multiple"
              value={toggleGroupValue}
              onValueChange={setToggleGroupValue}
              className="flex flex-wrap gap-2"
            >
              <ToggleGroupItem value="bold">Bold</ToggleGroupItem>
              <ToggleGroupItem value="italic">Italic</ToggleGroupItem>
              <ToggleGroupItem value="mono">Mono</ToggleGroupItem>
            </ToggleGroup>
          </div>
        </DemoCard>

        <DemoCard title="Select" description="下拉选择">
          <Select value={selectValue} onValueChange={setSelectValue}>
            <SelectTrigger>
              <SelectValue placeholder="选择扫描策略" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="alpha">Alpha</SelectItem>
              <SelectItem value="beta">Beta</SelectItem>
              <SelectItem value="gamma">Gamma</SelectItem>
            </SelectContent>
          </Select>
        </DemoCard>

        <DemoCard title="DateTimePicker" description="日期时间选择">
          <DateTimePicker value={dateTime} onChange={setDateTime} />
        </DemoCard>

        <DemoCard title="Calendar" description="日期日历">
          <Calendar mode="single" selected={calendarDate} onSelect={setCalendarDate} />
        </DemoCard>

        <DemoCard title="Dropzone" description="上传拖拽区">
          <Dropzone
            src={dropFiles}
            maxFiles={3}
            onDrop={(files) => setDropFiles(files)}
            className="border-dashed"
          >
            <DropzoneEmptyState />
            <DropzoneContent />
          </Dropzone>
        </DemoCard>

        <DemoCard title="Avatar" description="用户头像">
          <div className="flex items-center gap-3">
            <Avatar>
              <AvatarImage src="/images/icon-64.png" alt="User" />
              <AvatarFallback>LF</AvatarFallback>
            </Avatar>
            <span className="text-sm text-muted-foreground">Operator</span>
          </div>
        </DemoCard>
      </div>
    </DemoSection>
  )
}
