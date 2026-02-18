from django.urls import path, include
from rest_framework.routers import DefaultRouter
from .views import OrganizationViewSet, TargetViewSet
from apps.asset.views import (
    SubdomainViewSet, WebSiteViewSet, DirectoryViewSet,
    EndpointViewSet, HostPortMappingViewSet, VulnerabilityViewSet,
    ScreenshotViewSet
)

# 创建路由器
router = DefaultRouter()

# 注册 ViewSet
router.register(r'organizations', OrganizationViewSet, basename='organization')
router.register(r'targets', TargetViewSet, basename='target')

# Target 下的嵌套资产路由
target_subdomains_list = SubdomainViewSet.as_view({'get': 'list'})
target_subdomains_export = SubdomainViewSet.as_view({'get': 'export'})
target_subdomains_bulk_create = SubdomainViewSet.as_view({'post': 'bulk_create'})
target_websites_list = WebSiteViewSet.as_view({'get': 'list'})
target_websites_export = WebSiteViewSet.as_view({'get': 'export'})
target_websites_bulk_create = WebSiteViewSet.as_view({'post': 'bulk_create'})
target_directories_list = DirectoryViewSet.as_view({'get': 'list'})
target_directories_export = DirectoryViewSet.as_view({'get': 'export'})
target_directories_bulk_create = DirectoryViewSet.as_view({'post': 'bulk_create'})
target_endpoints_list = EndpointViewSet.as_view({'get': 'list'})
target_endpoints_export = EndpointViewSet.as_view({'get': 'export'})
target_endpoints_bulk_create = EndpointViewSet.as_view({'post': 'bulk_create'})
target_ip_addresses_list = HostPortMappingViewSet.as_view({'get': 'list'})
target_ip_addresses_export = HostPortMappingViewSet.as_view({'get': 'export'})
target_vulnerabilities_list = VulnerabilityViewSet.as_view({'get': 'list'})
target_screenshots_list = ScreenshotViewSet.as_view({'get': 'list'})
target_screenshots_bulk_delete = ScreenshotViewSet.as_view({'post': 'bulk_delete'})

urlpatterns = [
    path('', include(router.urls)),
    # 嵌套路由：/api/targets/{target_pk}/xxx/
    path('targets/<int:target_pk>/subdomains/', target_subdomains_list, name='target-subdomains-list'),
    path('targets/<int:target_pk>/subdomains/export/', target_subdomains_export, name='target-subdomains-export'),
    path('targets/<int:target_pk>/subdomains/bulk-create/', target_subdomains_bulk_create, name='target-subdomains-bulk-create'),
    path('targets/<int:target_pk>/websites/', target_websites_list, name='target-websites-list'),
    path('targets/<int:target_pk>/websites/export/', target_websites_export, name='target-websites-export'),
    path('targets/<int:target_pk>/websites/bulk-create/', target_websites_bulk_create, name='target-websites-bulk-create'),
    path('targets/<int:target_pk>/directories/', target_directories_list, name='target-directories-list'),
    path('targets/<int:target_pk>/directories/export/', target_directories_export, name='target-directories-export'),
    path('targets/<int:target_pk>/directories/bulk-create/', target_directories_bulk_create, name='target-directories-bulk-create'),
    path('targets/<int:target_pk>/endpoints/', target_endpoints_list, name='target-endpoints-list'),
    path('targets/<int:target_pk>/endpoints/export/', target_endpoints_export, name='target-endpoints-export'),
    path('targets/<int:target_pk>/endpoints/bulk-create/', target_endpoints_bulk_create, name='target-endpoints-bulk-create'),
    path('targets/<int:target_pk>/ip-addresses/', target_ip_addresses_list, name='target-ip-addresses-list'),
    path('targets/<int:target_pk>/ip-addresses/export/', target_ip_addresses_export, name='target-ip-addresses-export'),
    path('targets/<int:target_pk>/vulnerabilities/', target_vulnerabilities_list, name='target-vulnerabilities-list'),
    path('targets/<int:target_pk>/screenshots/', target_screenshots_list, name='target-screenshots-list'),
    path('targets/<int:target_pk>/screenshots/bulk-delete/', target_screenshots_bulk_delete, name='target-screenshots-bulk-delete'),
]
