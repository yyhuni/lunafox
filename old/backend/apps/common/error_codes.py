"""
标准化错误码定义

采用简化方案（参考 Stripe、GitHub 等大厂做法）：
- 只定义 5-10 个通用错误码
- 未知错误使用通用错误码
- 错误码格式：大写字母和下划线组成
"""


class ErrorCodes:
    """标准化错误码
    
    只定义通用错误码，其他错误使用通用消息。
    这是 Stripe、GitHub 等大厂的标准做法。
    
    错误码格式规范：
    - 使用大写字母和下划线
    - 简洁明了，易于理解
    - 前端通过错误码映射到 i18n 键
    """
    
    # 通用错误码（8 个）
    VALIDATION_ERROR = 'VALIDATION_ERROR'      # 输入验证失败
    NOT_FOUND = 'NOT_FOUND'                    # 资源未找到
    PERMISSION_DENIED = 'PERMISSION_DENIED'    # 权限不足
    SERVER_ERROR = 'SERVER_ERROR'              # 服务器内部错误
    BAD_REQUEST = 'BAD_REQUEST'                # 请求格式错误
    CONFLICT = 'CONFLICT'                      # 资源冲突（如重复创建）
    UNAUTHORIZED = 'UNAUTHORIZED'              # 未认证
    RATE_LIMITED = 'RATE_LIMITED'              # 请求过于频繁
