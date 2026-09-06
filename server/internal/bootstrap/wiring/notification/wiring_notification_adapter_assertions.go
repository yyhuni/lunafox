package notificationwiring

import (
	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	notificationrepo "github.com/yyhuni/lunafox/server/internal/modules/notification/repository"
)

var _ notificationapp.OutboxStore = (*notificationrepo.OutboxRepository)(nil)
var _ notificationapp.InboxStore = (*notificationrepo.InboxRepository)(nil)
var _ notificationapp.LocaleStore = (*notificationrepo.InboxRepository)(nil)
var _ notificationapp.DestinationStore = (*notificationrepo.DestinationRepository)(nil)
var _ notificationapp.DestinationFanoutStore = (*notificationrepo.DestinationRepository)(nil)
var _ notificationapp.DeliveryStore = (*notificationrepo.DeliveryRepository)(nil)
var _ notificationapp.MaterializationTransaction = (*notificationrepo.TransactionCoordinator)(nil)
var _ notificationapp.FactStore = (*notificationrepo.FactRepository)(nil)
var _ notificationapp.AudienceStore = (*notificationrepo.AudienceRepository)(nil)
var _ notificationapp.ActiveSuperuserAuthorizer = (*notificationrepo.ActiveSuperuserRepository)(nil)
