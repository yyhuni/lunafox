"""黑名单规则模型"""
from django.db import models


class BlacklistRule(models.Model):
    """黑名单规则模型
    
    用于存储黑名单过滤规则，支持域名、IP、CIDR 三种类型。
    支持两层作用域：全局规则和 Target 级规则。
    """
    
    class RuleType(models.TextChoices):
        DOMAIN = 'domain', '域名'
        IP = 'ip', 'IP地址'
        CIDR = 'cidr', 'CIDR范围'
        KEYWORD = 'keyword', '关键词'
    
    class Scope(models.TextChoices):
        GLOBAL = 'global', '全局规则'
        TARGET = 'target', 'Target规则'
    
    id = models.AutoField(primary_key=True)
    pattern = models.CharField(
        max_length=255, 
        help_text='规则模式，如 *.gov, 10.0.0.0/8, 192.168.1.1'
    )
    rule_type = models.CharField(
        max_length=20, 
        choices=RuleType.choices,
        help_text='规则类型：domain, ip, cidr'
    )
    scope = models.CharField(
        max_length=20, 
        choices=Scope.choices, 
        db_index=True,
        help_text='作用域：global 或 target'
    )
    target = models.ForeignKey(
        'targets.Target',
        on_delete=models.CASCADE,
        null=True, 
        blank=True,
        related_name='blacklist_rules',
        help_text='关联的 Target（仅 scope=target 时有值）'
    )
    description = models.CharField(
        max_length=500, 
        blank=True, 
        default='', 
        help_text='规则描述'
    )
    created_at = models.DateTimeField(auto_now_add=True)
    
    class Meta:
        db_table = 'blacklist_rule'
        indexes = [
            models.Index(fields=['scope', 'rule_type']),
            models.Index(fields=['target', 'scope']),
        ]
        constraints = [
            models.UniqueConstraint(
                fields=['pattern', 'scope', 'target'],
                name='unique_blacklist_rule'
            ),
        ]
        ordering = ['-created_at']
    
    def __str__(self):
        if self.scope == self.Scope.TARGET and self.target:
            return f"[{self.scope}:{self.target_id}] {self.pattern}"
        return f"[{self.scope}] {self.pattern}"
