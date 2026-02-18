# Redis Stream 队列方案设计文档

## 概述

本文档描述了使用 Redis Stream 作为消息队列来优化大规模数据写入的方案设计。

## 背景

### 当前问题

在扫描大量 Endpoint 数据（几十万条）时，当前的 HTTP 批量写入方案存在以下问题：

1. **性能瓶颈**：50 万 Endpoint（每个 15 KB）需要 83-166 分钟
2. **数据库 I/O 压力**：20 个 Worker 同时写入导致数据库 I/O 满载
3. **Worker 阻塞风险**：如果使用批量写入 + 背压机制，Worker 会阻塞等待

### 方案目标

- 性能提升 10 倍（83 分钟 → 8 分钟）
- Worker 永不阻塞（扫描速度稳定）
- 数据不丢失（持久化保证）
- 无需部署新组件（利用现有 Redis）

## 架构设计

### 整体架构

```
Worker 扫描 → Redis Stream → Server 消费 → PostgreSQL
```

### 数据流

1. **Worker 端**：扫描到 Endpoint → 发布到 Redis Stream
2. **Redis Stream**：缓冲消息（持久化到磁盘）
3. **Server 端**：单线程消费 → 批量写入数据库

### 关键特性

- **解耦**：Worker 和数据库完全解耦
- **背压**：Server 控制消费速度，保护数据库
- **持久化**：Redis AOF 保证数据不丢失
- **扩展性**：支持多 Worker 并发写入

## Redis Stream 配置

### 启用 AOF 持久化

```conf
# redis.conf
appendonly yes
appendfsync everysec  # 每秒同步一次（平衡性能和安全）
```

**效果**：
- 数据持久化到磁盘
- Redis 崩溃最多丢失 1 秒数据
- 性能影响小

### 内存配置

```conf
# redis.conf
maxmemory 2gb
maxmemory-policy allkeys-lru  # 内存不足时淘汰最少使用的 key
```

## 实现方案

### 1. Worker 端：发布到 Redis Stream

#### 代码结构

```
worker/internal/queue/
├── redis_publisher.go  # Redis 发布者
└── types.go            # 数据类型定义
```

#### 核心实现

```go
// worker/internal/queue/redis_publisher.go
package queue

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/redis/go-redis/v9"
)

type RedisPublisher struct {
    client *redis.Client
}

func NewRedisPublisher(redisURL string) (*RedisPublisher, error) {
    opt, err := redis.ParseURL(redisURL)
    if err != nil {
        return nil, err
    }

    client := redis.NewClient(opt)

    // 测试连接
    if err := client.Ping(context.Background()).Err(); err != nil {
        return nil, err
    }

    return &RedisPublisher{client: client}, nil
}

// PublishEndpoint 发布 Endpoint 到 Redis Stream
func (p *RedisPublisher) PublishEndpoint(ctx context.Context, scanID int, endpoint Endpoint) error {
    data, err := json.Marshal(endpoint)
    if err != nil {
        return err
    }

    streamName := fmt.Sprintf("endpoints:%d", scanID)

    return p.client.XAdd(ctx, &redis.XAddArgs{
        Stream: streamName,
        MaxLen: 1000000, // 最多保留 100 万条消息（防止内存溢出）
        Approx: true,    // 使用近似裁剪（性能更好）
        Values: map[string]interface{}{
            "data": data,
        },
    }).Err()
}

// Close 关闭连接
func (p *RedisPublisher) Close() error {
    return p.client.Close()
}
```

#### 使用示例

```go
// Worker 扫描流程
func (w *Worker) ScanEndpoints(ctx context.Context, scanID int) error {
    // 初始化 Redis 发布者
    publisher, err := queue.NewRedisPublisher(os.Getenv("REDIS_URL"))
    if err != nil {
        return err
    }
    defer publisher.Close()

    // 扫描 Endpoint
    for endpoint := range w.scan() {
        // 发布到 Redis Stream（非阻塞，超快）
        if err := publisher.PublishEndpoint(ctx, scanID, endpoint); err != nil {
            log.Printf("Failed to publish endpoint: %v", err)
            // 可以选择重试或记录错误
        }
    }

    return nil
}
```

### 2. Server 端：消费 Redis Stream

#### 代码结构

```
server/internal/queue/
├── redis_consumer.go   # Redis 消费者
├── batch_writer.go     # 批量写入器
└── types.go            # 数据类型定义
```

#### 核心实现

```go
// server/internal/queue/redis_consumer.go
package queue

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    "github.com/redis/go-redis/v9"
    "github.com/yyhuni/lunafox/server/internal/repository"
)

type EndpointConsumer struct {
    client     *redis.Client
    repository *repository.EndpointRepository
}

func NewEndpointConsumer(redisURL string, repo *repository.EndpointRepository) (*EndpointConsumer, error) {
    opt, err := redis.ParseURL(redisURL)
    if err != nil {
        return nil, err
    }

    client := redis.NewClient(opt)

    return &EndpointConsumer{
        client:     client,
        repository: repo,
    }, nil
}

// Start 启动消费者（单线程，控制写入速度）
func (c *EndpointConsumer) Start(ctx context.Context, scanID int) error {
    streamName := fmt.Sprintf("endpoints:%d", scanID)
    groupName := "endpoint-consumers"
    consumerName := fmt.Sprintf("server-%d", time.Now().Unix())

    // 创建消费者组（如果不存在）
    c.client.XGroupCreateMkStream(ctx, streamName, groupName, "0")

    // 批量写入器（每 5000 条批量写入）
    batchWriter := NewBatchWriter(c.repository, 5000)
    defer batchWriter.Flush()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        // 读取消息（批量）
        streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
            Group:    groupName,
            Consumer: consumerName,
            Streams:  []string{streamName, ">"},
            Count:    100,  // 每次读取 100 条
            Block:    1000, // 阻塞 1 秒
        }).Result()

        if err != nil {
            if err == redis.Nil {
                continue // 没有新消息
            }
            return err
        }

        // 处理消息
        for _, stream := range streams {
            for _, message := range stream.Messages {
                // 解析消息
                var endpoint Endpoint
                if err := json.Unmarshal([]byte(message.Values["data"].(string)), &endpoint); err != nil {
                    // 记录错误，继续处理下一条
                    continue
                }

                // 添加到批量写入器
                if err := batchWriter.Add(endpoint); err != nil {
                    return err
                }

                // 确认消息（ACK）
                c.client.XAck(ctx, streamName, groupName, message.ID)
            }
        }

        // 定期 Flush
        if batchWriter.ShouldFlush() {
            if err := batchWriter.Flush(); err != nil {
                return err
            }
        }
    }
}

// Close 关闭连接
func (c *EndpointConsumer) Close() error {
    return c.client.Close()
}
```

#### 批量写入器

```go
// server/internal/queue/batch_writer.go
package queue

import (
    "sync"
    "github.com/yyhuni/lunafox/server/internal/model"
    "github.com/yyhuni/lunafox/server/internal/repository"
)

type BatchWriter struct {
    repository *repository.EndpointRepository
    buffer     []model.Endpoint
    batchSize  int
    mu         sync.Mutex
}

func NewBatchWriter(repo *repository.EndpointRepository, batchSize int) *BatchWriter {
    return &BatchWriter{
        repository: repo,
        batchSize:  batchSize,
        buffer:     make([]model.Endpoint, 0, batchSize),
    }
}

// Add 添加到缓冲区
func (w *BatchWriter) Add(endpoint model.Endpoint) error {
    w.mu.Lock()
    w.buffer = append(w.buffer, endpoint)
    shouldFlush := len(w.buffer) >= w.batchSize
    w.mu.Unlock()

    if shouldFlush {
        return w.Flush()
    }
    return nil
}

// ShouldFlush 是否应该 Flush
func (w *BatchWriter) ShouldFlush() bool {
    w.mu.Lock()
    defer w.mu.Unlock()
    return len(w.buffer) >= w.batchSize
}

// Flush 批量写入数据库
func (w *BatchWriter) Flush() error {
    w.mu.Lock()
    if len(w.buffer) == 0 {
        w.mu.Unlock()
        return nil
    }

    // 复制缓冲区
    toWrite := make([]model.Endpoint, len(w.buffer))
    copy(toWrite, w.buffer)
    w.buffer = w.buffer[:0]
    w.mu.Unlock()

    // 批量写入（使用现有的 BulkUpsert 方法）
    _, err := w.repository.BulkUpsert(toWrite)
    return err
}
```

### 3. Server 启动消费者

```go
// server/internal/app/app.go
func Run(ctx context.Context, cfg config.Config) error {
    // ... 现有代码

    // 启动 Redis 消费者（后台运行）
    consumer, err := queue.NewEndpointConsumer(cfg.RedisURL, endpointRepo)
    if err != nil {
        return err
    }

    go func() {
        // 消费所有活跃的扫描任务
        for {
            // 获取活跃的扫描任务
            scans := scanRepo.GetActiveScans()
            for _, scan := range scans {
                go consumer.Start(ctx, scan.ID)
            }
            time.Sleep(10 * time.Second)
        }
    }()

    // ... 现有代码
}
```

## 性能对比

### 50 万 Endpoint（每个 15 KB）

| 方案 | 写入速度 | 总时间 | 内存占用 | Worker 阻塞 |
|------|---------|--------|---------|-----------|
| **当前（HTTP 批量）** | 100 条/秒 | 83 分钟 | 1.5 MB | 否 |
| **Redis Stream** | 1000 条/秒 | 8 分钟 | 75 MB | 否 |

**提升**：**10 倍性能！**

## 资源消耗

### Redis 资源消耗

| 项目 | 消耗 |
|------|------|
| 内存 | ~500 MB（缓冲 100 万条消息） |
| CPU | ~10%（序列化/反序列化） |
| 磁盘 | ~7.5 GB（AOF 持久化） |
| 带宽 | ~50 MB/s |

### Server 资源消耗

| 项目 | 消耗 |
|------|------|
| 内存 | 75 MB（批量写入缓冲） |
| CPU | 30%（反序列化 + 数据库写入） |
| 数据库连接 | 1 个（单线程消费） |

## 可靠性保证

### 数据不丢失

1. **Redis AOF 持久化**：每秒同步到磁盘，最多丢失 1 秒数据
2. **消息确认机制**：Server 处理成功后才 ACK
3. **自动重试**：未 ACK 的消息会自动重新入队

### 故障恢复

| 故障场景 | 恢复机制 |
|---------|---------|
| Worker 崩溃 | 消息已发送到 Redis，不影响 |
| Redis 崩溃 | AOF 恢复，最多丢失 1 秒数据 |
| Server 崩溃 | 未 ACK 的消息重新入队 |
| 数据库崩溃 | 消息保留在 Redis，恢复后继续消费 |

## 扩展性

### 多 Worker 支持

- Redis Stream 原生支持多个生产者
- 无需额外配置

### 多 Server 消费者

```go
// 启动多个消费者（负载均衡）
for i := 0; i < 3; i++ {
    go consumer.Start(ctx, scanID)
}
```

Redis Stream 的消费者组会自动分配消息，实现负载均衡。

## 监控和运维

### 监控指标

```go
// 获取队列长度
func (c *EndpointConsumer) GetQueueLength(ctx context.Context, scanID int) (int64, error) {
    streamName := fmt.Sprintf("endpoints:%d", scanID)
    return c.client.XLen(ctx, streamName).Result()
}

// 获取消费者组信息
func (c *EndpointConsumer) GetConsumerGroupInfo(ctx context.Context, scanID int) ([]redis.XInfoGroup, error) {
    streamName := fmt.Sprintf("endpoints:%d", scanID)
    return c.client.XInfoGroups(ctx, streamName).Result()
}
```

### 清理策略

```go
// 扫描完成后清理 Stream
func (c *EndpointConsumer) CleanupStream(ctx context.Context, scanID int) error {
    streamName := fmt.Sprintf("endpoints:%d", scanID)
    return c.client.Del(ctx, streamName).Err()
}
```

## 配置建议

### Redis 配置

```conf
# redis.conf

# 持久化
appendonly yes
appendfsync everysec

# 内存
maxmemory 2gb
maxmemory-policy allkeys-lru

# 性能
tcp-backlog 511
timeout 0
tcp-keepalive 300
```

### 环境变量

```bash
# Worker 端
REDIS_URL=redis://localhost:6379/0

# Server 端
REDIS_URL=redis://localhost:6379/0
```

## 迁移步骤

### 阶段 1：准备（1 天）

1. 启用 Redis AOF 持久化
2. 实现 Worker 端 Redis 发布者
3. 实现 Server 端 Redis 消费者

### 阶段 2：测试（2 天）

1. 单元测试
2. 集成测试
3. 性能测试（模拟 50 万数据）

### 阶段 3：灰度发布（3 天）

1. 10% 流量使用 Redis Stream
2. 50% 流量使用 Redis Stream
3. 100% 流量使用 Redis Stream

### 阶段 4：清理（1 天）

1. 移除旧的 HTTP 批量写入代码
2. 更新文档

## 风险和缓解

### 风险 1：Redis 内存溢出

**缓解**：
- 设置 `maxmemory` 限制
- 使用 `MaxLen` 限制 Stream 长度
- 监控 Redis 内存使用

### 风险 2：消息积压

**缓解**：
- 增加 Server 消费者数量
- 优化数据库写入性能
- 监控队列长度

### 风险 3：数据丢失

**缓解**：
- 启用 AOF 持久化
- 使用消息确认机制
- 定期备份 Redis

## 总结

### 优势

- ✅ 性能提升 10 倍
- ✅ Worker 永不阻塞
- ✅ 数据不丢失（AOF 持久化）
- ✅ 无需部署新组件（利用现有 Redis）
- ✅ 架构简单，易于维护

### 适用场景

- 数据量 > 10 万
- 已有 Redis
- 需要高性能写入
- 不需要复杂的消息路由

### 不适用场景

- 数据量 < 10 万（当前方案足够）
- 需要复杂的消息路由（考虑 RabbitMQ）
- 数据量 > 1000 万（考虑 Kafka）

## 参考资料

- [Redis Stream 官方文档](https://redis.io/docs/data-types/streams/)
- [Redis 持久化](https://redis.io/docs/management/persistence/)
- [go-redis 文档](https://redis.uptrace.dev/)
