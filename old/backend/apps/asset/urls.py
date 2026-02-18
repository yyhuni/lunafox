"""
Asset 应用 URL 配置
"""

from django.urls import path, include
from rest_framework.routers import DefaultRouter
from .views import (
    SubdomainViewSet,
    WebSiteViewSet,
    DirectoryViewSet,
    VulnerabilityViewSet,
    AssetStatisticsViewSet,
    AssetSearchView,
    AssetSearchExportView,
    EndpointViewSet,
    HostPortMappingViewSet,
    ScreenshotViewSet,
)

# 创建 DRF 路由器
router = DefaultRouter()

# 注册 ViewSet
router.register(r'subdomains', SubdomainViewSet, basename='subdomain')
router.register(r'websites', WebSiteViewSet, basename='website')
router.register(r'directories', DirectoryViewSet, basename='directory')
router.register(r'endpoints', EndpointViewSet, basename='endpoint')
router.register(r'ip-addresses', HostPortMappingViewSet, basename='ip-address')
router.register(r'vulnerabilities', VulnerabilityViewSet, basename='vulnerability')
router.register(r'screenshots', ScreenshotViewSet, basename='screenshot')
router.register(r'statistics', AssetStatisticsViewSet, basename='asset-statistics')

urlpatterns = [
    path('assets/', include(router.urls)),
    path('assets/search/', AssetSearchView.as_view(), name='asset-search'),
    path('assets/search/export/', AssetSearchExportView.as_view(), name='asset-search-export'),
]
