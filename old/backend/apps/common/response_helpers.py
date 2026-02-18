"""
标准化 API 响应辅助函数

遵循行业标准（RFC 9457 Problem Details）和大厂实践（Google、Stripe、GitHub）：
- 成功响应只包含数据，不包含 message 字段
- 错误响应使用机器可读的错误码，前端映射到 i18n 消息
"""
from typing import Any, Dict, List, Optional, Union

from rest_framework import status
from rest_framework.response import Response


def success_response(
    data: Optional[Union[Dict[str, Any], List[Any]]] = None,
    status_code: int = status.HTTP_200_OK
) -> Response:
    """
    标准化成功响应
    
    直接返回数据，不做包装，符合 Stripe/GitHub 等大厂标准。
    
    Args:
        data: 响应数据（dict 或 list）
        status_code: HTTP 状态码，默认 200
    
    Returns:
        Response: DRF Response 对象
    
    Examples:
        # 单个资源
        >>> success_response(data={'id': 1, 'name': 'Test'})
        {'id': 1, 'name': 'Test'}
        
        # 操作结果
        >>> success_response(data={'count': 3, 'scans': [...]})
        {'count': 3, 'scans': [...]}
        
        # 创建资源
        >>> success_response(data={'id': 1}, status_code=201)
    """
    # 注意：不能使用 data or {}，因为空列表 [] 会被转换为 {}
    if data is None:
        data = {}
    return Response(data, status=status_code)


def error_response(
    code: str,
    message: Optional[str] = None,
    details: Optional[List[Dict[str, Any]]] = None,
    status_code: int = status.HTTP_400_BAD_REQUEST
) -> Response:
    """
    标准化错误响应
    
    Args:
        code: 错误码（如 'VALIDATION_ERROR', 'NOT_FOUND'）
              格式：大写字母和下划线组成
        message: 开发者调试信息（非用户显示）
        details: 详细错误信息（如字段级验证错误）
        status_code: HTTP 状态码，默认 400
    
    Returns:
        Response: DRF Response 对象
    
    Examples:
        # 简单错误
        >>> error_response(code='NOT_FOUND', status_code=404)
        {'error': {'code': 'NOT_FOUND'}}
        
        # 带调试信息
        >>> error_response(
        ...     code='VALIDATION_ERROR',
        ...     message='Invalid input data',
        ...     details=[{'field': 'name', 'message': 'Required'}]
        ... )
        {'error': {'code': 'VALIDATION_ERROR', 'message': '...', 'details': [...]}}
    """
    error_body: Dict[str, Any] = {'code': code}
    
    if message:
        error_body['message'] = message
    
    if details:
        error_body['details'] = details
    
    return Response({'error': error_body}, status=status_code)
