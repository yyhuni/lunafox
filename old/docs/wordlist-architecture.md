# 字典文件管理架构

本文档介绍 XingRin 中字典文件的存储、同步和使用机制。

## 目录结构

```
/opt/xingrin/wordlists/
├── common.txt              # 通用字典
├── subdomains.txt          # 子域名字典
├── directories.txt         # 目录字典
└── ...
```

## 一、存储位置

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `WORDLISTS_BASE_PATH` | `/opt/xingrin/wordlists` | 字典文件存储目录 |

## 二、数据模型

```
Wordlist
├── id          # 字典 ID
├── name        # 字典名称（唯一，用于查询）
├── description # 描述
├── file_path   # 文件绝对路径
├── file_size   # 文件大小（字节）
├── line_count  # 行数
└── file_hash   # SHA256 哈希值（用于校验）
```

## 三、Server 端上传流程

1. 用户在前端上传字典文件
2. `WordlistService.create_wordlist()` 处理：
   - 保存文件到 `WORDLISTS_BASE_PATH` 目录
   - 计算 SHA256 哈希值
   - 统计文件大小和行数
   - 创建数据库记录

```mermaid
flowchart TB
    subgraph SERVER["🖥️ Server 容器"]
        direction TB
        
        subgraph UI["前端 UI"]
            direction LR
            UPLOAD["📤 上传字典<br/>选择文件"]
            EDIT["✏️ 编辑内容<br/>在线修改"]
            DELETE["🗑️ 删除字典"]
        end
        
        UPLOAD --> API
        EDIT --> API
        
        subgraph API["API 层"]
            VIEWSET["WordlistViewSet<br/>POST /api/wordlists/<br/>PUT .../content/"]
        end
        
        API --> SERVICE
        
        subgraph SERVICE["业务逻辑层"]
            CREATE["create_wordlist()<br/>创建字典"]
            UPDATE["update_wordlist_content()<br/>更新字典内容"]
        end
        
        CREATE --> PROCESS
        UPDATE --> PROCESS
        
        subgraph PROCESS["处理流程"]
            direction TB
            STEP1["1️⃣ 保存文件到<br/>/opt/xingrin/wordlists/"]
            STEP2["2️⃣ 计算 SHA256 哈希值"]
            STEP3["3️⃣ 统计文件大小和行数"]
            STEP4["4️⃣ 创建/更新数据库记录"]
            
            STEP1 --> STEP2
            STEP2 --> STEP3
            STEP3 --> STEP4
        end
        
        STEP4 --> DB
        STEP1 --> FS
        
        subgraph DB["💾 PostgreSQL 数据库"]
            DBRECORD["INSERT INTO wordlist<br/>name: 'subdomains'<br/>file_path: '/opt/xingrin/wordlists/subdomains.txt'<br/>file_size: 1024000<br/>line_count: 50000<br/>file_hash: 'sha256...'"]
        end
        
        subgraph FS["📁 文件系统"]
            FILES["/opt/xingrin/wordlists/<br/>├── common.txt<br/>├── subdomains.txt<br/>└── directories.txt"]
        end
    end
    
    style SERVER fill:#e6f3ff
    style UI fill:#fff4e6
    style API fill:#f0f0f0
    style SERVICE fill:#d4edda
    style PROCESS fill:#ffe6f0
    style DB fill:#cce5ff
    style FS fill:#e2e3e5
```

## 四、Worker 端获取流程

Worker 执行扫描任务时，通过 `ensure_wordlist_local()` 获取字典：

1. 根据字典名称查询数据库，获取 `file_path` 和 `file_hash`
2. 检查本地是否存在字典文件
   - 存在且 hash 匹配：直接使用
   - 存在但 hash 不匹配：重新下载
   - 不存在：从 Server API 下载
3. 下载地址：`GET /api/wordlists/download/?wordlist=<name>`
4. 返回本地字典文件路径

```mermaid
flowchart TB
    subgraph WORKER["🔧 Worker 容器"]
        direction TB
        
        START["🎯 扫描任务<br/>需要字典"]
        
        START --> ENSURE
        
        ENSURE["ensure_wordlist_local()<br/>参数: wordlist_name"]
        
        ENSURE --> QUERY
        
        QUERY["📊 查询 PostgreSQL<br/>获取 file_path, file_hash"]
        
        QUERY --> CHECK
        
        CHECK{"🔍 检查本地文件<br/>/opt/xingrin/wordlists/"}
        
        CHECK -->|不存在| DOWNLOAD
        CHECK -->|存在| HASH
        
        HASH["🔐 计算本地文件 SHA256<br/>与数据库 hash 比较"]
        
        HASH -->|一致| USE
        HASH -->|不一致| DOWNLOAD
        
        DOWNLOAD["📥 从 Server API 下载<br/>GET /api/wordlists/download/?wordlist=name"]
        
        DOWNLOAD --> SERVER
        
        SERVER["🌐 HTTP Request"]
        
        SERVER -.请求.-> API["Server (Django)<br/>返回文件内容"]
        API -.响应.-> SERVER
        
        SERVER --> SAVE
        
        SAVE["💾 保存到本地<br/>/opt/xingrin/wordlists/filename"]
        
        SAVE --> RETURN
        
        USE["✅ 直接使用"] --> RETURN
        
        RETURN["📂 返回本地字典文件路径<br/>/opt/xingrin/wordlists/subdomains.txt"]
        
        RETURN --> EXEC
        
        EXEC["🚀 执行扫描工具<br/>puredns bruteforce -w /opt/xingrin/wordlists/xxx.txt"]
    end
    
    style WORKER fill:#e6f3ff
    style START fill:#fff4e6
    style CHECK fill:#ffe6f0
    style HASH fill:#ffe6f0
    style USE fill:#d4edda
    style DOWNLOAD fill:#f8d7da
    style RETURN fill:#d4edda
    style EXEC fill:#cce5ff
```

## 五、Hash 校验机制

- 上传时计算 SHA256 并存入数据库
- Worker 使用前校验本地文件 hash
- 不匹配时自动重新下载
- 确保所有节点使用相同内容的字典

## 六、本地 Worker vs 远程 Worker

本地 Worker 和远程 Worker 获取字典的方式相同：

1. 从数据库查询字典元数据（file_hash）
2. 检查本地缓存是否存在且 hash 匹配
3. 不匹配则通过 HTTP API 下载

**注意**：Worker 容器只挂载了 `results` 和 `logs` 目录，没有挂载 `wordlists` 目录，所以字典文件需要通过 API 下载。

```mermaid
sequenceDiagram
    participant W as Worker (本地/远程)
    participant DB as PostgreSQL
    participant S as Server API
    participant FS as 本地缓存
    
    W->>DB: 1️⃣ 查询数据库获取 file_hash
    DB-->>W: 返回 file_hash
    
    W->>FS: 2️⃣ 检查本地缓存
    
    alt 存在且 hash 匹配
        FS-->>W: ✅ 直接使用
    else 不存在或不匹配
        W->>S: 3️⃣ GET /api/wordlists/download/
        S-->>W: 4️⃣ 返回文件内容
        W->>FS: 5️⃣ 保存到本地缓存<br/>/opt/xingrin/wordlists/
        FS-->>W: ✅ 使用缓存文件
    end
    
    Note over W,FS: 本地 Worker 优势：<br/>• 网络延迟更低（容器内网络）<br/>• 缓存可复用（同一宿主机多次任务）
```

### 本地 Worker 的优势

虽然获取方式相同，但本地 Worker 有以下优势：
- 网络延迟更低（容器内网络）
- 下载后的缓存可复用（同一宿主机上的多次任务）

## 七、配置项

在 `docker/.env` 或环境变量中配置：

```bash
# 字典文件存储目录
WORDLISTS_PATH=/opt/xingrin/wordlists

# Server 地址（Worker 用于下载文件）
PUBLIC_HOST=your-server-ip  # 远程 Worker 会通过 https://{PUBLIC_HOST}/api 访问
SERVER_PORT=8888  # 后端容器内部端口，仅 Docker 内网监听
```

## 八、常见问题

### Q: 字典文件更新后 Worker 没有使用新版本？

A: 更新字典内容后会重新计算 hash，Worker 下次使用时会检测到 hash 不匹配并重新下载。

### Q: 远程 Worker 下载文件失败？

A: 检查：
1. `PUBLIC_HOST` 是否配置为 Server 的外网 IP 或域名
2. Nginx 8083 (HTTPS) 是否可达（远程 Worker 通过 nginx 访问后端）
3. Worker 到 Server 的网络是否通畅

### Q: 如何批量导入字典？

A: 目前只支持通过前端逐个上传，后续可能支持批量导入功能。
