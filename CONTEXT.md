# WhatsApp 监控系统

自动监控 WhatsApp Web 聊天记录，识别订单和回款消息，通过 OCR 处理图片和 Excel 文件，并将数据登记到按月分 Sheet 的 Excel 销售明细表中。

## Language

**WhatsApp 消息**:
WhatsApp Web 上来自被监控联系人/群组的一条聊天消息，可以是文本、图片或 .xlsx 文件附件。
_Avoid_: 微信消息、短信

**被监控联系人**:
系统指定监控的 WhatsApp 个人聊天对象，通常是客户或业务员。
_Avoid_: 目标、用户

**被监控群组**:
系统指定监控的 WhatsApp 群组，如销售群、订单群。群内消息的发送者通过手机号与业务员花名册一一对应。
_Avoid_: 聊天室

**业务员花名册**:
Excel 中名为"业务员花名册"的 Sheet，记录每位业务员的姓名、身份证号、手机号、编号和业绩指标。用于群消息发送者身份识别和出库单业务员字段填充。
_Avoid_: 员工表

**订单消息**:
一条被识别为采购订单的 WhatsApp 消息，包含客户名称、联系方式和一个或多个产品明细行。
_Avoid_: 需求、询价

**订单明细行**:
订单消息中的一个产品条目，包含数量（pieces）、产品描述和单价（@价格）。一条订单消息可拆分为多个明细行写入 Excel。

**回款消息**:
一条被识别为收款/付款通知的 WhatsApp 消息，典型格式为 M-PESA 确认短信，包含交易流水号、金额、时间。
_Avoid_: 收款、转账

**M-PESA**:
肯尼亚 Safaricom 运营的移动支付服务，回款消息的主要来源。

**瀑布分配**:
当一笔回款金额大于当前订单行的未收金额时，将剩余金额顺延到下一个未结清行（同月或后续月份）依次分配的算法。

**OCR 提取**:
通过 Tesseract.js 从 WhatsApp 图片消息中识别出的文字内容。用于识别订单明细或回款截图。

**Excel 附件**:
通过 WhatsApp 收到的 `.xlsx` 文件，由 `xlsx` 包直接解析内容。

**销售明细表**:
按月分 Sheet 的 Excel 文件（4月/5月/6月/7月…），每行记录一个订单明细，包含销售信息、收款信息和客户信息。
_Avoid_: 台账、报表

**溢出行**:
当回款金额超过当前订单行的未收金额，或当前行已满时，在下方插入的新行。溢出行复制原行的日期、合同号、订单号、业务员和客户信息，但产品字段留空，仅填写收款信息。

**销售出库单**:
基于订单消息生成的出库单据 Excel 文件，包含收货方信息、产品明细（产品代码、描述、型号、数量、单价、总价）、总金额、业务员和制单人信息，用于仓库发货和出入库管理。
_Avoid_: 送货单、发货单

**Dashboard**:
本地 Web 界面，实时展示 WhatsApp 监控状态、最新消息、分类结果、Excel 写入结果和出库单生成记录。

---

# Claude Code Remote (Mobile Relay)

手机远程遥控电脑上 Claude Code 的系统。IPA（TrollStore）← Tailscale → Go Relay Daemon ←→ Claude Code CLI。

## Language

**Relay Daemon**:
电脑上常驻的 Go 服务进程。提供 REST API + WebSocket，接收手机指令、操作 Claude Code CLI、转发流式响应。
_Avoid_: 后端、服务端、中间件

**Tailscale Network**:
手机和电脑之间的加密 mesh VPN。手机在外用 4G/5G 也能直连电脑，无需公网 IP。
_Avoid_: VPN、代理

**Project**:
一个预配置的代码项目，含 `path`（文件系统路径）和 `name`（显示名）。手机端看到的"项目列表"中每一项对应一个 Project。Claude Code 在该路径下启动 session。
_Avoid_: 仓库、工作区

**Session**:
Claude Code 运行在某个 Project 目录下的一次对话。对应电脑上 `.claude/sessions/` 下的 JSON 文件。包含消息历史、上下文状态。
_Avoid_: 聊天、会话记录

**Session Summary**:
手机端看到的会话摘要（标题、项目名、最后消息时间、消息数量）。由 Relay Daemon 从 Session 文件解析生成。
_Avoid_: 卡片、预览

**Stream**:
手机发起消息后，Relay Daemon 通过 WebSocket 持续推送的 Claude Code 流式输出片段。每帧包含当前已生成的文本块。
_Avoid_: 数据流、推送

**Command**:
手机端发送给 Relay Daemon 的操作指令，如 `send_message`、`interrupt`、`list_sessions`、`start_session` 等。
_Avoid_: 请求、操作

**Interrupt**:
手机端发起的中断操作，Relay Daemon 向正在运行的 Claude Code 进程发送 SIGINT/Terminate，停止当前生成。
_Avoid_: 停止、取消

**Session Archive**:
Claude Code 自动归档的旧会话。手机端可选择是否显示归档会话。
_Avoid_: 历史记录
