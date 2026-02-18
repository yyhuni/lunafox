import React from "react"
import { useLocale, useTranslations } from "next-intl"

import { createOrganizationColumns } from "./organization-columns"
import { getDateLocale } from "@/lib/date-utils"
import {
  useBatchDeleteOrganizations,
  useDeleteOrganization,
  useOrganizations,
} from "@/hooks/use-organizations"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"

import type { Organization } from "@/types/organization.types"

export function useOrganizationListState() {
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tTooltips = useTranslations("tooltips")
  const tConfirm = useTranslations("common.confirm")
  const tOrg = useTranslations("organization")
  const locale = useLocale()

  const translations = React.useMemo(
    () => ({
      columns: {
        organization: tColumns("organization.organization"),
        description: tColumns("common.description"),
        totalTargets: tColumns("organization.totalTargets"),
        added: tColumns("organization.added"),
      },
      actions: {
        scheduleScan: tTooltips("scheduleScan"),
        editOrganization: tCommon("actions.edit"),
        delete: tCommon("actions.delete"),
        openMenu: tCommon("actions.openMenu"),
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
      tooltips: {
        organizationDetails: tTooltips("organizationDetails"),
        initiateScan: tTooltips("initiateScan"),
      },
    }),
    [tColumns, tCommon, tTooltips]
  )

  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [addDialogOpen, setAddDialogOpen] = React.useState(false)
  const [initiateScanDialogOpen, setInitiateScanDialogOpen] = React.useState(false)
  const [scheduleScanDialogOpen, setScheduleScanDialogOpen] = React.useState(false)
  const [organizationToDelete, setOrganizationToDelete] = React.useState<Organization | null>(null)
  const [organizationToEdit, setOrganizationToEdit] = React.useState<Organization | null>(null)
  const [organizationToScan, setOrganizationToScan] = React.useState<Organization | null>(null)
  const [organizationToSchedule, setOrganizationToSchedule] = React.useState<Organization | null>(null)
  const [selectedOrganizations, setSelectedOrganizations] = React.useState<Organization[]>([])
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)

  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: 10,
  })

  const [searchQuery, setSearchQuery] = React.useState("")

  const {
    data,
    isLoading,
    isFetching,
    error,
    refetch,
  } = useOrganizations(
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: searchQuery || undefined,
    },
    { enabled: true }
  )

  const { isSearching, handleSearchChange } = useSearchState({
    isFetching,
    setSearchValue: setSearchQuery,
    onResetPage: () => setPagination((prev) => ({ ...prev, pageIndex: 0 })),
  })

  const deleteOrganization = useDeleteOrganization()
  const batchDeleteOrganizations = useBatchDeleteOrganizations()

  const formatDate = React.useCallback(
    (dateString: string): string => {
      return new Date(dateString).toLocaleString(getDateLocale(locale), {
        year: "numeric",
        month: "numeric",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
      })
    },
    [locale]
  )

  const handleDelete = React.useCallback((org: Organization) => {
    setOrganizationToDelete(org)
    setDeleteDialogOpen(true)
  }, [])

  const handleEdit = React.useCallback((org: Organization) => {
    setOrganizationToEdit(org)
    setEditDialogOpen(true)
  }, [])

  const handleInitiateScan = React.useCallback((org: Organization) => {
    setOrganizationToScan(org)
    setInitiateScanDialogOpen(true)
  }, [])

  const handleScheduleScan = React.useCallback((org: Organization) => {
    setOrganizationToSchedule(org)
    setScheduleScanDialogOpen(true)
  }, [])

  const columns = React.useMemo(
    () =>
      createOrganizationColumns({
        formatDate,
        handleEdit,
        handleDelete,
        handleInitiateScan,
        handleScheduleScan,
        t: translations,
      }),
    [
      formatDate,
      handleEdit,
      handleDelete,
      handleInitiateScan,
      handleScheduleScan,
      translations,
    ]
  )

  const confirmDelete = async () => {
    if (!organizationToDelete) return

    setDeleteDialogOpen(false)
    setOrganizationToDelete(null)

    deleteOrganization.mutate(Number(organizationToDelete.id))
  }

  const handleOrganizationEdited = () => {
    setEditDialogOpen(false)
    setOrganizationToEdit(null)
  }

  const handleBulkDelete = () => {
    if (selectedOrganizations.length === 0) {
      return
    }
    setBulkDeleteDialogOpen(true)
  }

  const confirmBulkDelete = async () => {
    if (selectedOrganizations.length === 0) return

    const deletedIds = selectedOrganizations.map((org) => Number(org.id))

    setBulkDeleteDialogOpen(false)
    setSelectedOrganizations([])

    batchDeleteOrganizations.mutate(deletedIds)
  }

  const handlePaginationChange = React.useCallback(
    (newPagination: { pageIndex: number; pageSize: number }) => {
      setPagination(newPagination)
    },
    []
  )

  const paginationInfo = data
    ? buildPaginationInfo({
        ...normalizePagination(
          data.pagination,
          pagination.pageIndex + 1,
          pagination.pageSize
        ),
        minTotalPages: 1,
      })
    : undefined

  return {
    tCommon,
    tConfirm,
    tOrg,
    data,
    organizations: data?.organizations ?? [],
    isLoading,
    error,
    refetch,
    translations,
    columns,
    pagination,
    setPagination,
    paginationInfo,
    handlePaginationChange,
    searchQuery,
    isSearching,
    handleSearchChange,
    deleteDialogOpen,
    setDeleteDialogOpen,
    editDialogOpen,
    setEditDialogOpen,
    addDialogOpen,
    setAddDialogOpen,
    initiateScanDialogOpen,
    setInitiateScanDialogOpen,
    scheduleScanDialogOpen,
    setScheduleScanDialogOpen,
    organizationToDelete,
    organizationToEdit,
    organizationToScan,
    organizationToSchedule,
    setOrganizationToScan,
    setOrganizationToSchedule,
    selectedOrganizations,
    setSelectedOrganizations,
    bulkDeleteDialogOpen,
    setBulkDeleteDialogOpen,
    deleteOrganization,
    batchDeleteOrganizations,
    handleDelete,
    handleEdit,
    handleInitiateScan,
    handleScheduleScan,
    confirmDelete,
    handleOrganizationEdited,
    handleBulkDelete,
    confirmBulkDelete,
  }
}

export type OrganizationListState = ReturnType<typeof useOrganizationListState>
