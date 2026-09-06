import React from "react";
import { AlertTriangle, IconPlus, IconX } from "@/components/icons";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage, } from "@/components/ui/form";
import type { UseFormReturn } from "react-hook-form";
import type { AddToolFormValues } from "@/components/tools/config/add-tool-dialog-state";
import { CategoryNameMap } from "@/types/tool.types";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string;
interface ToolCategorySelectorProps {
    t: TranslationFn;
    availableCategories: string[];
    selectedCategories: string[];
    disabled: boolean;
    onToggle: (name: string) => void;
    onRemove: (name: string) => void;
}
function ToolCategorySelector({ t, availableCategories, selectedCategories, disabled, onToggle, onRemove, }: ToolCategorySelectorProps) {
    return (<div className="gap-2 grid">
      <FormLabel>{t("categoryTags")}</FormLabel>

      {selectedCategories.length > 0 && (<div className="bg-muted/50 border flex flex-wrap gap-2 p-3 rounded-md">
          {selectedCategories.map((categoryName) => (<Badge key={categoryName} variant="default" className="flex gap-1 items-center px-2 py-1">
              {CategoryNameMap[categoryName] || categoryName}
              <button type="button" onClick={() => onRemove(categoryName)} disabled={disabled} className="hover:bg-primary/20 ml-1 p-0.5 rounded-full">
                <IconX className="h-3 w-3"/>
              </button>
            </Badge>))}
        </div>)}

      <div className="border flex flex-wrap gap-2 p-3 rounded-md">
        {availableCategories.length > 0 ? (availableCategories.map((categoryName) => {
            const isSelected = selectedCategories.includes(categoryName);
            return (<Badge key={categoryName} variant={isSelected ? "secondary" : "outline"} className="hover:bg-secondary/80 transition-colors" render={<button type="button" onClick={() => onToggle(categoryName)}/>}>
                  {CategoryNameMap[categoryName] || categoryName}
                </Badge>);
        })) : (<p className="text-muted-foreground text-sm">{t("noCategories")}</p>)}
      </div>
    </div>);
}
interface AddToolBasicInfoSectionProps {
    t: TranslationFn;
    form: UseFormReturn<AddToolFormValues>;
    isPending: boolean;
    availableCategories: string[];
    selectedCategories: string[];
    onToggleCategory: (name: string) => void;
    onRemoveCategory: (name: string) => void;
}
export function AddToolBasicInfoSection({ t, form, isPending, availableCategories, selectedCategories, onToggleCategory, onRemoveCategory, }: AddToolBasicInfoSectionProps) {
    return (<div className="space-y-4">
      <h3 className={cn(textRole.sectionTitle, "text-muted-foreground")}>{t("basicInfo")}</h3>

      <FormField control={form.control} name="name" render={({ field }) => (<FormItem>
            <FormLabel>{t("toolName")} <span className="text-destructive">*</span></FormLabel>
            <FormControl>
              <Input placeholder={t("toolNamePlaceholder")} disabled={isPending} maxLength={255} autoComplete="off" {...field}/>
            </FormControl>
            <FormDescription>{t("characters", { count: field.value.length, max: 255 })}</FormDescription>
            <FormMessage />
          </FormItem>)}/>

      <FormField control={form.control} name="repoUrl" render={({ field }) => (<FormItem>
            <FormLabel>{t("repoUrl")}</FormLabel>
            <FormControl>
              <Input type="url" placeholder={t("repoUrlPlaceholder")} disabled={isPending} maxLength={512} autoComplete="url" inputMode="url" {...field}/>
            </FormControl>
            <FormMessage />
          </FormItem>)}/>

      <FormField control={form.control} name="version" render={({ field }) => (<FormItem>
            <FormLabel>{t("currentVersion")}</FormLabel>
            <FormControl>
              <Input placeholder={t("versionPlaceholder")} disabled={isPending} maxLength={100} autoComplete="off" {...field}/>
            </FormControl>
            <FormMessage />
          </FormItem>)}/>

      <FormField control={form.control} name="description" render={({ field }) => (<FormItem>
            <FormLabel>{t("toolDesc")}</FormLabel>
            <FormControl>
              <Textarea placeholder={t("toolDescPlaceholder")} disabled={isPending} rows={3} maxLength={1000} autoComplete="off" {...field}/>
            </FormControl>
            <FormDescription>{t("characters", { count: (field.value || "").length, max: 1000 })}</FormDescription>
            <FormMessage />
          </FormItem>)}/>

      <ToolCategorySelector t={t} availableCategories={availableCategories} selectedCategories={selectedCategories} disabled={isPending} onToggle={onToggleCategory} onRemove={onRemoveCategory}/>
    </div>);
}
interface AddToolCommandSectionProps {
    t: TranslationFn;
    form: UseFormReturn<AddToolFormValues>;
    isPending: boolean;
}
export function AddToolCommandSection({ t, form, isPending, }: AddToolCommandSectionProps) {
    return (<div className="space-y-4">
      <h3 className={cn(textRole.sectionTitle, "text-muted-foreground")}>{t("commandConfig")}</h3>

      <FormField control={form.control} name="installCommand" render={({ field }) => (<FormItem>
            <FormLabel>{t("installCommand")} <span className="text-destructive">*</span></FormLabel>
            <FormControl>
              <Textarea placeholder={t("installCommandPlaceholder")} disabled={isPending} rows={3} className="font-mono text-sm" autoComplete="off" {...field}/>
            </FormControl>
            <FormDescription className="space-y-1">
              <span className="block"><strong>{t("installCommandHint")}</strong></span>
              <span className="block">• {t("installCommandGit")} <code className="bg-muted px-1 py-0.5 rounded">git clone https://github.com/user/tool</code></span>
              <span className="block">• {t("installCommandGo")} <code className="bg-muted px-1 py-0.5 rounded">go install -v github.com/tool@v1.2.3</code></span>
              <span className="flex gap-1 items-center text-warning">
                <AlertTriangle className="h-3.5 w-3.5"/>
                {t("installCommandNote")}
              </span>
            </FormDescription>
            <FormMessage />
          </FormItem>)}/>

      <FormField control={form.control} name="updateCommand" render={({ field }) => (<FormItem>
            <FormLabel>{t("updateCommand")} <span className="text-destructive">*</span></FormLabel>
            <FormControl>
              <Textarea placeholder={t("updateCommandPlaceholder")} disabled={isPending} rows={2} className="font-mono text-sm" autoComplete="off" {...field}/>
            </FormControl>
            <FormDescription className="space-y-1">
              <span className="block">• {t("updateCommandGitHint")} <code className="bg-muted px-1 py-0.5 rounded">git pull</code></span>
              <span className="block">• {t("updateCommandGoHint")}</span>
            </FormDescription>
            <FormMessage />
          </FormItem>)}/>

      <FormField control={form.control} name="versionCommand" render={({ field }) => (<FormItem>
            <FormLabel>
              {t("versionCommand")} <span className="text-destructive">*</span>
              {field.value && (<span className={cn("ml-2", textRole.helperText)}>
                  {t("versionCommandAutoGenerated")}
                </span>)}
            </FormLabel>
            <FormControl>
              <Input placeholder={t("versionCommandPlaceholder")} disabled={isPending} maxLength={500} className="font-mono text-sm" autoComplete="off" {...field}/>
            </FormControl>
            <FormDescription className="space-y-1">
              <span className="block">{t("versionCommandHint")}</span>
              <span className="block">• <code className="bg-muted px-1 py-0.5 rounded">toolname -v</code></span>
              <span className="block">• <code className="bg-muted px-1 py-0.5 rounded">toolname -V</code></span>
              <span className="block">• <code className="bg-muted px-1 py-0.5 rounded">toolname --version</code></span>
              <span className="block">• <code className="bg-muted px-1 py-0.5 rounded">python tool_name.py -v</code></span>
            </FormDescription>
            <FormMessage />
          </FormItem>)}/>
    </div>);
}
interface AddToolDialogFooterProps {
    t: TranslationFn;
    isEditMode: boolean;
    isPending: boolean;
    isFormValid: boolean;
}
export function AddToolDialogFooter({ t, isEditMode, isPending, isFormValid, }: AddToolDialogFooterProps) {
    return (<DialogFooter>
      <Button type="submit" disabled={isPending || !isFormValid} loading={isPending} loadingLabel={isEditMode ? t("saving") : t("creating")}>
        <>
          <IconPlus className="h-5 w-5"/>
          {isEditMode ? t("saveChanges") : t("createTool")}
        </>
      </Button>
    </DialogFooter>);
}
