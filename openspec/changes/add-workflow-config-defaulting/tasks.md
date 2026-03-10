## 1. Implementation
- [ ] 1.1 为 server 侧 canonical config 归一化补充行为测试
- [ ] 1.2 在 server 引入基于 workflow contract / manifest 的显式规范化入口
- [ ] 1.3 调整 scan create 链路，持久化 canonical workflow YAML
- [ ] 1.4 为 worker 强类型解码增加短配置默认值补齐测试
- [ ] 1.5 在 worker 强类型解码中实现 contract / manifest 驱动默认值补齐
- [ ] 1.6 调整 generator，明确 schema `default` 仅为 contract 语义镜像
- [ ] 1.7 重生成 schema / manifest / profile / docs 产物并补齐一致性测试
- [ ] 1.8 运行 server / worker 相关测试与 OpenSpec 校验
