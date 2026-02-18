"""定时扫描任务模型"""

from django.db import models
from django.contrib.postgres.fields import ArrayField


class ScheduledScan(models.Model):
    """定时扫描任务模型"""
    
    id = models.AutoField(primary_key=True)
    
    name = models.CharField(max_length=200, help_text='任务名称')
    
    engine_ids = ArrayField(
        models.IntegerField(),
        default=list,
        help_text='引擎 ID 列表'
    )
    engine_names = models.JSONField(
        default=list,
        help_text='引擎名称列表，如 ["引擎A", "引擎B"]'
    )
    yaml_configuration = models.TextField(
        default='',
        help_text='YAML 格式的扫描配置'
    )
    
    organization = models.ForeignKey(
        'targets.Organization',
        on_delete=models.CASCADE,
        related_name='scheduled_scans',
        null=True,
        blank=True,
        help_text='扫描组织（设置后执行时动态获取组织下所有目标）'
    )
    
    target = models.ForeignKey(
        'targets.Target',
        on_delete=models.CASCADE,
        related_name='scheduled_scans',
        null=True,
        blank=True,
        help_text='扫描单个目标（与 organization 二选一）'
    )
    
    cron_expression = models.CharField(
        max_length=100,
        default='0 2 * * *',
        help_text='Cron 表达式，格式：分 时 日 月 周'
    )
    
    is_enabled = models.BooleanField(default=True, db_index=True, help_text='是否启用')
    
    run_count = models.IntegerField(default=0, help_text='已执行次数')
    last_run_time = models.DateTimeField(null=True, blank=True, help_text='上次执行时间')
    next_run_time = models.DateTimeField(null=True, blank=True, help_text='下次执行时间')
    
    created_at = models.DateTimeField(auto_now_add=True, help_text='创建时间')
    updated_at = models.DateTimeField(auto_now=True, help_text='更新时间')
    
    class Meta:
        db_table = 'scheduled_scan'
        verbose_name = '定时扫描任务'
        verbose_name_plural = '定时扫描任务'
        ordering = ['-created_at']
        indexes = [
            models.Index(fields=['-created_at']),
            models.Index(fields=['is_enabled', '-created_at']),
            models.Index(fields=['name']),
        ]
    
    def __str__(self):
        return f"ScheduledScan #{self.id} - {self.name}"
