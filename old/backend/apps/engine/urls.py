from django.urls import path, include
from rest_framework.routers import DefaultRouter

from .views import (
    ScanEngineViewSet,
    WorkerNodeViewSet,
    WordlistViewSet,
    NucleiTemplateRepoViewSet,
)
from .views.fingerprints import (
    EholeFingerprintViewSet,
    GobyFingerprintViewSet,
    WappalyzerFingerprintViewSet,
    FingersFingerprintViewSet,
    FingerPrintHubFingerprintViewSet,
    ARLFingerprintViewSet,
)


# 创建路由器
router = DefaultRouter()
router.register(r"engines", ScanEngineViewSet, basename="engine")
router.register(r"workers", WorkerNodeViewSet, basename="worker")
router.register(r"wordlists", WordlistViewSet, basename="wordlist")
router.register(r"nuclei/repos", NucleiTemplateRepoViewSet, basename="nuclei-repos")
# 指纹管理
router.register(r"fingerprints/ehole", EholeFingerprintViewSet, basename="ehole-fingerprint")
router.register(r"fingerprints/goby", GobyFingerprintViewSet, basename="goby-fingerprint")
router.register(r"fingerprints/wappalyzer", WappalyzerFingerprintViewSet, basename="wappalyzer-fingerprint")
router.register(r"fingerprints/fingers", FingersFingerprintViewSet, basename="fingers-fingerprint")
router.register(r"fingerprints/fingerprinthub", FingerPrintHubFingerprintViewSet, basename="fingerprinthub-fingerprint")
router.register(r"fingerprints/arl", ARLFingerprintViewSet, basename="arl-fingerprint")

urlpatterns = [
    path("", include(router.urls)),
]

