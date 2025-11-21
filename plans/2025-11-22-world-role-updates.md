# 世界频道权限与旁观角色变更发布说明

## 数据迁移步骤
1. **补充 `world_invites.role` 字段**  
   ```sql
   ALTER TABLE world_invites ADD COLUMN IF NOT EXISTS role VARCHAR(24) DEFAULT 'member';
   UPDATE world_invites SET role = 'member' WHERE COALESCE(role, '') = '';
   ```
2. **为历史频道补充 `spectator` 角色**  
   新版本启动后会在创建频道时自动生成，如需立即落地可在数据库中执行：
   ```sql
   INSERT INTO perm_channel_roles (id, name, desc, channel_id, created_at, updated_at)
   SELECT DISTINCT 'ch-' || id || '-spectator', '旁观者', '自动生成的旁观角色', id, NOW(), NOW()
   FROM channels c
   WHERE NOT EXISTS (
     SELECT 1 FROM perm_channel_roles r WHERE r.id = 'ch-' || c.id || '-spectator'
   );
   ```
   然后为新建角色写入权限：`func_channel_read` 与 `func_channel_read_all`。
3. **回填世界管理员/旁观者的频道角色**  
   运行新提供的工具：
   ```bash
   go run ./cmd/world_role_sync
   ```
   该脚本会遍历所有活跃世界，把世界拥有者/管理员同步为每个频道的管理员，并为旁观者补齐 `spectator` 频道角色。

## 验证方案
1. **管理员同步**：在旧世界中选一名管理员，运行脚本后检查其是否在所有频道的“管理员”列表里，并验证可以编辑频道信息。
2. **旁观邀请**：通过世界管理创建“旁观邀请”，使用新账号加入后确认可阅读所有频道但发送消息会被拒绝（缺少发送权限）。
3. **频道成员选择器**：在频道设置 -> 成员管理中，确认“成员”角色的候选列表展示来自世界成员，而非好友列表，且搜索/翻页正常。
4. **邀请链接展示**：世界管理的邀请面板应分别展示“成员”和“旁观”两类链接，复制后的 URL 能被消费。
5. **世界大厅标签**：在世界大厅我的世界列表中，角色标签会显示“拥有者/管理员/旁观者/成员”，并可从旁观者手动退出世界。
6. **回归测试**：抽样验证频道消息收发、好友聊天、世界管理 CRUD 与现有邀请消费流程，确保没有权限回退。
