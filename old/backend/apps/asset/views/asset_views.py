import logging
from rest_framework import viewsets, status, filters
from rest_framework.decorators import action
from rest_framework.response import Response
from apps.common.response_helpers import success_response, error_response
from apps.common.error_codes import ErrorCodes
from rest_framework.request import Request
from rest_framework.exceptions import NotFound, ValidationError as DRFValidationError
from django.core.exceptions import ValidationError, ObjectDoesNotExist
from django.db import DatabaseError, IntegrityError, OperationalError

from ..serializers import (
    SubdomainListSerializer, WebSiteSerializer, DirectorySerializer, 
    VulnerabilitySerializer, EndpointListSerializer, IPAddressAggregatedSerializer,
    SubdomainSnapshotSerializer, WebsiteSnapshotSerializer, DirectorySnapshotSerializer,
    EndpointSnapshotSerializer, VulnerabilitySnapshotSerializer
)
from ..services import (
    SubdomainService, WebSiteService, DirectoryService, 
    VulnerabilityService, AssetStatisticsService, EndpointService, HostPortMappingService
)
from ..services.snapshot import (
    SubdomainSnapshotsService, WebsiteSnapshotsService, DirectorySnapshotsService,
    EndpointSnapshotsService, HostPortMappingSnapshotsService, VulnerabilitySnapshotsService
)
from apps.common.pagination import BasePagination

logger = logging.getLogger(__name__)


class AssetStatisticsViewSet(viewsets.ViewSet):
    """
    资产统计 API
    
    提供仪表盘所需的统计数据（预聚合，读取缓存表）
    """
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = AssetStatisticsService()
    
    def list(self, request):
        """
        获取资产统计数据
        
        GET /assets/statistics/
        
        返回:
        - totalTargets: 目标总数
        - totalSubdomains: 子域名总数
        - totalIps: IP 总数
        - totalEndpoints: 端点总数
        - totalWebsites: 网站总数
        - totalVulns: 漏洞总数
        - totalAssets: 总资产数
        - runningScans: 运行中的扫描数
        - updatedAt: 统计更新时间
        """
        try:
            stats = self.service.get_statistics()
            return success_response(data={
                'totalTargets': stats['total_targets'],
                'totalSubdomains': stats['total_subdomains'],
                'totalIps': stats['total_ips'],
                'totalEndpoints': stats['total_endpoints'],
                'totalWebsites': stats['total_websites'],
                'totalVulns': stats['total_vulns'],
                'totalAssets': stats['total_assets'],
                'runningScans': stats['running_scans'],
                'updatedAt': stats['updated_at'],
                # 变化值
                'changeTargets': stats['change_targets'],
                'changeSubdomains': stats['change_subdomains'],
                'changeIps': stats['change_ips'],
                'changeEndpoints': stats['change_endpoints'],
                'changeWebsites': stats['change_websites'],
                'changeVulns': stats['change_vulns'],
                'changeAssets': stats['change_assets'],
                # 漏洞严重程度分布
                'vulnBySeverity': stats['vuln_by_severity'],
            })
        except (DatabaseError, OperationalError) as e:
            logger.exception("获取资产统计数据失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Failed to get statistics',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )

    @action(detail=False, methods=['get'], url_path='history')
    def history(self, request: Request):
        """
        获取统计历史数据（用于折线图）
        
        GET /assets/statistics/history/?days=7
        
        Query Parameters:
            days: 获取最近多少天的数据，默认 7，最大 90
        
        Returns:
            历史数据列表
        """
        try:
            days_param = request.query_params.get('days', '7')
            try:
                days = int(days_param)
            except (ValueError, TypeError):
                days = 7
            days = min(max(days, 1), 90)  # 限制在 1-90 天
            
            history = self.service.get_statistics_history(days=days)
            return success_response(data=history)
        except (DatabaseError, OperationalError) as e:
            logger.exception("获取统计历史数据失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Failed to get history data',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )


# 注意：IPAddress 模型已被重构为 HostPortMapping
# IPAddressViewSet 已删除，需要根据新架构重新实现


class SubdomainViewSet(viewsets.ModelViewSet):
    """子域名管理 ViewSet
    
    支持两种访问方式：
    1. 嵌套路由：GET /api/targets/{target_pk}/subdomains/
    2. 独立路由：GET /api/subdomains/（全局查询）
    
    支持智能过滤语法（filter 参数）：
    - name="api"         子域名模糊匹配
    - name=="api.example.com"  精确匹配
    - 多条件空格分隔     AND 关系
    """
    
    serializer_class = SubdomainListSerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering = ['-created_at']
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = SubdomainService()
    
    def get_queryset(self):
        """根据是否有 target_pk 参数决定查询范围，支持智能过滤"""
        target_pk = self.kwargs.get('target_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if target_pk:
            return self.service.get_subdomains_by_target(target_pk, filter_query=filter_query)
        return self.service.get_all(filter_query=filter_query)

    @action(detail=False, methods=['post'], url_path='bulk-create')
    def bulk_create(self, request, **kwargs):
        """批量创建子域名
        
        POST /api/targets/{target_pk}/subdomains/bulk-create/
        
        请求体:
        {
            "subdomains": ["sub1.example.com", "sub2.example.com"]
        }
        
        响应:
        {
            "data": {
                "createdCount": 10,
                "skippedCount": 2,
                "invalidCount": 1,
                "mismatchedCount": 1,
                "totalReceived": 14
            }
        }
        """
        from apps.targets.models import Target
        
        target_pk = self.kwargs.get('target_pk')
        if not target_pk:
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='Must create subdomains under a target',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        # 获取目标
        try:
            target = Target.objects.get(pk=target_pk)
        except Target.DoesNotExist:
            return error_response(
                code=ErrorCodes.NOT_FOUND,
                message='Target not found',
                status_code=status.HTTP_404_NOT_FOUND
            )
        
        # 验证目标类型必须为域名
        if target.type != Target.TargetType.DOMAIN:
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='Only domain type targets support subdomain import',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        # 获取请求体中的子域名列表
        subdomains = request.data.get('subdomains', [])
        if not subdomains or not isinstance(subdomains, list):
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='Request body cannot be empty or invalid format',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        # 调用 service 层处理
        try:
            result = self.service.bulk_create_subdomains(
                target_id=int(target_pk),
                target_name=target.name,
                subdomains=subdomains
            )
        except Exception as e:
            logger.exception("批量创建子域名失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Server internal error',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )
        
        return success_response(data={
            'createdCount': result.created_count,
            'skippedCount': result.skipped_count,
            'invalidCount': result.invalid_count,
            'mismatchedCount': result.mismatched_count,
            'totalReceived': result.total_received,
        })

    @action(detail=False, methods=['get'], url_path='export')
    def export(self, request, **kwargs):
        """导出子域名为 CSV 格式
        
        CSV 列：name, created_at
        """
        from apps.common.utils import create_csv_export_response, format_datetime
        
        target_pk = self.kwargs.get('target_pk')
        if not target_pk:
            raise DRFValidationError('必须在目标下导出')
        
        data_iterator = self.service.iter_raw_data_for_csv_export(target_id=target_pk)
        
        headers = ['name', 'created_at']
        formatters = {'created_at': format_datetime}
        
        return create_csv_export_response(
            data_iterator=data_iterator,
            headers=headers,
            filename=f"target-{target_pk}-subdomains.csv",
            field_formatters=formatters
        )

    @action(detail=False, methods=['post'], url_path='bulk-delete')
    def bulk_delete(self, request, **kwargs):
        """批量删除子域名
        
        POST /api/assets/subdomains/bulk-delete/
        
        请求体: {"ids": [1, 2, 3]}
        响应: {"deletedCount": 3}
        """
        ids = request.data.get('ids', [])
        if not ids or not isinstance(ids, list):
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='ids is required and must be a list',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        try:
            from ..models import Subdomain
            deleted_count, _ = Subdomain.objects.filter(id__in=ids).delete()
            return success_response(data={'deletedCount': deleted_count})
        except Exception as e:
            logger.exception("批量删除子域名失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Failed to delete subdomains',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )


class WebSiteViewSet(viewsets.ModelViewSet):
    """站点管理 ViewSet
    
    支持两种访问方式：
    1. 嵌套路由：GET /api/targets/{target_pk}/websites/
    2. 独立路由：GET /api/websites/（全局查询）
    
    支持智能过滤语法（filter 参数）：
    - url="api"          URL 模糊匹配
    - host="example"     主机名模糊匹配
    - title="login"      标题模糊匹配
    - status="200,301"   状态码多值匹配
    - tech="nginx"       技术栈匹配（数组字段）
    - 多条件空格分隔     AND 关系
    """
    
    serializer_class = WebSiteSerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering = ['-created_at']
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = WebSiteService()
    
    def get_queryset(self):
        """根据是否有 target_pk 参数决定查询范围，支持智能过滤"""
        target_pk = self.kwargs.get('target_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if target_pk:
            return self.service.get_websites_by_target(target_pk, filter_query=filter_query)
        return self.service.get_all(filter_query=filter_query)

    @action(detail=False, methods=['post'], url_path='bulk-create')
    def bulk_create(self, request, **kwargs):
        """批量创建网站
        
        POST /api/targets/{target_pk}/websites/bulk-create/
        
        请求体:
        {
            "urls": ["https://example.com", "https://test.com"]
        }
        
        响应:
        {
            "data": {
                "createdCount": 10
            }
        }
        """
        from apps.targets.models import Target
        
        target_pk = self.kwargs.get('target_pk')
        if not target_pk:
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='Must create websites under a target',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        # 获取目标
        try:
            target = Target.objects.get(pk=target_pk)
        except Target.DoesNotExist:
            return error_response(
                code=ErrorCodes.NOT_FOUND,
                message='Target not found',
                status_code=status.HTTP_404_NOT_FOUND
            )
        
        # 获取请求体中的 URL 列表
        urls = request.data.get('urls', [])
        if not urls or not isinstance(urls, list):
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='Request body cannot be empty or invalid format',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        # 调用 service 层处理
        try:
            created_count = self.service.bulk_create_urls(
                target_id=int(target_pk),
                target_name=target.name,
                target_type=target.type,
                urls=urls
            )
        except Exception as e:
            logger.exception("批量创建网站失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Server internal error',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )
        
        return success_response(data={
            'createdCount': created_count,
        })

    @action(detail=False, methods=['get'], url_path='export')
    def export(self, request, **kwargs):
        """导出网站为 CSV 格式
        
        CSV 列：url, host, location, title, status_code, content_length, content_type, webserver, tech, response_body, response_headers, vhost, created_at
        """
        from apps.common.utils import create_csv_export_response, format_datetime, format_list_field
        
        target_pk = self.kwargs.get('target_pk')
        if not target_pk:
            raise DRFValidationError('必须在目标下导出')
        
        data_iterator = self.service.iter_raw_data_for_csv_export(target_id=target_pk)
        
        headers = [
            'url', 'host', 'location', 'title', 'status_code',
            'content_length', 'content_type', 'webserver', 'tech',
            'response_body', 'response_headers', 'vhost', 'created_at'
        ]
        formatters = {
            'created_at': format_datetime,
            'tech': lambda x: format_list_field(x, separator=','),
        }
        
        return create_csv_export_response(
            data_iterator=data_iterator,
            headers=headers,
            filename=f"target-{target_pk}-websites.csv",
            field_formatters=formatters
        )

    @action(detail=False, methods=['post'], url_path='bulk-delete')
    def bulk_delete(self, request, **kwargs):
        """批量删除网站
        
        POST /api/assets/websites/bulk-delete/
        
        请求体: {"ids": [1, 2, 3]}
        响应: {"deletedCount": 3}
        """
        ids = request.data.get('ids', [])
        if not ids or not isinstance(ids, list):
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='ids is required and must be a list',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        try:
            from ..models import WebSite
            deleted_count, _ = WebSite.objects.filter(id__in=ids).delete()
            return success_response(data={'deletedCount': deleted_count})
        except Exception as e:
            logger.exception("批量删除网站失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Failed to delete websites',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )


class DirectoryViewSet(viewsets.ModelViewSet):
    """目录管理 ViewSet
    
    支持两种访问方式：
    1. 嵌套路由：GET /api/targets/{target_pk}/directories/
    2. 独立路由：GET /api/directories/（全局查询）
    
    支持智能过滤语法（filter 参数）：
    - url="admin"        URL 模糊匹配
    - status="200,301"   状态码多值匹配
    - 多条件空格分隔     AND 关系
    """
    
    serializer_class = DirectorySerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering = ['-created_at']
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = DirectoryService()
    
    def get_queryset(self):
        """根据是否有 target_pk 参数决定查询范围，支持智能过滤"""
        target_pk = self.kwargs.get('target_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if target_pk:
            return self.service.get_directories_by_target(target_pk, filter_query=filter_query)
        return self.service.get_all(filter_query=filter_query)

    @action(detail=False, methods=['post'], url_path='bulk-create')
    def bulk_create(self, request, **kwargs):
        """批量创建目录
        
        POST /api/targets/{target_pk}/directories/bulk-create/
        
        请求体:
        {
            "urls": ["https://example.com/admin", "https://example.com/api"]
        }
        
        响应:
        {
            "data": {
                "createdCount": 10
            }
        }
        """
        from apps.targets.models import Target
        
        target_pk = self.kwargs.get('target_pk')
        if not target_pk:
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='Must create directories under a target',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        # 获取目标
        try:
            target = Target.objects.get(pk=target_pk)
        except Target.DoesNotExist:
            return error_response(
                code=ErrorCodes.NOT_FOUND,
                message='Target not found',
                status_code=status.HTTP_404_NOT_FOUND
            )
        
        # 获取请求体中的 URL 列表
        urls = request.data.get('urls', [])
        if not urls or not isinstance(urls, list):
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='Request body cannot be empty or invalid format',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        # 调用 service 层处理
        try:
            created_count = self.service.bulk_create_urls(
                target_id=int(target_pk),
                target_name=target.name,
                target_type=target.type,
                urls=urls
            )
        except Exception as e:
            logger.exception("批量创建目录失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Server internal error',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )
        
        return success_response(data={
            'createdCount': created_count,
        })

    @action(detail=False, methods=['get'], url_path='export')
    def export(self, request, **kwargs):
        """导出目录为 CSV 格式
        
        CSV 列：url, status, content_length, words, lines, content_type, duration, created_at
        """
        from apps.common.utils import create_csv_export_response, format_datetime
        
        target_pk = self.kwargs.get('target_pk')
        if not target_pk:
            raise DRFValidationError('必须在目标下导出')
        
        data_iterator = self.service.iter_raw_data_for_csv_export(target_id=target_pk)
        
        headers = [
            'url', 'status', 'content_length', 'words',
            'lines', 'content_type', 'duration', 'created_at'
        ]
        formatters = {
            'created_at': format_datetime,
        }
        
        return create_csv_export_response(
            data_iterator=data_iterator,
            headers=headers,
            filename=f"target-{target_pk}-directories.csv",
            field_formatters=formatters
        )

    @action(detail=False, methods=['post'], url_path='bulk-delete')
    def bulk_delete(self, request, **kwargs):
        """批量删除目录
        
        POST /api/assets/directories/bulk-delete/
        
        请求体: {"ids": [1, 2, 3]}
        响应: {"deletedCount": 3}
        """
        ids = request.data.get('ids', [])
        if not ids or not isinstance(ids, list):
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='ids is required and must be a list',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        try:
            from ..models import Directory
            deleted_count, _ = Directory.objects.filter(id__in=ids).delete()
            return success_response(data={'deletedCount': deleted_count})
        except Exception as e:
            logger.exception("批量删除目录失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Failed to delete directories',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )


class EndpointViewSet(viewsets.ModelViewSet):
    """端点管理 ViewSet
    
    支持两种访问方式：
    1. 嵌套路由：GET /api/targets/{target_pk}/endpoints/
    2. 独立路由：GET /api/endpoints/（全局查询）
    
    支持智能过滤语法（filter 参数）：
    - url="api"          URL 模糊匹配
    - host="example"     主机名模糊匹配
    - title="login"      标题模糊匹配
    - status="200,301"   状态码多值匹配
    - tech="nginx"       技术栈匹配（数组字段）
    - 多条件空格分隔     AND 关系
    """
    
    serializer_class = EndpointListSerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering = ['-created_at']
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = EndpointService()
    
    def get_queryset(self):
        """根据是否有 target_pk 参数决定查询范围，支持智能过滤"""
        target_pk = self.kwargs.get('target_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if target_pk:
            return self.service.get_endpoints_by_target(target_pk, filter_query=filter_query)
        return self.service.get_all(filter_query=filter_query)

    @action(detail=False, methods=['post'], url_path='bulk-create')
    def bulk_create(self, request, **kwargs):
        """批量创建端点
        
        POST /api/targets/{target_pk}/endpoints/bulk-create/
        
        请求体:
        {
            "urls": ["https://example.com/api/v1", "https://example.com/api/v2"]
        }
        
        响应:
        {
            "data": {
                "createdCount": 10
            }
        }
        """
        from apps.targets.models import Target
        
        target_pk = self.kwargs.get('target_pk')
        if not target_pk:
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='Must create endpoints under a target',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        # 获取目标
        try:
            target = Target.objects.get(pk=target_pk)
        except Target.DoesNotExist:
            return error_response(
                code=ErrorCodes.NOT_FOUND,
                message='Target not found',
                status_code=status.HTTP_404_NOT_FOUND
            )
        
        # 获取请求体中的 URL 列表
        urls = request.data.get('urls', [])
        if not urls or not isinstance(urls, list):
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='Request body cannot be empty or invalid format',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        # 调用 service 层处理
        try:
            created_count = self.service.bulk_create_urls(
                target_id=int(target_pk),
                target_name=target.name,
                target_type=target.type,
                urls=urls
            )
        except Exception as e:
            logger.exception("批量创建端点失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Server internal error',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )
        
        return success_response(data={
            'createdCount': created_count,
        })

    @action(detail=False, methods=['get'], url_path='export')
    def export(self, request, **kwargs):
        """导出端点为 CSV 格式
        
        CSV 列：url, host, location, title, status_code, content_length, content_type, webserver, tech, response_body, response_headers, vhost, matched_gf_patterns, created_at
        """
        from apps.common.utils import create_csv_export_response, format_datetime, format_list_field
        
        target_pk = self.kwargs.get('target_pk')
        if not target_pk:
            raise DRFValidationError('必须在目标下导出')
        
        data_iterator = self.service.iter_raw_data_for_csv_export(target_id=target_pk)
        
        headers = [
            'url', 'host', 'location', 'title', 'status_code',
            'content_length', 'content_type', 'webserver', 'tech',
            'response_body', 'response_headers', 'vhost', 'matched_gf_patterns', 'created_at'
        ]
        formatters = {
            'created_at': format_datetime,
            'tech': lambda x: format_list_field(x, separator=','),
            'matched_gf_patterns': lambda x: format_list_field(x, separator=','),
        }
        
        return create_csv_export_response(
            data_iterator=data_iterator,
            headers=headers,
            filename=f"target-{target_pk}-endpoints.csv",
            field_formatters=formatters
        )

    @action(detail=False, methods=['post'], url_path='bulk-delete')
    def bulk_delete(self, request, **kwargs):
        """批量删除端点
        
        POST /api/assets/endpoints/bulk-delete/
        
        请求体: {"ids": [1, 2, 3]}
        响应: {"deletedCount": 3}
        """
        ids = request.data.get('ids', [])
        if not ids or not isinstance(ids, list):
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='ids is required and must be a list',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        try:
            from ..models import Endpoint
            deleted_count, _ = Endpoint.objects.filter(id__in=ids).delete()
            return success_response(data={'deletedCount': deleted_count})
        except Exception as e:
            logger.exception("批量删除端点失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Failed to delete endpoints',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )


class HostPortMappingViewSet(viewsets.ModelViewSet):
    """主机端口映射管理 ViewSet（IP 地址聚合视图）
    
    支持两种访问方式：
    1. 嵌套路由：GET /api/targets/{target_pk}/ip-addresses/
    2. 独立路由：GET /api/ip-addresses/（全局查询）
    
    返回按 IP 聚合的数据，每个 IP 显示其关联的所有 hosts 和 ports
    
    支持智能过滤语法（filter 参数）：
    - ip="192.168"       IP 模糊匹配
    - port="80,443"      端口多值匹配
    - host="api"         主机名模糊匹配
    - 多条件空格分隔     AND 关系
    
    注意：由于返回的是聚合数据（字典列表），不支持 DRF SearchFilter
    """
    
    serializer_class = IPAddressAggregatedSerializer
    pagination_class = BasePagination
    
    # 智能过滤字段映射
    FILTER_FIELD_MAPPING = {
        'ip': 'ip',
        'port': 'port',
        'host': 'host',
    }
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = HostPortMappingService()
    
    def get_queryset(self):
        """根据是否有 target_pk 参数决定查询范围，返回按 IP 聚合的数据
        
        支持智能过滤语法（filter 参数）
        """
        target_pk = self.kwargs.get('target_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if target_pk:
            return self.service.get_ip_aggregation_by_target(target_pk, filter_query=filter_query)
        return self.service.get_all_ip_aggregation(filter_query=filter_query)

    @action(detail=False, methods=['get'], url_path='export')
    def export(self, request, **kwargs):
        """导出 IP 地址为 CSV 格式
        
        CSV 列：ip, host, port, created_at
        """
        from apps.common.utils import create_csv_export_response, format_datetime
        
        target_pk = self.kwargs.get('target_pk')
        if not target_pk:
            raise DRFValidationError('必须在目标下导出')
        
        # 获取流式数据迭代器
        data_iterator = self.service.iter_raw_data_for_csv_export(target_id=target_pk)
        
        # CSV 表头和格式化器
        headers = ['ip', 'host', 'port', 'created_at']
        formatters = {
            'created_at': format_datetime
        }
        
        return create_csv_export_response(
            data_iterator=data_iterator,
            headers=headers,
            filename=f"target-{target_pk}-ip-addresses.csv",
            field_formatters=formatters
        )

    @action(detail=False, methods=['post'], url_path='bulk-delete')
    def bulk_delete(self, request, **kwargs):
        """批量删除 IP 地址映射
        
        POST /api/assets/ip-addresses/bulk-delete/
        
        请求体: {"ips": ["192.168.1.1", "10.0.0.1"]}
        响应: {"deletedCount": 3}
        
        注意：由于 IP 地址是聚合显示的，删除时传入 IP 列表，
        会删除该 IP 下的所有 host:port 映射记录
        """
        ips = request.data.get('ips', [])
        if not ips or not isinstance(ips, list):
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='ips is required and must be a list',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        try:
            from ..models import HostPortMapping
            deleted_count, _ = HostPortMapping.objects.filter(ip__in=ips).delete()
            return success_response(data={'deletedCount': deleted_count})
        except Exception as e:
            logger.exception("批量删除 IP 地址映射失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Failed to delete ip addresses',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )


class VulnerabilityViewSet(viewsets.ModelViewSet):
    """漏洞资产管理 ViewSet（只读）
    
    支持两种访问方式：
    1. 嵌套路由：GET /api/targets/{target_pk}/vulnerabilities/
    2. 独立路由：GET /api/vulnerabilities/（全局查询）
    
    支持智能过滤语法（filter 参数）：
    - type="xss"         漏洞类型模糊匹配
    - severity="high"    严重程度匹配
    - source="nuclei"    来源工具匹配
    - url="api"          URL 模糊匹配
    - 多条件空格分隔     AND 关系
    """
    
    serializer_class = VulnerabilitySerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering = ['-created_at']
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = VulnerabilityService()
    
    def get_queryset(self):
        """根据是否有 target_pk 参数决定查询范围，支持智能过滤"""
        target_pk = self.kwargs.get('target_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if target_pk:
            return self.service.get_vulnerabilities_by_target(target_pk, filter_query=filter_query)
        return self.service.get_all(filter_query=filter_query)


# ==================== 快照 ViewSet（Scan 嵌套路由） ====================

class SubdomainSnapshotViewSet(viewsets.ModelViewSet):
    """子域名快照 ViewSet - 嵌套路由：GET /api/scans/{scan_pk}/subdomains/
    
    支持智能过滤语法（filter 参数）：
    - name="api"         子域名模糊匹配
    - name=="api.example.com"  精确匹配
    - name!="test"       排除匹配
    """
    
    serializer_class = SubdomainSnapshotSerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering_fields = ['name', 'created_at']
    ordering = ['-created_at']
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = SubdomainSnapshotsService()
    
    def get_queryset(self):
        scan_pk = self.kwargs.get('scan_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if scan_pk:
            return self.service.get_by_scan(scan_pk, filter_query=filter_query)
        return self.service.get_all(filter_query=filter_query)

    @action(detail=False, methods=['get'], url_path='export')
    def export(self, request, **kwargs):
        """导出子域名快照为 CSV 格式
        
        CSV 列：name, created_at
        """
        from apps.common.utils import create_csv_export_response, format_datetime
        
        scan_pk = self.kwargs.get('scan_pk')
        if not scan_pk:
            raise DRFValidationError('必须在扫描下导出')
        
        data_iterator = self.service.iter_raw_data_for_csv_export(scan_id=scan_pk)
        
        headers = ['name', 'created_at']
        formatters = {'created_at': format_datetime}
        
        return create_csv_export_response(
            data_iterator=data_iterator,
            headers=headers,
            filename=f"scan-{scan_pk}-subdomains.csv",
            field_formatters=formatters
        )


class WebsiteSnapshotViewSet(viewsets.ModelViewSet):
    """网站快照 ViewSet - 嵌套路由：GET /api/scans/{scan_pk}/websites/
    
    支持智能过滤语法（filter 参数）：
    - url="api"          URL 模糊匹配
    - host="example"     主机名模糊匹配
    - title="login"      标题模糊匹配
    - status="200"       状态码匹配
    - webserver="nginx"  服务器类型匹配
    - tech="php"         技术栈匹配
    """
    
    serializer_class = WebsiteSnapshotSerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering = ['-created_at']
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = WebsiteSnapshotsService()
    
    def get_queryset(self):
        scan_pk = self.kwargs.get('scan_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if scan_pk:
            return self.service.get_by_scan(scan_pk, filter_query=filter_query)
        return self.service.get_all(filter_query=filter_query)

    @action(detail=False, methods=['get'], url_path='export')
    def export(self, request, **kwargs):
        """导出网站快照为 CSV 格式
        
        CSV 列：url, host, location, title, status_code, content_length, content_type, webserver, tech, response_body, response_headers, vhost, created_at
        """
        from apps.common.utils import create_csv_export_response, format_datetime, format_list_field
        
        scan_pk = self.kwargs.get('scan_pk')
        if not scan_pk:
            raise DRFValidationError('必须在扫描下导出')
        
        data_iterator = self.service.iter_raw_data_for_csv_export(scan_id=scan_pk)
        
        headers = [
            'url', 'host', 'location', 'title', 'status_code',
            'content_length', 'content_type', 'webserver', 'tech',
            'response_body', 'response_headers', 'vhost', 'created_at'
        ]
        formatters = {
            'created_at': format_datetime,
            'tech': lambda x: format_list_field(x, separator=','),
        }
        
        return create_csv_export_response(
            data_iterator=data_iterator,
            headers=headers,
            filename=f"scan-{scan_pk}-websites.csv",
            field_formatters=formatters
        )


class DirectorySnapshotViewSet(viewsets.ModelViewSet):
    """目录快照 ViewSet - 嵌套路由：GET /api/scans/{scan_pk}/directories/
    
    支持智能过滤语法（filter 参数）：
    - url="admin"        URL 模糊匹配
    - status="200"       状态码匹配
    - content_type="html" 内容类型匹配
    """
    
    serializer_class = DirectorySnapshotSerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering = ['-created_at']
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = DirectorySnapshotsService()
    
    def get_queryset(self):
        scan_pk = self.kwargs.get('scan_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if scan_pk:
            return self.service.get_by_scan(scan_pk, filter_query=filter_query)
        return self.service.get_all(filter_query=filter_query)

    @action(detail=False, methods=['get'], url_path='export')
    def export(self, request, **kwargs):
        """导出目录快照为 CSV 格式
        
        CSV 列：url, status, content_length, words, lines, content_type, duration, created_at
        """
        from apps.common.utils import create_csv_export_response, format_datetime
        
        scan_pk = self.kwargs.get('scan_pk')
        if not scan_pk:
            raise DRFValidationError('必须在扫描下导出')
        
        data_iterator = self.service.iter_raw_data_for_csv_export(scan_id=scan_pk)
        
        headers = [
            'url', 'status', 'content_length', 'words',
            'lines', 'content_type', 'duration', 'created_at'
        ]
        formatters = {
            'created_at': format_datetime,
        }
        
        return create_csv_export_response(
            data_iterator=data_iterator,
            headers=headers,
            filename=f"scan-{scan_pk}-directories.csv",
            field_formatters=formatters
        )


class EndpointSnapshotViewSet(viewsets.ModelViewSet):
    """端点快照 ViewSet - 嵌套路由：GET /api/scans/{scan_pk}/endpoints/
    
    支持智能过滤语法（filter 参数）：
    - url="api"          URL 模糊匹配
    - host="example"     主机名模糊匹配
    - title="login"      标题模糊匹配
    - status="200"       状态码匹配
    - webserver="nginx"  服务器类型匹配
    - tech="php"         技术栈匹配
    """
    
    serializer_class = EndpointSnapshotSerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering = ['-created_at']
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = EndpointSnapshotsService()
    
    def get_queryset(self):
        scan_pk = self.kwargs.get('scan_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if scan_pk:
            return self.service.get_by_scan(scan_pk, filter_query=filter_query)
        return self.service.get_all(filter_query=filter_query)

    @action(detail=False, methods=['get'], url_path='export')
    def export(self, request, **kwargs):
        """导出端点快照为 CSV 格式
        
        CSV 列：url, host, location, title, status_code, content_length, content_type, webserver, tech, response_body, response_headers, vhost, matched_gf_patterns, created_at
        """
        from apps.common.utils import create_csv_export_response, format_datetime, format_list_field
        
        scan_pk = self.kwargs.get('scan_pk')
        if not scan_pk:
            raise DRFValidationError('必须在扫描下导出')
        
        data_iterator = self.service.iter_raw_data_for_csv_export(scan_id=scan_pk)
        
        headers = [
            'url', 'host', 'location', 'title', 'status_code',
            'content_length', 'content_type', 'webserver', 'tech',
            'response_body', 'response_headers', 'vhost', 'matched_gf_patterns', 'created_at'
        ]
        formatters = {
            'created_at': format_datetime,
            'tech': lambda x: format_list_field(x, separator=','),
            'matched_gf_patterns': lambda x: format_list_field(x, separator=','),
        }
        
        return create_csv_export_response(
            data_iterator=data_iterator,
            headers=headers,
            filename=f"scan-{scan_pk}-endpoints.csv",
            field_formatters=formatters
        )


class HostPortMappingSnapshotViewSet(viewsets.ModelViewSet):
    """主机端口映射快照 ViewSet - 嵌套路由：GET /api/scans/{scan_pk}/ip-addresses/
    
    支持智能过滤语法（filter 参数）：
    - ip="192.168"       IP 模糊匹配
    - port="80"          端口匹配
    - host="api"         主机名模糊匹配
    
    注意：由于返回的是聚合数据（字典列表），过滤在 Service 层处理
    """
    
    serializer_class = IPAddressAggregatedSerializer
    pagination_class = BasePagination
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = HostPortMappingSnapshotsService()
    
    def get_queryset(self):
        scan_pk = self.kwargs.get('scan_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if scan_pk:
            return self.service.get_ip_aggregation_by_scan(scan_pk, filter_query=filter_query)
        return self.service.get_all_ip_aggregation(filter_query=filter_query)

    @action(detail=False, methods=['get'], url_path='export')
    def export(self, request, **kwargs):
        """导出 IP 地址为 CSV 格式
        
        CSV 列：ip, host, port, created_at
        """
        from apps.common.utils import create_csv_export_response, format_datetime
        
        scan_pk = self.kwargs.get('scan_pk')
        if not scan_pk:
            raise DRFValidationError('必须在扫描下导出')
        
        # 获取流式数据迭代器
        data_iterator = self.service.iter_raw_data_for_csv_export(scan_id=scan_pk)
        
        # CSV 表头和格式化器
        headers = ['ip', 'host', 'port', 'created_at']
        formatters = {
            'created_at': format_datetime
        }
        
        return create_csv_export_response(
            data_iterator=data_iterator,
            headers=headers,
            filename=f"scan-{scan_pk}-ip-addresses.csv",
            field_formatters=formatters
        )


class VulnerabilitySnapshotViewSet(viewsets.ModelViewSet):
    """漏洞快照 ViewSet - 嵌套路由：GET /api/scans/{scan_pk}/vulnerabilities/
    
    支持智能过滤语法（filter 参数）：
    - type="xss"         漏洞类型模糊匹配
    - url="api"          URL 模糊匹配
    - severity="high"    严重程度匹配
    - source="nuclei"    来源工具匹配
    """
    
    serializer_class = VulnerabilitySnapshotSerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering = ['-created_at']
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.service = VulnerabilitySnapshotsService()
    
    def get_queryset(self):
        scan_pk = self.kwargs.get('scan_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        if scan_pk:
            return self.service.get_by_scan(scan_pk, filter_query=filter_query)
        return self.service.get_all(filter_query=filter_query)


# ==================== 截图 ViewSet ====================

class ScreenshotViewSet(viewsets.ModelViewSet):
    """截图资产 ViewSet
    
    支持两种访问方式：
    1. 嵌套路由：GET /api/targets/{target_pk}/screenshots/
    2. 独立路由：GET /api/screenshots/（全局查询）
    
    支持智能过滤语法（filter 参数）：
    - url="example"      URL 模糊匹配
    """
    
    from ..serializers import ScreenshotListSerializer
    
    serializer_class = ScreenshotListSerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering = ['-created_at']
    
    def get_queryset(self):
        """根据是否有 target_pk 参数决定查询范围"""
        from ..models import Screenshot
        
        target_pk = self.kwargs.get('target_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        queryset = Screenshot.objects.all()
        if target_pk:
            queryset = queryset.filter(target_id=target_pk)
        
        if filter_query:
            # 简单的 URL 模糊匹配
            queryset = queryset.filter(url__icontains=filter_query)
        
        return queryset.order_by('-created_at')
    
    @action(detail=True, methods=['get'], url_path='image')
    def image(self, request, pk=None, **kwargs):
        """获取截图图片
        
        GET /api/assets/screenshots/{id}/image/
        
        返回 WebP 格式的图片二进制数据
        """
        from django.http import HttpResponse
        from ..models import Screenshot
        
        try:
            screenshot = Screenshot.objects.get(pk=pk)
            if not screenshot.image:
                return error_response(
                    code=ErrorCodes.NOT_FOUND,
                    message='Screenshot image not found',
                    status_code=status.HTTP_404_NOT_FOUND
                )
            
            response = HttpResponse(screenshot.image, content_type='image/webp')
            response['Content-Disposition'] = f'inline; filename="screenshot_{pk}.webp"'
            return response
        except Screenshot.DoesNotExist:
            return error_response(
                code=ErrorCodes.NOT_FOUND,
                message='Screenshot not found',
                status_code=status.HTTP_404_NOT_FOUND
            )
    
    @action(detail=False, methods=['post'], url_path='bulk-delete')
    def bulk_delete(self, request, **kwargs):
        """批量删除截图
        
        POST /api/assets/screenshots/bulk-delete/
        
        请求体: {"ids": [1, 2, 3]}
        响应: {"deletedCount": 3}
        """
        ids = request.data.get('ids', [])
        if not ids or not isinstance(ids, list):
            return error_response(
                code=ErrorCodes.VALIDATION_ERROR,
                message='ids is required and must be a list',
                status_code=status.HTTP_400_BAD_REQUEST
            )
        
        try:
            from ..models import Screenshot
            deleted_count, _ = Screenshot.objects.filter(id__in=ids).delete()
            return success_response(data={'deletedCount': deleted_count})
        except Exception as e:
            logger.exception("批量删除截图失败")
            return error_response(
                code=ErrorCodes.SERVER_ERROR,
                message='Failed to delete screenshots',
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR
            )


class ScreenshotSnapshotViewSet(viewsets.ModelViewSet):
    """截图快照 ViewSet - 嵌套路由：GET /api/scans/{scan_pk}/screenshots/
    
    支持智能过滤语法（filter 参数）：
    - url="example"      URL 模糊匹配
    """
    
    from ..serializers import ScreenshotSnapshotListSerializer
    
    serializer_class = ScreenshotSnapshotListSerializer
    pagination_class = BasePagination
    filter_backends = [filters.OrderingFilter]
    ordering = ['-created_at']
    
    def get_queryset(self):
        """根据 scan_pk 参数查询"""
        from ..models import ScreenshotSnapshot
        
        scan_pk = self.kwargs.get('scan_pk')
        filter_query = self.request.query_params.get('filter', None)
        
        queryset = ScreenshotSnapshot.objects.all()
        if scan_pk:
            queryset = queryset.filter(scan_id=scan_pk)
        
        if filter_query:
            # 简单的 URL 模糊匹配
            queryset = queryset.filter(url__icontains=filter_query)
        
        return queryset.order_by('-created_at')
    
    @action(detail=True, methods=['get'], url_path='image')
    def image(self, request, pk=None, **kwargs):
        """获取截图快照图片
        
        GET /api/scans/{scan_pk}/screenshots/{id}/image/
        
        返回 WebP 格式的图片二进制数据
        """
        from django.http import HttpResponse
        from ..models import ScreenshotSnapshot
        
        try:
            screenshot = ScreenshotSnapshot.objects.get(pk=pk)
            if not screenshot.image:
                return error_response(
                    code=ErrorCodes.NOT_FOUND,
                    message='Screenshot image not found',
                    status_code=status.HTTP_404_NOT_FOUND
                )
            
            response = HttpResponse(screenshot.image, content_type='image/webp')
            response['Content-Disposition'] = f'inline; filename="screenshot_snapshot_{pk}.webp"'
            return response
        except ScreenshotSnapshot.DoesNotExist:
            return error_response(
                code=ErrorCodes.NOT_FOUND,
                message='Screenshot snapshot not found',
                status_code=status.HTTP_404_NOT_FOUND
            )
