# 扫描流程架构

## 完整扫描流程

```mermaid
flowchart TB
    START[Start Scan]
    TARGET[Input Target]
    
    START --> TARGET
    
    subgraph STAGE1["Stage 1: Discovery Sequential"]
        direction TB
        
        subgraph SUB["Subdomain Discovery"]
            direction TB
            SUBFINDER[subfinder]
            AMASS[amass]
            SUBLIST3R[sublist3r]
            ASSETFINDER[assetfinder]
            MERGE[Merge & Deduplicate]
            BRUTEFORCE[puredns bruteforce<br/>Dictionary Attack]
            MUTATE[dnsgen + puredns<br/>Mutation Generation]
            RESOLVE[puredns resolve<br/>Alive Verification]
            
            SUBFINDER --> MERGE
            AMASS --> MERGE
            SUBLIST3R --> MERGE
            ASSETFINDER --> MERGE
            MERGE --> BRUTEFORCE
            BRUTEFORCE --> MUTATE
            MUTATE --> RESOLVE
        end
        
        subgraph PORT["Port Scan"]
            NAABU[naabu<br/>Port Discovery]
        end
        
        subgraph SITE["Site Scan"]
            HTTPX1[httpx<br/>Web Service Detection]
        end
        
        subgraph FINGER["Fingerprint Detect"]
            XINGFINGER[xingfinger<br/>Tech Stack Detection]
        end
        
        RESOLVE --> NAABU
        NAABU --> HTTPX1
        HTTPX1 --> XINGFINGER
    end
    
    TARGET --> SUBFINDER
    TARGET --> AMASS
    TARGET --> SUBLIST3R
    TARGET --> ASSETFINDER
    
    subgraph STAGE2["Stage 2: URL Collection Parallel"]
        direction TB
        
        subgraph URL["URL Fetch"]
            direction TB
            WAYMORE[waymore<br/>Historical URLs]
            KATANA[katana<br/>Crawler]
            URO[uro<br/>URL Deduplication]
            HTTPX2[httpx<br/>Alive Verification]
            
            WAYMORE --> URO
            KATANA --> URO
            URO --> HTTPX2
        end
        
        subgraph DIR["Directory Scan"]
            FFUF[ffuf<br/>Directory Bruteforce]
        end
    end
    
    XINGFINGER --> WAYMORE
    XINGFINGER --> KATANA
    XINGFINGER --> FFUF
    
    subgraph STAGE3["Stage 3: Screenshot Sequential"]
        direction TB
        SCREENSHOT[Playwright<br/>Page Screenshot]
    end
    
    HTTPX2 --> SCREENSHOT
    FFUF --> SCREENSHOT
    
    subgraph STAGE4["Stage 4: Vulnerability Sequential"]
        direction TB
        
        subgraph VULN["Vulnerability Scan"]
            direction LR
            DALFOX[dalfox<br/>XSS Scan]
            NUCLEI[nuclei<br/>Vulnerability Scan]
        end
    end
    
    SCREENSHOT --> DALFOX
    SCREENSHOT --> NUCLEI
    
    DALFOX --> FINISH
    NUCLEI --> FINISH
    
    FINISH[Scan Complete]
    
    style START fill:#ff9999
    style FINISH fill:#99ff99
    style TARGET fill:#ffcc99
    style STAGE1 fill:#e6f3ff
    style STAGE2 fill:#fff4e6
    style STAGE3 fill:#ffe6f0
```

## 执行阶段定义

```python
# backend/apps/scan/configs/command_templates.py
# Stage 1: 资产发现 - 子域名 → 端口 → 站点探测 → 指纹识别
# Stage 2: URL 收集 - URL 获取 + 目录扫描（并行）
# Stage 3: 截图 - 在 URL 收集完成后执行，捕获更多发现的页面
# Stage 4: 漏洞扫描 - 最后执行
EXECUTION_STAGES = [
    {'mode': 'sequential', 'flows': ['subdomain_discovery', 'port_scan', 'site_scan', 'fingerprint_detect']},
    {'mode': 'parallel', 'flows': ['url_fetch', 'directory_scan']},
    {'mode': 'sequential', 'flows': ['screenshot']},
    {'mode': 'sequential', 'flows': ['vuln_scan']},
]
```

## 各阶段输出

| Flow | 工具 | 输出表 |
|------|------|--------|
| subdomain_discovery | subfinder, amass, sublist3r, assetfinder, puredns | Subdomain |
| port_scan | naabu | HostPortMapping |
| site_scan | httpx | WebSite |
| fingerprint_detect | xingfinger | WebSite.tech（更新） |
| url_fetch | waymore, katana, uro, httpx | Endpoint |
| directory_scan | ffuf | Directory |
| screenshot | Playwright | Screenshot |
| vuln_scan | dalfox, nuclei | Vulnerability |
