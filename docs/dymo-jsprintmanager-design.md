# JSPrintManager 完全兼容与 DYMO 标签打印重设计方案

## 1. 文档目的

本文档给出 OpenPrintHub 面向 **Windows + DYMO LabelWriter 450** 的完整重设计方案，并将 **JSPrintManager (JSPM)** 定义为 OpenPrintHub 的 **公开集成契约基准**。新的目标不是“做一个大致像 JSPM 的替代品”，而是让 OpenPrintHub 的前端调用模型、对象模型、WebSocket 生命周期与常用查询接口尽量与 JSPM 保持一致。

这份方案重点解决三个核心问题：

1. **公开契约一致性**：浏览器端的 `JSPM.JSPrintManager`、`ClientPrintJob`、`InstalledPrinter`、`PrintFilePDF`、`sendToClient()` 等使用方式尽量保持一致。
2. **标签打印准确性**：DYMO 标签打印不再走“通用 PDF 打印”思路，而是走“标签尺寸优先、像素精确可控”的专用链路，解决宽高、缩放、字体大小、边距失真问题。
3. **内核可扩展性**：即使公开契约向 JSPM 对齐，内部仍统一到 OpenPrintHub 的打印核心，以便分别处理 DYMO、Zebra、Brother、A4 办公打印等不同路径。

---

## 2. 背景与现状问题

当前 OpenPrintHub 已经可以在 Windows 上把 PDF 任务发送到打印机，但如果目标是 **“让前端像 JSPM 一样调用，同时让 DYMO 450 打印尺寸准确”**，现状存在几类结构性问题：

1. **当前公开接口不是 JSPM 模型**
   - 现在项目的主集成面是自定义 REST API：`POST /v1/print`、`GET /v1/printers` 等。
   - 而 JSPM 的典型使用方式是浏览器端通过 `JSPM.JSPrintManager.start()` 建立本地连接，再创建 `ClientPrintJob`、`InstalledPrinter`、`PrintFilePDF` 等对象并调用 `sendToClient()`。
   - 如果 OpenPrintHub 继续以现有 REST 契约为中心，前端迁移成本仍然很高。

2. **PDF 能触发打印，不代表物理尺寸正确**
   - PDF 查看器、Windows 打印子系统、打印驱动都可能参与缩放。
   - 只要页面尺寸、纸张名称、驱动默认纸型、自动适应页面中的任一项不一致，最终打印出的标签宽高就会偏。

3. **DYMO 标签机不是 A4 办公打印机**
   - A4 打印常见策略是“把文档交给 PDF 查看器/驱动处理”。
   - DYMO 更关心的是“目标标签的物理尺寸是否精确”、“边距是否可控”、“字体是否按标签像素栅格落点”。

4. **现有 Windows PDF 路径控制粒度不足**
   - 当前实现本质上是把 PDF 交给 ShellExecute 或 SumatraPDF。
   - 这条链路无法稳定控制：
     - 目标纸张名称 / Label stock
     - 缩放策略（exact / fit / shrink）
     - 实际打印区域
     - PDF 页尺寸与标签纸尺寸不一致时的处理方式

5. **当前项目还没有历史包袱**
   - 既然库暂时还没有人用，就不必为了兼容现在的 `/v1/*` 设计而束手束脚。
   - 这意味着可以直接把 **JSPM 公共接口模型** 作为新的主契约，而把当前 REST API 降级为内部适配层或管理接口。

结论：

1. **对外公开契约应从“REST-first”切换到“JSPM-compatible public contract first”**
2. **对 DYMO 450，直接把 PDF 当成最终打印载体不是最佳实践；PDF 只能作为输入格式之一，不能作为 DYMO 标签打印的默认执行格式**

---

## 3. 设计目标

### 3.1 主要目标

1. **JSPM 公共接口完全对齐**
   - 支持 `JSPM.JSPrintManager.start()`、`JSPrintManager.WS.onOpen/onClose/onStatusChanged`、`websocket_status`。
   - 支持 `ClientPrintJob`、`ClientPrintJobGroup`、`DefaultPrinter`、`InstalledPrinter`、`UserSelectedPrinter`、`ParallelPortPrinter`、`SerialPortPrinter`、`NetworkPrinter`。
   - 支持 `PrintFile`、`PrintFilePDF`、`PrintFileTIF`、`PrintFileTXT` 等常用文件模型，以及 `pageSizing`、`printScale`、`printAutoCenter`、`paperName`、`trayName`、`mediaType` 等关键属性。

2. **前端最小迁移**
   - 现有 JSPM 风格业务代码应尽量做到“只换脚本来源/本地服务端”即可运行。
   - 新前端也直接按 JSPM 的对象模型来写，而不是再发明一套 OpenPrintHub 专用调用协议。

3. **对 DYMO 标签做到物理尺寸精确**
   - 以标签纸物理尺寸为一等公民，而不是以 PDF 页尺寸为中心。
   - 默认禁止隐式自动缩放。

4. **统一公开契约、分家族执行**
   - A4/激光/喷墨：继续允许通用 PDF 路径。
   - DYMO/Brother 标签机：走标签专用渲染链路。
   - Zebra/TSC：继续优先使用 ZPL/TSPL/raw。

5. **失败要显式，不允许“默默打错”**
   - 纸张不匹配、找不到对应 label stock、缩放策略不明确时，应返回错误或警告，不自动降级到不确定行为。

### 3.2 非目标

1. 不保留当前 `/v1/*` 作为主集成入口。
2. 不复刻 JSPM 的商业授权、更新分发或商业化包装体系。
3. 不把所有打印机都强行统一为一条打印链路。
4. 不让 DYMO 标签继续默认走 ShellExecute PDF 打印。

---

## 4. 总体设计原则

### 4.1 “公开契约兼容，打印内核归一”

对外以 **JSPM 公共接口** 作为第一契约，对内统一转换成 OpenPrintHub 的规范化打印票据（Print Ticket）。

### 4.2 “先兼容 WebSocket 会话模型，再兼容管理型 REST”

对浏览器集成来说，JSPM 的关键不只是方法名，还包括：

- `JSPrintManager.start(secure?, host?, port?)`
- `JSPrintManager.WS`
- `WS.onOpen / onClose / onStatusChanged`
- `JSPrintManager.websocket_status`
- `ClientPrintJob.sendToClient()`

因此必须先兼容这套会话模型，再考虑保留多少现有 REST 接口。

### 4.3 “标签尺寸优先于文档格式”

对于 DYMO，系统首先关心：

- 打印机是什么
- 当前纸张/label stock 是什么
- 目标标签宽高是多少
- 是否允许缩放

而不是先问“这是 PDF、PNG 还是 JPG”。

### 4.4 “标签打印先渲染，再输出”

对 DYMO 这样的标签机：

- **首选**：把标签内容渲染成目标尺寸位图，再按精确尺寸打印。
- **次选**：如果输入是 PDF，先把 PDF 栅格化成目标标签位图，再打印。
- **不推荐默认**：直接把 PDF 原样交给查看器/驱动。

---

## 5. 目标架构

```text
Browser / Existing JSPM-style App
        |
        |  JSPM-compatible public JS surface
        v
OpenPrintHub Local Client Runtime
        |
        |  JSPM-compatible WebSocket endpoint
        v
JSPM Message / Object Adapter
        |
        |  normalize
        v
Canonical Print Ticket
        |
        +--> Printer Capability Registry
        |
        +--> Render Strategy Selector
                 |
                 +--> Generic Document Pipeline
                 +--> DYMO Label Pipeline
                 +--> Raw Command Pipeline
        |
        v
Platform Print Executor
```

系统拆成 6 层：

1. **JSPM 兼容公开层**
2. **本地 WebSocket 运行时**
3. **JSPM 消息/对象适配器**
4. **规范化打印票据（Canonical Print Ticket）**
5. **打印机能力与介质注册表（Capability Registry）**
6. **平台执行器（Platform Executor）**

说明：

- 浏览器应用的主入口不再是 `POST /v1/print`，而是 JSPM 风格的 JS + WebSocket 会话。
- 当前的 `/v1/*` 接口可以保留给管理后台、诊断工具和内部适配器使用，但不再是推荐给业务前端的主集成面。

---

## 6. JSPM 兼容层设计

### 6.1 兼容目标级别

本方案把兼容目标拆成 4 层，目标是全部覆盖：

1. **命名空间与方法签名兼容**
   - 例如 `JSPM.JSPrintManager.start()`、`getPrintersInfo()`、`getPapers()`、`getPaperInfo()`、`getTrays()`、`getInstances()`
2. **对象模型兼容**
   - 例如 `ClientPrintJob`、`ClientPrintJobGroup`、`InstalledPrinter`、`PrintFilePDF`
3. **传输与异步行为兼容**
   - 例如 `WS.onOpen`、`WS.onClose`、`WS.onStatusChanged`、`sendToClient()`、job trace / status 回调
4. **行为语义兼容**
   - 例如 `paperName`、`trayName`、`mediaType`、`pageSizing`、`printScale` 对最终打印行为的影响

### 6.2 首批必须支持的 JSPM 公共面

| 类别 | 首批必须支持的公共项 |
|---|---|
| 会话控制 | `JSPM.JSPrintManager.start(secure?, host?, port?)`, `stop()`, `auto_reconnect`, `WS`, `websocket_status` |
| 打印机选择 | `DefaultPrinter`, `InstalledPrinter(printerName, printToDefaultIfNotFound?, trayName?, paperName?, duplex?, autoDetectRawModeDataType?, driverModel?, mediaType?)`, `UserSelectedPrinter`, `ParallelPortPrinter`, `SerialPortPrinter`, `NetworkPrinter` |
| 打印任务 | `ClientPrintJob`, `ClientPrintJobGroup`, `sendToClient()`, `onUpdated`, `onFinished`, `onError`, `printerCommands`, `binaryPrinterCommands`, `files` |
| 文件模型 | `PrintFile`, `PrintFilePDF`, `PrintFileTIF`, `PrintFileTXT`, `FileSourceType.BLOB`, `FileSourceType.ExternalURL` |
| PDF/TIF 关键属性 | `pageSizing`, `printScale`, `printAutoCenter`, `printAutoRotate`, `printRotation`, `printRange`, `manualDuplex` |
| 打印机查询 | `getPrinters`, `getPrintersInfo`, `getDefaultPaperName`, `getDefaultTrayName`, `getPapers`, `getPaperInfo`, `getTrays`, `getMediaTypes` |
| 环境/实例查询 | `getClientAppInfo`, `getInstances`, `getUser`, `refreshPrinters`, `onPrinterCreated`, `onPrinterUpdated`, `onPrinterDeleted` |

### 6.3 传输与运行时兼容方式

OpenPrintHub 本地进程需要直接提供 **JSPM 兼容的 WebSocket 会话入口**，而不是只暴露一个自定义 REST API。兼容要求如下：

1. `JSPrintManager.start(secure?, host?, port?)` 的参数语义保持一致
2. 默认 host/port 行为遵循 JSPM 的单用户会话约定
3. `getInstances()` 提供多用户会话发现能力，返回 `user + port` 之类的信息
4. `JSPrintManager.WS.onOpen / onClose / onStatusChanged` 能反映本地运行时状态
5. `ClientPrintJob.sendToClient()` 触发异步提交，并以 JSPM 风格的 job trace / 状态回调返回过程事件

### 6.4 兼容边界

需要明确的是，“完全一致”是指 **公开契约与前端使用方式**，不是要求内部实现逐字节复制 JSPM。OpenPrintHub 内部允许做不同的实现选择，只要对外行为一致：

1. **DYMO 上的 `PrintFilePDF` 可在内部改走 PDF 栅格化标签链路**
2. **A4 打印仍可走通用文档链路**
3. **当前 `/v1/*` REST API 不再定义公开契约，只作为内部或管理用途**

### 6.5 兼容设计决策

既然当前库还没有用户，这里做出一个明确决策：

1. **新设计以 JSPM 公共契约为第一优先级**
2. **现有 REST 接口允许大改或降级**
3. **未来的前端示例、SDK 与文档全部围绕 JSPM 风格写法组织**

---

## 7. 规范化打印票据（Canonical Print Ticket）

兼容层提交后，不直接进入平台打印，而是先归一化成统一票据：

```json
{
  "printer_selector": {
    "mode": "installed",
    "name": "DYMO LabelWriter 450"
  },
  "document": {
    "source_type": "pdf",
    "payload_ref": "job-blob://123",
    "name": "shipping-label.pdf"
  },
  "intent": {
    "class": "label",
    "printer_family": "dymo",
    "media": {
      "paper_name": "Label 50x25mm",
      "width_mm": 50.0,
      "height_mm": 25.0
    },
    "layout": {
      "scale_policy": "exact",
      "rotation": 0,
      "margins_mm": 0
    }
  },
  "copies": 1
}
```

### 7.1 建议新增字段

当前 `PrintSettings` 对 A4 打印足够，但对标签不够。建议扩展为：

| 字段 | 说明 |
|---|---|
| `paper_name` | 驱动里的纸张名 / label stock 名称 |
| `width_mm` / `height_mm` | 物理尺寸，优先用于标签 |
| `printer_family` | `auto` / `dymo` / `zebra` / `brother` / `generic` |
| `document_class` | `office-document` / `label` / `receipt` / `raw` |
| `scale_policy` | `exact` / `fit` / `fill` / `shrink-to-fit` |
| `margins_mm` | 显式边距，标签默认 0 |
| `paper_source` / `tray_name` | 与 JSPM `trayName` 对齐 |
| `force_rasterize_pdf` | 是否强制 PDF 栅格化 |
| `font_policy` | 字体回退与字号策略 |

### 7.2 默认策略

对于 `printer_family=dymo` 或自动识别为 DYMO 的打印机，默认值应为：

| 项目 | 默认值 |
|---|---|
| `document_class` | `label` |
| `scale_policy` | `exact` |
| `fit_to_page` | `false` |
| `margins_mm` | `0` |
| `force_rasterize_pdf` | `true` |
| `paper_name` | 必填；若缺失，则尝试从当前默认纸张推断，否则报错 |

---

## 8. 打印机能力与介质注册表

### 8.1 为什么必须有能力注册表

标签打印要稳定，不能只知道“打印机名字”。还必须知道：

- 是什么厂商 / 家族
- 支持哪些纸张名称
- 当前默认纸张是什么
- 可打印区域有多大
- 分辨率 / 方向 / 自定义纸张是否可用

### 8.2 能力来源

能力信息分为两类：

1. **动态发现**
   - Windows 已安装打印机信息
   - 支持纸张列表
   - 支持 tray/bin
   - 默认纸张 / 默认 tray
   - 驱动返回的 DPI / 可打印区域

2. **静态内置 profile**
   - 已知打印机家族的默认行为
   - DYMO 450 常见标签 stock 的别名映射
   - 纸张名称匹配规则

### 8.3 建议新增内部/管理能力接口

在现有 `/v1/printers` 基础上补充：

- `GET /v1/printers/:id/capabilities`
- `GET /v1/printers/:id/papers`
- `GET /v1/printers/:id/papers/:paperName`
- `GET /v1/printers/:id/trays`

这些接口主要作为 **OpenPrintHub 内部能力源** 与管理后台接口存在，再由 JSPM 兼容层对外实现 `getPapers()`、`getPaperInfo()`、`getTrays()`、`getMediaTypes()` 等公开方法。

---

## 9. Windows + DYMO 450 专用执行链路

这是本方案的核心。

### 9.1 链路总原则

**DYMO 不默认走“通用 PDF 打印器”，而是走“标签介质感知 + 精确渲染 + 精确落纸”链路。**

### 9.2 请求进入后的处理顺序

1. 识别打印机是否为 DYMO 家族
2. 解析或确认 `paper_name` / `width_mm` / `height_mm`
3. 校验当前驱动是否支持该纸张
4. 选择渲染策略
5. 生成最终位图
6. 用 Windows 标签执行器按精确尺寸提交给打印机

### 9.3 渲染策略选择

### A. 输入本身就是图片（PNG/JPG/BMP）

直接进入标签位图链路，但必须：

1. 先确认目标物理尺寸
2. 按目标尺寸决定最终像素大小
3. 根据 `scale_policy` 决定是否缩放

### B. 输入是结构化标签数据（推荐长期方向）

例如：

- 文本字段
- 条码 / 二维码
- 图片 logo
- 对齐方式

这种情况下，系统直接在目标标签画布上排版，输出质量最好，字体和位置最稳定。

### C. 输入是 PDF（兼容旧系统的关键）

对于 DYMO，**PDF 只作为输入格式，不作为最终打印格式。**

处理方式：

1. 先用 PDF 渲染器把目标页栅格化
2. 输出到标签尺寸对应的目标位图
3. 再由标签执行器打印

默认情况下：

- `force_rasterize_pdf=true`
- 不再把 PDF 原样交给 ShellExecute / SumatraPDF

### 9.4 PDF 栅格化方案

推荐设计：

1. **优先方案：PDFium**
   - 负责把 PDF 页渲染成位图
   - 能稳定按目标 DPI 输出

2. **可选后备：MuPDF / Ghostscript**
   - 仅作为实现阶段备选，不作为默认方案

栅格化时要显式指定：

- 目标页
- 输出 DPI
- 背景色
- 页面裁切区域
- 旋转方向

### 9.5 标签位图尺寸计算

系统必须从物理尺寸推导目标像素尺寸：

```text
width_px  = width_mm  / 25.4 * dpi_x
height_px = height_mm / 25.4 * dpi_y
```

注意：

1. 不同打印机可能横纵 DPI 不同
2. 标签宽高与打印方向要解耦
3. 不能依赖 PDF 文件自带页尺寸作为唯一事实来源

### 9.6 最终打印执行器

Windows 标签执行器负责：

1. 设置目标打印机
2. 匹配 `paper_name`
3. 设置 0 边距或显式边距
4. 关闭“自动适应页面”类行为
5. 将目标位图绘制到正确物理区域

执行器必须满足：

- 不依赖用户界面
- 不弹打印对话框
- 可以准确拿到纸张信息
- 失败时返回明确错误

### 9.7 为什么不能继续用当前 PDF 方案

当前 PDF 方案把文件交给：

- `ShellExecute(print)`
- 或 SumatraPDF

这条路径的问题是：

1. 纸张控制能力弱
2. 缩放行为不可完全预测
3. 查看器与驱动的默认行为差异大
4. DYMO 标签对物理尺寸敏感，容错极低

因此：

**A4 可以继续保留 PDF 原生路径；DYMO 标签必须切到标签专用路径。**

---

## 10. 字体、版式与缩放策略

你现在看到的“字体大小都不对”，本质不是字体本身错了，而是**整个页面被按错误比例缩放了**。解决方式是把字体控制放到渲染阶段，而不是交给 PDF 查看器。

### 10.1 字体策略

推荐引入显式字体策略：

- 指定首选字体族
- 指定 fallback 字体族
- 对中文场景提供稳定 fallback
- 在渲染时完成字号换算与文字测量

例如：

```json
{
  "font_policy": {
    "primary_family": "Arial",
    "fallback_families": ["Microsoft YaHei UI", "Segoe UI", "sans-serif"],
    "embed_mode": "render-time"
  }
}
```

### 10.2 缩放策略

建议定义 4 种显式策略：

| 策略 | 含义 | DYMO 默认 |
|---|---|---|
| `exact` | 严格按目标物理尺寸输出，不自动缩放 | 是 |
| `fit` | 等比缩放到可打印区域内 | 否 |
| `fill` | 等比放大，允许裁切 | 否 |
| `shrink-to-fit` | 仅当内容过大时缩小 | 否 |

DYMO 标签默认必须是 `exact`，否则你看到的宽高与字号问题会反复出现。

---

## 11. 公共接口重设计建议

### 11.1 主集成面改为 JSPM 兼容 JS + WebSocket

新的主集成面不再是 `POST /v1/print`，而是：

1. 浏览器引入 JSPM 兼容脚本
2. 浏览器调用 `JSPM.JSPrintManager.start()`
3. 浏览器通过 `ClientPrintJob` / `InstalledPrinter` / `PrintFilePDF` / `sendToClient()` 与本地 OpenPrintHub 运行时交互

也就是说：

- **业务前端的主接口 = JSPM 风格 JS API + 本地 WebSocket**
- **`/v1/*` = 管理后台、健康检查、诊断、内部适配用途**

### 11.2 目标前端调用方式

目标是让前端可以直接按 JSPM 风格写：

```javascript
JSPM.JSPrintManager.auto_reconnect = true;
await JSPM.JSPrintManager.start();

const cpj = new JSPM.ClientPrintJob();
cpj.clientPrinter = new JSPM.InstalledPrinter(
  "DYMO LabelWriter 450",
  false,
  "Auto",
  "30252 Address",
);

const pdf = new JSPM.PrintFilePDF(fileBlob, JSPM.FileSourceType.BLOB, "label.pdf", 1);
pdf.pageSizing = JSPM.Sizing.None;
pdf.printScale = 100;
pdf.printAutoRotate = false;
pdf.printAutoCenter = false;

cpj.files.push(pdf);
cpj.onUpdated = data => console.log(data);
cpj.onFinished = data => console.log(data);
cpj.sendToClient();
```

对于 DYMO，虽然前端还是这样调用，但内部执行会做两件关键事：

1. 从 `InstalledPrinter.paperName` / `trayName` / `mediaType` 推导目标 label stock
2. 对 `PrintFilePDF` 默认走 **PDF -> 位图 -> exact 标签打印** 链路

### 11.3 OpenPrintHub 内部/管理接口定位

以下接口仍然建议保留，但定位改变：

- `GET /health`
- `GET /v1/info`
- `GET /v1/printers`
- `GET /v1/printers/default`
- `GET /v1/jobs/:id`
- `GET /v1/stats`
- 管理后台所需的其他内部接口

同时：

- `POST /v1/print`、`POST /v1/print/batch` 不再作为推荐给业务前端的主契约
- 它们可以保留成 **legacy adapter**，把旧式 REST 请求翻译成新的规范化打印票据
- 管理后台和测试脚本仍然可以用这套接口做快速验证

### 11.4 JSPM 公开契约到 OPH 内核的映射关系

| JSPM 公开面 | OPH 内部含义 |
|---|---|
| `JSPrintManager.start(secure, host, port)` | 建立到本地 OPH 运行时的 JSPM 兼容 WebSocket 会话 |
| `JSPrintManager.websocket_status` | 映射到本地运行时连接状态 |
| `ClientPrintJob` | 一个规范化打印票据 |
| `ClientPrintJobGroup` | 多个规范化打印票据的组合 |
| `clientPrinter = DefaultPrinter()` | `printer_selector.mode=default` |
| `clientPrinter = InstalledPrinter(name, ..., trayName, paperName, ..., mediaType)` | `printer_selector.mode=installed` + 介质/纸张/来源信息 |
| `PrintFilePDF` | `document.source_type=pdf` |
| `PrintFile` (JPG/PNG/BMP) | `document.source_type=image` |
| `pageSizing`, `printScale`, `printAutoCenter`, `printAutoRotate` | `intent.layout` 的缩放、旋转、定位策略 |
| `printerCommands` / `binaryPrinterCommands` | raw command pipeline |
| `getPapers/getPaperInfo/getTrays/getMediaTypes` | printer capability registry |
| `onUpdated/onFinished/onError` | job status / trace 事件 |

---

## 12. 推荐实现路线

### Phase 1：JSPM 公开契约骨架

目标：

1. 实现 `JSPM.JSPrintManager.start()` / `stop()` / `websocket_status`
2. 实现 `JSPrintManager.WS.onOpen/onClose/onStatusChanged`
3. 实现 `ClientPrintJob`、`DefaultPrinter`、`InstalledPrinter`、`PrintFile`、`PrintFilePDF`
4. 实现 `sendToClient()` 的最小可用路径

完成标志：

- JSPM 风格前端代码能够连接本地 OPH 运行时
- 能列出打印机并提交基本打印任务
- 前端不需要再调用 `/v1/print`

### Phase 2：打印机查询与能力契约对齐

目标：

1. 实现 `getPrintersInfo`、`getPapers`、`getPaperInfo`、`getTrays`、`getMediaTypes`
2. 实现 `getDefaultPaperName`、`getDefaultTrayName`
3. 实现 `refreshPrinters`、`onPrinterCreated`、`onPrinterUpdated`、`onPrinterDeleted`
4. 实现 `getClientAppInfo`、`getInstances`、`getUser`

完成标志：

- JSPM 风格的前端能完整读取打印机能力
- 前端能用 `paperName` / `trayName` 直接指定目标介质
- 多用户/多实例场景具备可发现能力

### Phase 3：Windows DYMO 精确标签执行器

目标：

1. 自动识别 DYMO 打印机家族
2. 建立图片标签的 exact 打印链路
3. 保证 `paperName`、`trayName`、`mediaType` 对最终介质选择生效

完成标志：

- 图片标签在 DYMO 450 上宽高可控
- `fit` / 自动缩放不再篡改标签物理尺寸
- 错误纸张会被显式拒绝

### Phase 4：DYMO 的 `PrintFilePDF` 精确兼容

目标：

1. 对 DYMO 上的 `PrintFilePDF` 默认启用 PDF 栅格化
2. 兼容 `pageSizing`、`printScale`、`printAutoCenter`、`printAutoRotate`
3. 保证前端保持 JSPM 风格调用，但内部不再依赖通用 PDF 查看器路径

完成标志：

- 同一份 PDF 在同一 label stock 上重复打印，尺寸稳定
- 字体大小与布局不再随驱动默认行为漂移
- 前端继续使用 `PrintFilePDF`，无需为 DYMO 单独写另一套 API

### Phase 5：Legacy REST 与管理后台收口

目标：

1. 将现有 `/v1/print`、`/v1/print/batch` 改为 secondary adapter
2. 管理后台与测试脚本继续使用 `/v1/*`
3. 统一所有入口都先归一化到新的打印内核

完成标志：

- 公开主契约已经是 JSPM 风格
- `/v1/*` 仅承担内部/管理/诊断职责
- 没有新的能力只在 REST 路径里实现而不进入 JSPM 兼容层

---

## 13. 错误处理策略

标签打印最怕“看起来成功，实际尺寸错误”。因此错误处理必须偏严格。

以下情况应直接失败，而不是静默降级：

1. DYMO 任务未指定 `paperName`，且无法从默认纸张推断
2. 请求纸张不在该打印机支持列表内
3. PDF 页数不符合标签打印要求（例如多页 PDF 未指定页码）
4. 目标像素尺寸无法根据物理尺寸和 DPI 可靠计算
5. 渲染后内容超出标签边界且 `scale_policy=exact`

错误信息应包含：

- 打印机名
- 纸张名
- 请求尺寸
- 驱动报告尺寸
- 失败阶段（capability / render / submit）

---

## 14. 验收标准

方案落地后，至少满足以下验收条件：

### 14.1 功能验收

1. 现有 JSPM 风格前端代码除脚本来源或本地 host/port 配置外，无需重写业务逻辑即可调用 `JSPM.JSPrintManager.start()` 与本地 OPH 运行时通信
2. `ClientPrintJob`、`InstalledPrinter`、`PrintFilePDF`、`sendToClient()` 可正常工作
3. `getPrintersInfo`、`getPapers`、`getPaperInfo`、`getTrays`、`getMediaTypes` 返回可用能力数据
4. `paperName` / `trayName` / `mediaType` 对 DYMO 介质选择生效
5. PDF 输入在 DYMO 上默认走栅格化标签链路
6. 图片输入在 DYMO 上默认走 exact 标签链路
7. `onUpdated` / `onFinished` / `onError` 与 `WS.onStatusChanged` 能返回一致的任务过程信息

### 14.2 打印质量验收

1. 同一标签重复打印 10 次，物理尺寸一致
2. 文本字号不再因驱动默认缩放漂移
3. 内容不再出现明显“整页被压缩/拉伸”
4. 横竖方向切换后尺寸仍正确
5. `PrintFilePDF.pageSizing = None` + `printScale = 100` 时，输出物理尺寸与 label stock 一致

### 14.3 行为验收

1. A4 办公打印不受标签链路影响
2. Zebra/TSC raw 打印不受影响
3. 当纸张不匹配时，系统明确报错而不是误打
4. 当前 `/v1/*` 接口若保留，其行为必须映射到同一打印内核，而不是形成第二套公开契约

---

## 15. 最终结论

既然当前库还没有用户，**就不应该让现有自定义 REST API 继续决定未来的公开集成方式**。新的方向应该非常明确：

1. **OpenPrintHub 的公开主契约改为 JSPM 兼容的 JS + WebSocket 模型**
2. **当前 `/v1/*` 只保留为管理/诊断/legacy adapter**
3. **DYMO LabelWriter 450 保持 JSPM 风格前端调用，但内部执行改为 label-first、media-first、exact-first**

对 **DYMO LabelWriter 450**：

- **PDF 不是最佳实践的最终执行格式**
- **最佳实践是：标签尺寸先确定，内容先渲染成目标位图，再精确打印**
- **若前端提交的是 `PrintFilePDF`，也应在内部先栅格化，再打印**

换句话说，这次重设计的关键不是“做一个大概像 JSPM 的适配层”，而是：

1. **对业务代码真正像 JSPM**
2. **对 DYMO 标签打印比传统 PDF 直打更稳定**
3. **对不同打印机家族采用正确的底层执行原理**
