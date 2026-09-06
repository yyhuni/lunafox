# Shared Visualization Rules

## Log Surfaces

日志界面按数据语义使用两种组合模式，不要创建同时接管 raw、structured、数据请求和领域状态的万能 viewer。

- `RawLogViewer` 用于扫描进度、命令输出和其他可能包含 ANSI 的原始文本。它拥有内容转义、纯文本级别着色、搜索高亮能力和靠近底部时的自动跟随，但调用方不应因此自动增加工具栏或 footer；需要正文右上角操作时通过 `topRightAction` 传入。
- `StructuredLogViewer` 只负责结构化日志行的解析与展示。行格式统一为 `[timestamp] [LEVEL] message caller key=value`；小窗口保持连续 `<pre>` 文本流，大窗口通过调用方传入的共享 viewport ref 只挂载可见行与有限 overscan，并按自然换行后的实际高度动态测量。语义颜色不得引入 Badge、固定宽度填充或改变复制文本顺序。
- `LiveLogSurface` 为 Agent 和 Server 结构化实时日志提供共享 terminal body、滚动 viewport、焦点样式、跳到最新按钮和 footer shell。需要终端正文右上角操作时通过 `topRightAction` 传入共享 action；数据 hook、过滤状态、来源文案、错误映射和 footer 内容仍由业务调用方拥有。
- `TerminalLogToolbar` 拥有 Agent 与 Server 共享的搜索、级别筛选以及可选行数窗口 Select。行数窗口的 options/value/change handler 必须通过该组合边界传入，业务入口不得各自实现 Select；loading 时共享 Select 必须禁用并保留几何。
- `TerminalLogCopyAllButton` 统一拥有日志正文右上角的“复制全部”图标、所有静态/hover/focus/Tooltip-open 状态均无边框无背景的紧凑外观、Tooltip、禁用状态和复制反馈；日志调用方只传入当前已加载的完整文本及本地化文案。

结构化日志的 latest-N 内存窗口与 DOM 挂载窗口是两个契约：筛选、计数和复制全部始终基于完整内存窗口，虚拟 viewer 只控制正文挂载量。浏览器原生文本选择不保证跨越未挂载行，完整跨窗口复制必须使用 `TerminalLogCopyAllButton`。共享流 hook 在 follow 页面没有新增唯一日志时必须保留原 `lines` 数组引用；metadata 可以继续更新，但不得因此重新提交正文或触发自动滚屏布局。

`TERMINAL_LOG_BODY_CLASS`、`TERMINAL_LOG_VIEWPORT_CLASS` 和 `TERMINAL_LOG_FOOTER_CLASS` 是终端正文几何的唯一共享事实源。业务模块可以拥有页面或 drawer 外壳，但不要复制等宽字号、正文 padding、滚动、焦点或 jump chrome。

共享边界由 `__tests__/raw-log-viewer.contract.test.ts`、`__tests__/raw-log-viewer.test.tsx`、`__tests__/structured-log-viewer.test.tsx` 和 `__tests__/terminal-log-surface.contract.test.ts` 保护。
