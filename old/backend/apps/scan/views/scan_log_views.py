"""
扫描日志 API

提供扫描日志查询接口，支持游标分页用于增量轮询。
"""

from rest_framework.views import APIView
from rest_framework.response import Response

from apps.scan.models import ScanLog
from apps.scan.serializers import ScanLogSerializer


class ScanLogListView(APIView):
    """
    GET /scans/{scan_id}/logs/
    
    游标分页 API，用于增量查询日志
    
    查询参数：
    - afterId: 只返回此 ID 之后的日志（用于增量轮询，避免时间戳重复导致的重复日志）
    - limit: 返回数量限制（默认 200，最大 1000）
    
    返回：
    - results: 日志列表
    - hasMore: 是否还有更多日志
    """
    
    def get(self, request, scan_id: int):
        # 参数解析
        after_id = request.query_params.get('afterId')
        try:
            limit = min(int(request.query_params.get('limit', 200)), 1000)
        except (ValueError, TypeError):
            limit = 200
        
        # 查询日志（按 ID 排序，ID 是自增的，保证顺序一致）
        queryset = ScanLog.objects.filter(scan_id=scan_id).order_by('id')
        
        # 游标过滤（使用 ID 而非时间戳，避免同一时间戳多条日志导致重复）
        if after_id:
            try:
                queryset = queryset.filter(id__gt=int(after_id))
            except (ValueError, TypeError):
                pass
        
        # 限制返回数量（多取一条用于判断 hasMore）
        logs = list(queryset[:limit + 1])
        has_more = len(logs) > limit
        if has_more:
            logs = logs[:limit]
        
        return Response({
            'results': ScanLogSerializer(logs, many=True).data,
            'hasMore': has_more,
        })
