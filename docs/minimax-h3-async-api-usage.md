# MiniMax H3 系列异步视频 API 下游调用文档

本文档适用于 `minimax-h3-480p`、`minimax-h3-768p`、`minimax-h3-2k`、`minimax-h3-4k`。四个模型均使用与 `video-2.5` 相同的异步请求格式，并按生成视频时长计费。

下游应使用上述带分辨率后缀的正式模型名，不应使用 `h3` 或 `hailuo-03`。

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
| `minimax-h3-480p` | 5 秒 | 5–15 秒 | 按秒计费 |
| `minimax-h3-768p` | 5 秒 | 5–15 秒 | 按秒计费 |
| `minimax-h3-2k` | 5 秒 | 5–15 秒 | 按秒计费 |
| `minimax-h3-4k` | 5 秒 | 5–15 秒 | 按秒计费 |

计费秒数优先读取 `duration`，也兼容 `seconds`。例如每秒价格为 `0.1`，生成 8 秒视频时，模型费用为 `0.1 × 8`，再按系统分组倍率结算。

## 3. 请求参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 四个正式模型名之一 |
| `prompt` | string | 是 | 视频提示词 |
| `duration` | integer | 否 | 生成时长，5–15 秒，默认 5 秒 |
| `seconds` | integer/string | 否 | `duration` 的兼容字段 |
| `aspect_ratio` | string | 否 | `16:9`、`9:16`、`1:1`、`4:3`、`3:4`、`21:9` |
| `size` | string | 否 | 必须属于当前模型的固定尺寸档位 |
| `width` / `height` | integer | 否 | 显式宽高，需要成对传入 |
| `image_url` | string | 否 | 单张参考图片 URL |
| `image_urls` / `images` | string[] | 否 | 多张参考图片，最多 9 张 |
| `image_guidance` | object[] | 否 | 图片对象数组，可携带 `url`、`strength` |
| `start_image_url` / `end_image_url` | string | 否 | 首帧、尾帧图片 URL |
| `start_frame` / `end_frame` | object[] | 否 | 首尾帧对象数组，格式为 `[{"url":"..."}]` |
| `video_url` | string | 否 | 单个参考视频 URL |
| `video_reference` | object[] | 否 | 多视频参考，最多 3 个 |
| `audio_url` | string | 否 | 单个参考音频 URL |
| `audio_reference` | object[] | 否 | 多音频参考，最多 3 个 |
| `async` | boolean | 否 | 兼容字段；接口始终异步执行 |

尺寸字段优先级：`width` / `height` 高于 `size`，`size` 高于 `aspect_ratio`。

## 4. 固定尺寸映射

| `aspect_ratio` | `minimax-h3-480p` | `minimax-h3-768p` | `minimax-h3-2k` | `minimax-h3-4k` |
| --- | --- | --- | --- | --- |
| `16:9` | `856x480` | `1376x768` | `2560x1440` | `3840x2160` |
| `9:16` | `480x856` | `768x1376` | `1440x2560` | `2160x3840` |
| `1:1` | `480x480` | `768x768` | `1440x1440` | `2160x2160` |
| `4:3` | `640x480` | `1024x768` | `1920x1440` | `2880x2160` |
| `3:4` | `480x640` | `768x1024` | `1440x1920` | `2160x2880` |
| `21:9` | `1120x480` | `1792x768` | `3360x1440` | `5040x2160` |

每个模型只能使用本档尺寸，不能跨档传递。推荐只传 `aspect_ratio`，由网关自动换算。

## 5. 素材限制与模式规则

| 素材 | 数量限制 | 单个时长 | 总时长 |
| --- | ---: | --- | --- |
| 图片参考 | 最多 9 张 | - | - |
| 视频参考 | 最多 3 个 | 1–15 秒 | 不超过 15 秒 |
| 音频参考 | 最多 3 个 | 1–15 秒 | 不超过 15 秒 |

- 不传参考素材时为文生视频。
- 图片、视频、音频参考可以单独或组合使用。
- `start_image_url`、`start_frame`、`end_image_url`、`end_frame` 属于首尾帧模式。
- 图片、视频、音频组成的多模态参考不能与首尾帧模式混用。
- `video_reference[].duration` 和 `audio_reference[].duration` 是参考素材自身时长，不是生成时长。
- 参考对象可以不传 `duration`；若传入，网关会校验单个时长和总时长。

## 6. 请求示例

### 文生视频

```json
{
  "model": "minimax-h3-2k",
  "prompt": "一只白色机器人在海边缓慢行走，电影感光影",
  "duration": 8,
  "aspect_ratio": "16:9"
}
```

### 多图参考

```json
{
  "model": "minimax-h3-768p",
  "prompt": "让参考角色出现在参考场景中，自然运动",
  "duration": 6,
  "aspect_ratio": "9:16",
  "image_urls": [
    "https://example.com/character.png",
    "https://example.com/scene.png"
  ]
}
```

### 首尾帧

```json
{
  "model": "minimax-h3-4k",
  "prompt": "从清晨自然过渡到黄昏",
  "duration": 10,
  "aspect_ratio": "16:9",
  "start_image_url": "https://example.com/start.png",
  "end_image_url": "https://example.com/end.png"
}
```

### 图片、视频与音频多模态参考

```json
{
  "model": "minimax-h3-2k",
  "prompt": "结合角色、动作和配乐生成电影预告片",
  "duration": 12,
  "aspect_ratio": "21:9",
  "image_url": "https://example.com/character.png",
  "video_reference": [
    {"url": "https://example.com/motion-1.mp4", "duration": 6},
    {"url": "https://example.com/motion-2.mp4", "duration": 7}
  ],
  "audio_reference": [
    {"url": "https://example.com/music.mp3", "duration": 12}
  ]
}
```

## 7. 提交与查询响应

提交成功示例：

```json
{
  "id": "task_01HXYZ",
  "task_id": "task_01HXYZ",
  "object": "video.generation",
  "model": "minimax-h3-2k",
  "status": "in_progress",
  "progress": 0,
  "created_at": 1760000000
}
```

查询任务：

```bash
curl "https://你的域名/v1/video/async-generations/task_01HXYZ" \
  -H "Authorization: Bearer sk-你的令牌"
```

完成响应示例：

```json
{
  "id": "task_01HXYZ",
  "task_id": "task_01HXYZ",
  "object": "video.generation",
  "model": "minimax-h3-2k",
  "status": "completed",
  "url": "https://example.com/generated-video.mp4",
  "progress": 100,
  "created_at": 1760000000,
  "completed_at": 1760000120
}
```

常见状态包括 `submitted`、`queued`、`processing`、`in_progress`、`completed`、`failed`，应以实际返回的 `status` 为准。

## 8. 常见错误说明

| 错误码 / 错误信息 | 说明与处理建议 |
| --- | --- |
| `duration must be an integer between 5 and 15` | 生成时长不在 5–15 秒范围内。 |
| `supports at most 9 image references` | 图片参考超过 9 张。 |
| `supports at most 3 video references` | 视频参考超过 3 个。 |
| `total video reference duration must not exceed 15 seconds` | 参考视频总时长超过 15 秒。 |
| `multimodal references and frame mode cannot be combined` | 多模态参考与首尾帧模式冲突。 |
| `unsupported size` | 尺寸不属于当前模型的固定分辨率档位。 |
| `502`，`generation failed: generate error: An error occurred.` | 通常是提示词超出上游限制，请缩短后重试。 |
| `400`，`invalid image_urls[n]: image url returned 404` | 图片链接无效或上游无法访问。 |
| `400`，`invalid audio_url: audio url did not return a supported audio type` | 音频链接或格式无效。 |
| `400`，`invalid image_urls[n]: wait for init image failed: graphql request failed: Post "https://...": EOF` | 上游短暂网络波动，通常重试即可。 |
| `PROVIDER_TIMEOUT` | 生成超时；若带 `CHILD`、`TRADEMARK`、`NSFW`、`NUDITY`，通常表示内容审核未通过或卡住。 |

## 9. cURL 示例

```bash
curl -X POST "https://你的域名/v1/video/async-generations" \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "minimax-h3-2k",
    "prompt": "一只机器人穿过雨夜中的霓虹街道",
    "duration": 8,
    "aspect_ratio": "16:9"
  }'
```
