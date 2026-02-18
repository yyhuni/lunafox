"""
自定义异常处理器

统一处理 DRF 异常，确保错误响应格式一致
"""

from rest_framework.views import exception_handler
from rest_framework import status
from rest_framework.exceptions import AuthenticationFailed, NotAuthenticated

from apps.common.response_helpers import error_response
from apps.common.error_codes import ErrorCodes


def custom_exception_handler(exc, context):
    """
    自定义异常处理器
    
    处理认证相关异常，返回统一格式的错误响应
    """
    # 先调用 DRF 默认的异常处理器
    response = exception_handler(exc, context)
    
    if response is not None:
        # 处理 401 未认证错误
        if response.status_code == status.HTTP_401_UNAUTHORIZED:
            return error_response(
                code=ErrorCodes.UNAUTHORIZED,
                message='Authentication required',
                status_code=status.HTTP_401_UNAUTHORIZED
            )
        
        # 处理 403 权限不足错误
        if response.status_code == status.HTTP_403_FORBIDDEN:
            return error_response(
                code=ErrorCodes.PERMISSION_DENIED,
                message='Permission denied',
                status_code=status.HTTP_403_FORBIDDEN
            )
    
    # 处理 NotAuthenticated 和 AuthenticationFailed 异常
    if isinstance(exc, (NotAuthenticated, AuthenticationFailed)):
        return error_response(
            code=ErrorCodes.UNAUTHORIZED,
            message='Authentication required',
            status_code=status.HTTP_401_UNAUTHORIZED
        )
    
    return response
