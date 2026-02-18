from django.db import models


class ScreenshotSnapshot(models.Model):
    """
    截图快照
    
    记录：某次扫描中捕获的网站截图
    """

    id = models.AutoField(primary_key=True)
    scan = models.ForeignKey(
        'scan.Scan',
        on_delete=models.CASCADE,
        related_name='screenshot_snapshots',
        help_text='所属的扫描任务'
    )
    url = models.TextField(help_text='截图对应的 URL')
    status_code = models.SmallIntegerField(null=True, blank=True, help_text='HTTP 响应状态码')
    image = models.BinaryField(help_text='截图 WebP 二进制数据（压缩后）')
    created_at = models.DateTimeField(auto_now_add=True, help_text='创建时间')

    class Meta:
        db_table = 'screenshot_snapshot'
        verbose_name = '截图快照'
        verbose_name_plural = '截图快照'
        ordering = ['-created_at']
        indexes = [
            models.Index(fields=['scan']),
            models.Index(fields=['-created_at']),
        ]
        constraints = [
            models.UniqueConstraint(
                fields=['scan', 'url'],
                name='unique_screenshot_per_scan_snapshot'
            ),
        ]

    def __str__(self):
        return f'{self.url} (Scan #{self.scan_id})'


class Screenshot(models.Model):
    """
    截图资产
    
    存储：目标的最新截图（从快照同步）
    """

    id = models.AutoField(primary_key=True)
    target = models.ForeignKey(
        'targets.Target',
        on_delete=models.CASCADE,
        related_name='screenshots',
        help_text='所属目标'
    )
    url = models.TextField(help_text='截图对应的 URL')
    status_code = models.SmallIntegerField(null=True, blank=True, help_text='HTTP 响应状态码')
    image = models.BinaryField(help_text='截图 WebP 二进制数据（压缩后）')
    created_at = models.DateTimeField(auto_now_add=True, help_text='创建时间')
    updated_at = models.DateTimeField(auto_now=True, help_text='更新时间')

    class Meta:
        db_table = 'screenshot'
        verbose_name = '截图'
        verbose_name_plural = '截图'
        ordering = ['-created_at']
        indexes = [
            models.Index(fields=['target']),
            models.Index(fields=['-created_at']),
        ]
        constraints = [
            models.UniqueConstraint(
                fields=['target', 'url'],
                name='unique_screenshot_per_target'
            ),
        ]

    def __str__(self):
        return f'{self.url} (Target #{self.target_id})'
