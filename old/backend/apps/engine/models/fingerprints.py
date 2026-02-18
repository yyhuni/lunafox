"""指纹相关 Models

包含 EHole、Goby、Wappalyzer 等指纹格式的数据模型
"""

from django.db import models


class GobyFingerprint(models.Model):
    """Goby 格式指纹规则
    
    Goby 使用逻辑表达式和规则数组进行匹配：
    - logic: 逻辑表达式，如 "a||b", "(a&&b)||c"
    - rule: 规则数组，每条规则包含 label, feature, is_equal
    """
    
    name = models.CharField(max_length=300, unique=True, help_text='产品名称')
    logic = models.CharField(max_length=500, help_text='逻辑表达式')
    rule = models.JSONField(default=list, help_text='规则数组')
    created_at = models.DateTimeField(auto_now_add=True)
    
    class Meta:
        db_table = 'goby_fingerprint'
        verbose_name = 'Goby 指纹'
        verbose_name_plural = 'Goby 指纹'
        ordering = ['-created_at']
        indexes = [
            models.Index(fields=['name']),
            models.Index(fields=['logic']),
            models.Index(fields=['-created_at']),
        ]
    
    def __str__(self) -> str:
        return f"{self.name} ({self.logic})"


class EholeFingerprint(models.Model):
    """EHole 格式指纹规则（字段与 ehole.json 一致）"""
    
    cms = models.CharField(max_length=200, help_text='产品/CMS名称')
    method = models.CharField(max_length=200, default='keyword', help_text='匹配方式')
    location = models.CharField(max_length=200, default='body', help_text='匹配位置')
    keyword = models.JSONField(default=list, help_text='关键词列表')
    is_important = models.BooleanField(default=False, help_text='是否重点资产')
    type = models.CharField(max_length=100, blank=True, default='-', help_text='分类')
    created_at = models.DateTimeField(auto_now_add=True)
    
    class Meta:
        db_table = 'ehole_fingerprint'
        verbose_name = 'EHole 指纹'
        verbose_name_plural = 'EHole 指纹'
        ordering = ['-created_at']
        indexes = [
            # 搜索过滤字段索引
            models.Index(fields=['cms']),
            models.Index(fields=['method']),
            models.Index(fields=['location']),
            models.Index(fields=['type']),
            models.Index(fields=['is_important']),
            # 排序字段索引
            models.Index(fields=['-created_at']),
        ]
        constraints = [
            # 唯一约束：cms + method + location 组合不能重复
            models.UniqueConstraint(
                fields=['cms', 'method', 'location'],
                name='unique_ehole_fingerprint'
            ),
        ]
    
    def __str__(self) -> str:
        return f"{self.cms} ({self.method}@{self.location})"


class WappalyzerFingerprint(models.Model):
    """Wappalyzer 格式指纹规则
    
    Wappalyzer 支持多种检测方式：cookies, headers, scriptSrc, js, meta, html 等
    """
    
    name = models.CharField(max_length=300, unique=True, help_text='应用名称')
    cats = models.JSONField(default=list, help_text='分类 ID 数组')
    cookies = models.JSONField(default=dict, blank=True, help_text='Cookie 检测规则')
    headers = models.JSONField(default=dict, blank=True, help_text='HTTP Header 检测规则')
    script_src = models.JSONField(default=list, blank=True, help_text='脚本 URL 正则数组')
    js = models.JSONField(default=list, blank=True, help_text='JavaScript 变量检测规则')
    implies = models.JSONField(default=list, blank=True, help_text='依赖关系数组')
    meta = models.JSONField(default=dict, blank=True, help_text='HTML meta 标签检测规则')
    html = models.JSONField(default=list, blank=True, help_text='HTML 内容正则数组')
    description = models.TextField(blank=True, default='', help_text='应用描述')
    website = models.URLField(max_length=500, blank=True, default='', help_text='官网链接')
    cpe = models.CharField(max_length=300, blank=True, default='', help_text='CPE 标识符')
    created_at = models.DateTimeField(auto_now_add=True)
    
    class Meta:
        db_table = 'wappalyzer_fingerprint'
        verbose_name = 'Wappalyzer 指纹'
        verbose_name_plural = 'Wappalyzer 指纹'
        ordering = ['-created_at']
        indexes = [
            models.Index(fields=['name']),
            models.Index(fields=['website']),
            models.Index(fields=['cpe']),
            models.Index(fields=['-created_at']),
        ]
    
    def __str__(self) -> str:
        return f"{self.name}"


class FingersFingerprint(models.Model):
    """Fingers 格式指纹规则 (fingers_http.json)
    
    使用正则表达式和标签进行匹配，支持 favicon hash、header、body 等多种检测方式
    """
    
    name = models.CharField(max_length=300, unique=True, help_text='指纹名称')
    link = models.URLField(max_length=500, blank=True, default='', help_text='相关链接')
    rule = models.JSONField(default=list, help_text='匹配规则数组')
    tag = models.JSONField(default=list, help_text='标签数组')
    focus = models.BooleanField(default=False, help_text='是否重点关注')
    default_port = models.JSONField(default=list, blank=True, help_text='默认端口数组')
    created_at = models.DateTimeField(auto_now_add=True)
    
    class Meta:
        db_table = 'fingers_fingerprint'
        verbose_name = 'Fingers 指纹'
        verbose_name_plural = 'Fingers 指纹'
        ordering = ['-created_at']
        indexes = [
            models.Index(fields=['name']),
            models.Index(fields=['link']),
            models.Index(fields=['focus']),
            models.Index(fields=['-created_at']),
        ]
    
    def __str__(self) -> str:
        return f"{self.name}"


class FingerPrintHubFingerprint(models.Model):
    """FingerPrintHub 格式指纹规则 (fingerprinthub_web.json)
    
    基于 nuclei 模板格式，使用 HTTP 请求和响应特征进行匹配
    """
    
    fp_id = models.CharField(max_length=200, unique=True, help_text='指纹ID')
    name = models.CharField(max_length=300, help_text='指纹名称')
    author = models.CharField(max_length=200, blank=True, default='', help_text='作者')
    tags = models.CharField(max_length=500, blank=True, default='', help_text='标签')
    severity = models.CharField(max_length=50, blank=True, default='info', help_text='严重程度')
    metadata = models.JSONField(default=dict, blank=True, help_text='元数据')
    http = models.JSONField(default=list, help_text='HTTP 匹配规则')
    source_file = models.CharField(max_length=500, blank=True, default='', help_text='来源文件')
    created_at = models.DateTimeField(auto_now_add=True)
    
    class Meta:
        db_table = 'fingerprinthub_fingerprint'
        verbose_name = 'FingerPrintHub 指纹'
        verbose_name_plural = 'FingerPrintHub 指纹'
        ordering = ['-created_at']
        indexes = [
            models.Index(fields=['fp_id']),
            models.Index(fields=['name']),
            models.Index(fields=['author']),
            models.Index(fields=['severity']),
            models.Index(fields=['-created_at']),
        ]
    
    def __str__(self) -> str:
        return f"{self.name} ({self.fp_id})"


class ARLFingerprint(models.Model):
    """ARL 格式指纹规则 (ARL.yaml)
    
    使用简单的 name + rule 表达式格式
    """
    
    name = models.CharField(max_length=300, unique=True, help_text='指纹名称')
    rule = models.TextField(help_text='匹配规则表达式')
    created_at = models.DateTimeField(auto_now_add=True)
    
    class Meta:
        db_table = 'arl_fingerprint'
        verbose_name = 'ARL 指纹'
        verbose_name_plural = 'ARL 指纹'
        ordering = ['-created_at']
        indexes = [
            models.Index(fields=['name']),
            models.Index(fields=['-created_at']),
        ]
    
    def __str__(self) -> str:
        return f"{self.name}"
