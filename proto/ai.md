# Proto 文件同步指南

本文档用于跟踪官方 proto 定义与本地 proto 文件的同步。

## 文件位置

### 官方 JSON 定义

| 协议 | JSON 文件路径 |
|------|--------------|
| Conn 连接层 | `openclaw-plugin-yuanbao/dist/src/yuanbao-server/ws/proto/conn.json` |
| Biz 业务层 | `openclaw-plugin-yuanbao/dist/src/yuanbao-server/ws/proto/biz.json` |

### 本地 Proto 文件

| 协议 | Proto 文件路径 |
|------|--------------|
| Conn 连接层 | `yuanbao-go/proto/conn.proto` |
| Biz 业务层 | `yuanbao-go/proto/biz.proto` |

---

## 同步步骤

### 1. 检查 JSON 中的定义

```bash
# 统计 conn.json 中的 message 和 enum 数量
grep -n '"fields":' openclaw-plugin-yuanbao/dist/src/yuanbao-server/ws/proto/conn.json
grep -n '"values":' openclaw-plugin-yuanbao/dist/src/yuanbao-server/ws/proto/conn.json

# 提取所有定义名称
grep -B2 '"fields":' openclaw-plugin-yuanbao/dist/src/yuanbao-server/ws/proto/conn.json | grep -v '"fields"'
```

### 2. 对比本地 proto 文件

```bash
# 查看 conn.proto 中所有定义
grep -n '^message\|^enum' yuanbao-go/proto/conn.proto

# 查看 biz.proto 中所有定义
grep -n '^message\|^enum' yuanbao-go/proto/biz.proto
```

### 3. 缺失项添加到 proto 文件

根据 JSON 定义，补充本地 proto 文件中缺失的 message 和 enum。

### 4. 添加中文注解

为每个 message 和 enum 添加中文注释：

- message 注释格式：`// 消息说明`
- 字段注释格式：`// 字段说明`
- enum 值注释格式：`// 值说明`

---

## 当前状态

### conn.proto ✅ 已完成

| # | 名称 | 类型 | 状态 |
|---|------|------|------|
| 1 | HeadMeta | message | ✅ |
| 2 | Head | message | ✅ |
| 3 | ConnMsg | message | ✅ |
| 4 | DirectedPush | message | ✅ |
| 5 | AuthInfo | message | ✅ |
| 6 | RetCode | enum | ✅ |
| 7 | DeviceInfo | message | ✅ |
| 8 | Container | message | ✅ |
| 9 | AuthBindReq | message | ✅ |
| 10 | AuthBindRsp | message | ✅ |
| 11 | PingReq | message | ✅ |
| 12 | PingRsp | message | ✅ |
| 13 | KickoutMsg | message | ✅ |
| 14 | Meta | message | ✅ |
| 15 | UpdateMetaReq | message | ✅ |
| 16 | UpdateMetaRsp | message | ✅ |
| 17 | PushMsg | message | ✅ |

### biz.proto ✅ 已完成

| # | 名称 | 类型 | 状态 |
|---|------|------|------|
| 1 | ImMsgSeq | message | ✅ |
| 2 | ImImageInfoArray | message | ✅ |
| 3 | MsgContent | message | ✅ |
| 4 | MsgBodyElement | message | ✅ |
| 5 | LogInfoExt | message | ✅ |
| 6 | EnumCLawMsgType | enum | ✅ |
| 7 | SendC2CMessageReq | message | ✅ |
| 8 | SendC2CMessageRsp | message | ✅ |
| 9 | SendGroupMessageReq | message | ✅ |
| 10 | SendGroupMessageRsp | message | ✅ |
| 11 | InboundMessagePush | message | ✅ |
| 12 | GetGroupMemberListReq | message | ✅ |
| 13 | GetGroupMemberListRsp | message | ✅ |
| 14 | Member | message | ✅ |
| 15 | QueryGroupInfoReq | message | ✅ |
| 16 | QueryGroupInfoRsp | message | ✅ |
| 17 | GroupInfo | message | ✅ |
| 18 | EnumHeartbeat | enum | ✅ |
| 19 | SendPrivateHeartbeatReq | message | ✅ |
| 20 | SendPrivateHeartbeatRsp | message | ✅ |
| 21 | SendGroupHeartbeatReq | message | ✅ |
| 22 | SendGroupHeartbeatRsp | message | ✅ |

---

## 更新日志

### 2024-xx-xx - 初始化同步

- [x] 检查 conn.json 并同步到 conn.proto
- [x] 检查 biz.json 并同步到 biz.proto
- [x] 为所有定义添加中文注解
