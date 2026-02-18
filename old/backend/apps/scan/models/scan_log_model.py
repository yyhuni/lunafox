"""扫描日志模型"""

from django.db import models


class ScanLog(models.Model):
    """扫描日志模型"""
    
    class Level(models.TextChoices):
        INFO = 'info', 'Info'
        WARNING = 'warning', 'Warning'
        ERROR = 'error', 'Error'
    
    id = models.BigAutoField(primary_key=True)
    scan = models.ForeignKey(
        'Scan',
        on_delete=models.CASCADE,
        related_name='logs',
        db_index=True,
        help_text='关联的扫描任务'
    )
    level = models.CharField(
        max_length=10,
        choices=Level.choices,
        default=Level.INFO,
        help_text='日志级别'
    )
    content = models.TextField(help_text='日志内容')
    created_at = models.DateTimeField(auto_now_add=True, db_index=True, help_text='创建时间')
    
    class Meta:
        db_table = 'scan_log'
        verbose_name = '扫描日志'
        verbose_name_plural = '扫描日志'
        ordering = ['created_at']
        indexes = [
            models.Index(fields=['scan', 'created_at']),
        ]
    
    def __str__(self):
        return f"[{self.level}] {self.content[:50]}"
