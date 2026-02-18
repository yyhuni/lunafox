"""
WebSocket 认证基类

提供需要认证的 WebSocket Consumer 基类
"""

import logging
from channels.generic.websocket import AsyncWebsocketConsumer

logger = logging.getLogger(__name__)


class AuthenticatedWebsocketConsumer(AsyncWebsocketConsumer):
    """
    需要认证的 WebSocket Consumer 基类
    
    子类应该重写 on_connect() 方法实现具体的连接逻辑
    """
    
    async def connect(self):
        """
        连接时验证用户认证状态
        
        未认证时使用 close(code=4001) 拒绝连接
        """
        user = self.scope.get('user')
        
        if not user or not user.is_authenticated:
            logger.warning(
                f"WebSocket 连接被拒绝：用户未认证 - Path: {self.scope.get('path')}"
            )
            await self.close(code=4001)
            return
        
        # 调用子类的连接逻辑
        await self.on_connect()
    
    async def on_connect(self):
        """
        子类实现具体的连接逻辑
        
        默认实现：接受连接
        """
        await self.accept()
