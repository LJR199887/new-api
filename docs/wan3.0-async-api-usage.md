# Wan 3.0 系列异步视频 API 下游调用文档

本文档适用于 `wan3.0-480p`、`wan3.0-720p`、`wan3.0-1080p`。三个模型均使用与 `video-2.5` 相同的异步请求格式，并按生成视频时长计费。

## 1. 接入信息

```text
Base URL: https://你的域名
提交任务: POST /v1/video/async-generations
查询任务: GET  /v1/video/async-generations/{task_id}
```

也兼容 `/v1/video/generations` 及其查询路径。认证请求头：

```http
Authorization: Bearer sk-你的令牌
Content-Type: application/json
```

调用流程：提交任务 → 保存 `task_id` → 每 `3-5` 秒轮询 → 完成后读取 `url`。

## 2. 模型、时长与计费

| 模型 | 默认时长 | 支持时长 | 计费方式 |
| --- | ---: | --- | --- |
| `wan3.0-480p` | 5 秒 | 2–30 秒 | 按秒计费 |
| `wan3.0-720p` | 5 秒 | 2–30 秒 | 按秒计费 |
| `wan3.0-1080p` | 5 秒 | 2–30 秒 | 按秒计费 |

计费秒数优先读取 `duration`，也兼容 `seconds`。例如每秒价格为 `0.05`，生成 10 秒视频时，模型费用为 `0.05 × 10`，再按系统分组倍率结算。

## 3. 请求参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 三个正式模型名之一 |
| `prompt` | string | 是 | 视频提示词 |
| `duration` | integer | 否 | 生成时长，2–30 秒，默认 5 秒 |
| `seconds` | integer/string | 否 | `duration` 的兼容字段 |
| `aspect_ratio` | string | 否 | `16:9`、`4:3`、`1:1`、`3:4`、`9:16` |
| `size` | string | 否 | 必须属于当前模型的固定尺寸档位 |
| `width` / `height` | integer | 否 | 显式宽高，需要成对传入 |
| `image_url` | string | 否 | 单张参考图片 URL |
| `image_urls` / `images` | string[] | 否 | 多张参考图片，最多 10 张 |
| `image_guidance` | object[] | 否 | 图片对象数组，可携带 `url`、`strength` |
| `start_image_url` / `end_image_url` | string | 否 | 首帧、尾帧图片 URL |
| `start_frame` / `end_frame` | object[] | 否 | 首尾帧对象数组，格式为 `[{"url":"..."}]` |
| `video_url` | string | 否 | 单个参考视频 URL |
| `video_reference` | object[] | 否 | 多视频参考，最多 5 个 |
| `audio_url` | string | 否 | 单个参考音频 URL |
| `audio_reference` | object[] | 否 | 多音频参考，最多 5 个 |
| `async` | boolean | 否 | 兼容字段；接口始终异步执行 |

尺寸字段优先级：`width` / `height` 高于 `size`，`size` 高于 `aspect_ratio`。

## 4. 固定尺寸映射

| `aspect_ratio` | `wan3.0-480p` | `wan3.0-720p` | `wan3.0-1080p` |
| --- | --- | --- | --- |
| `16:9` | `854x480` | `1280x720` | `1920x1080` |
| `4:3` | `736x552` | `1104x828` | `1656x1242` |
| `1:1` | `640x640` | `960x960` | `1440x1440` |
| `3:4` | `552x736` | `828x1104` | `1242x1656` |
| `9:16` | `480x854` | `720x1280` | `1080x1920` |

每个模型只能使用本档尺寸，不能跨档传递。推荐只传 `aspect_ratio`，由网关自动换算。

## 5. 素材限制

| 素材 | 数量限制 | 单个时长 | 总时长 |
| --- | ---: | --- | --- |
| 图片参考 | 最多 10 张 | - | - |
| 视频参考 | 最多 5 个 | 1–15 秒 | 不超过 15 秒 |
| 音频参考 | 最多 5 个 | 1–15 秒 | 不超过 15 秒 |

- 支持文生视频、图片参考、首尾帧、视频参考和音频参考。
- 图片、视频、音频可以按业务需要组合使用。
- `video_reference[].duration` 和 `audio_reference[].duration` 是参考素材自身时长。
- 参考对象可以不传 `duration`；若传入，网关会校验单个时长和总时长。
- 顶层 `duration` 表示生成结果时长，并用于按秒计费。

## 6. 请求示例

### 文生视频

```json
{
  "model": "wan3.0-720p",
  "prompt": "一艘帆船穿过晨雾，镜头缓慢推进",
  "duration": 10,
  "aspect_ratio": "16:9"
}
```

### 多图参考

```json
{
  "model": "wan3.0-1080p",
  "prompt": "让参考人物出现在参考场景中并自然行走",
  "duration": 8,
  "aspect_ratio": "9:16",
  "image_urls": [
    "https://example.com/person.png",
    "https://example.com/scene.png"
  ]
}
```

### 首尾帧

```json
{
  "model": "wan3.0-480p",
  "prompt": "花朵从含苞自然绽放",
  "duration": 6,
  "aspect_ratio": "1:1",
  "start_image_url": "https://example.com/start.png",
  "end_image_url": "https://example.com/end.png"
}
```

### 视频参考

```json
{
  "model": "wan3.0-720p",
  "prompt": "参考视频中的运镜和动作节奏生成新场景",
  "duration": 12,
  "aspect_ratio": "16:9",
  "video_reference": [
    {"url": "https://example.com/reference-1.mp4", "duration": 7},
    {"url": "https://example.com/reference-2.mp4", "duration": 8}
  ]
}
```

### 图片、视频与音频组合参考

```json
{
  "model": "wan3.0-1080p",
  "prompt": "结合人物形象、动作节奏和配乐生成广告短片",
  "duration": 15,
  "aspect_ratio": "9:16",
  "image_url": "https://example.com/product.png",
  "video_reference": [
    {"url": "https://example.com/motion.mp4", "duration": 10}
  ],
  "audio_reference": [
    {"url": "https://example.com/music.mp3", "duration": 15}
  ]
}
```

## 7. 提交与查询响应

提交成功示例：

```json
{
  "id": "task_01WAN30",
  "task_id": "task_01WAN30",
  "object": "video.generation",
  "model": "wan3.0-720p",
  "status": "in_progress",
  "progress": 0,
  "created_at": 1760000000
}
```

查询任务：

```bash
curl "https://你的域名/v1/video/async-generations/task_01WAN30" \
  -H "Authorization: Bearer sk-你的令牌"
```

完成响应示例：

```json
{
  "id": "task_01WAN30",
  "task_id": "task_01WAN30",
  "object": "video.generation",
  "model": "wan3.0-720p",
  "status": "completed",
  "url": "https://example.com/generated-video.mp4",
  "progress": 100,
  "created_at": 1760000000,
  "completed_at": 1760000090
}
```

常见状态包括 `submitted`、`queued`、`processing`、`in_progress`、`completed`、`failed`，应以实际返回的 `status` 为准。

## 8. 常见错误说明

| 错误信息 | 说明与处理建议 |
| --- | --- |
| `duration must be an integer between 2 and 30` | 生成时长不在 2–30 秒范围内。 |
| `supports at most 10 image references` | 图片参考超过 10 张。 |
| `supports at most 5 video references` | 视频参考超过 5 个。 |
| `video_reference[n].duration must be between 1 and 15 seconds` | 单个参考视频时长不在 1–15 秒范围内。 |
| `total video reference duration must not exceed 15 seconds` | 所有参考视频总时长超过 15 秒。 |
| `unsupported size` | 尺寸不属于当前模型的固定分辨率档位。 |
| 图片或视频 URL 返回 404 | 素材链接失效，或上游无法访问。 |
| `PROVIDER_TIMEOUT` | 上游生成超时，可稍后重试并检查内容审核结果。 |

## 9. cURL 示例

```bash
curl -X POST "https://你的域名/v1/video/async-generations" \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "wan3.0-720p",
    "prompt": "一列火车穿过雪山峡谷，航拍镜头",
    "duration": 10,
    "aspect_ratio": "16:9"
  }'
```
