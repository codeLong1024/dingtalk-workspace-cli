---
category: Fixed
---

- **群公告定时发布** (#1438) — `chat group notice create --run-at` 现在将输入时间归一化为后端 `runAtText` 要求的 `yyyy-MM-dd HH:mm:ss`（北京时间）格式，兼容 ISO-8601 输入（如 `2026-07-03T09:00:00+08:00`），非法输入返回参数校验错误；修正帮助与技能文档中错误的 ISO-8601 示例。
