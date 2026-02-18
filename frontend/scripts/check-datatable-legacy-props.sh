#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

LEGACY_PROPS_REGEX="pagination|setPagination|paginationInfo|onPaginationChange|hidePagination|pageSizeOptions|hideToolbar|toolbarLeft|toolbarRight|searchMode|searchPlaceholder|searchValue|onSearch|isSearching|filterFields|filterExamples|enableRowSelection|rowSelection|onRowSelectionChange|onSelectionChange|onBulkDelete|bulkDeleteLabel|showBulkDelete|onAddNew|onAddHover|addButtonLabel|showAddButton|onBulkAdd|bulkAddLabel|showBulkAdd|downloadOptions|columnVisibility|onColumnVisibilityChange|sorting|onSortingChange|defaultSorting|emptyMessage|emptyComponent|deleteConfirmation|enableAutoColumnSizing"

COMPAT_REFS="$(
  rg -n "props\\.(${LEGACY_PROPS_REGEX})\\b" components/ui/data-table/unified-data-table.tsx || true
)"

if [[ -n "$COMPAT_REFS" ]]; then
  echo "Found legacy compatibility references in unified-data-table.tsx:"
  echo "$COMPAT_REFS"
  exit 1
fi

MATCHES="$(
  rg -nUP "<UnifiedDataTable[\\s\\S]{0,1200}?\\n\\s{4,12}(${LEGACY_PROPS_REGEX})\\s*=" components || true
)"

if [[ -n "$MATCHES" ]]; then
  echo "Found legacy UnifiedDataTable props usage:"
  echo "$MATCHES"
  exit 1
fi

echo "UnifiedDataTable legacy props check passed."
