package catalogwiring

import catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"

var _ catalogapp.TargetQueryStore = (*catalogTargetStoreAdapter)(nil)
var _ catalogapp.TargetCommandStore = (*catalogTargetStoreAdapter)(nil)
var _ catalogapp.TargetBatchCommandStore = (*catalogTargetStoreAdapter)(nil)
var _ catalogapp.TransactionCoordinator = (*catalogTransactionCoordinator)(nil)

var _ catalogapp.OrganizationTargetBindingStore = (*catalogOrganizationTargetBindingStoreAdapter)(nil)
var _ catalogapp.OrganizationTargetBindingContextStore = (*catalogOrganizationTargetBindingStoreAdapter)(nil)

var _ catalogapp.WordlistQueryStore = (*catalogWordlistStoreAdapter)(nil)
var _ catalogapp.WordlistCommandStore = (*catalogWordlistStoreAdapter)(nil)

var _ catalogapp.EngineCatalogQueryStore = (*engineCatalogQueryStoreAdapter)(nil)
var _ catalogapp.SubfinderAPIKeySettingsStore = (*catalogSubfinderProviderSettingsStoreAdapter)(nil)
var _ catalogapp.ExecutionSubfinderProviderSettingsStore = (*catalogSubfinderProviderSettingsStoreAdapter)(nil)
