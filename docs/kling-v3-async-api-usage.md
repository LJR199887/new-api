# kling-v3 异步视频 API 下游调用文档

本文档面向下游系统调用 `kling-v3` 视频模型。该模型统一走异步任务接口，支持文生视频和图生视频。

## 1. 接入信息

Base URL：

```text
https://linksky.top
```

请求头：

```http
Authorization: Bearer sk-你的令牌
Content-Type: application/json
```

接口：

| 用途 | 方法与路径 |
| --- | --- |
| 提交异步视频任务 | `POST /v1/video/async-generations` |
| 查询异步视频任务 | `GET /v1/video/async-generations/{task_id}` |

调用流程：提交任务 -> 获取 `task_id` -> 每 `3-5` 秒轮询查询接口 -> `completed` 后读取 `video_url`、`url` 或 `data[0].url`。

## 2. 请求参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 固定为 `kling-v3` |
| `prompt` | string | 是 | 视频生成提示词 |
| `duration` | number | 否 | 视频时长，允许 `3-15` 秒，默认 `5` |
| `aspect_ratio` | string | 否 | 允许 `16:9` 或 `9:16`，默认 `16:9` |
| `generate_audio` | boolean | 否 | 是否生成音频，默认 `true` |
| `generateAudio` | boolean | 否 | 音频开关兼容字段，建议和 `generate_audio` 同时传 `true` |
| `async` | boolean | 否 | 建议传 `true` |
| `image_url` | string | 否 | 图生视频参考图 URL |
| `images` | string[] | 否 | 图生视频参考图 URL 数组，最多使用前 2 张 |

`kling-v3` 按次计费；`duration` 只影响生成时长，不作为下游按秒倍率扣费。

## 3. 文生视频

```bash
curl -X POST "https://linksky.top/v1/video/async-generations" \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "kling-v3",
    "prompt": "奥特曼在城市废墟中打怪兽，电影级特摄镜头，环境音真实",
    "duration": 8,
    "aspect_ratio": "16:9",
    "generate_audio": true,
    "generateAudio": true,
    "async": true
  }'
```

成功响应示例：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "status": "queued"
}
```

## 4. 图生视频

单图：

```bash
curl -X POST "https://linksky.top/v1/video/async-generations" \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "kling-v3",
    "prompt": "让画面中的角色向镜头走来，电影级运镜，环境音真实",
    "duration": 15,
    "aspect_ratio": "9:16",
    "generate_audio": true,
    "generateAudio": true,
    "async": true,
    "image_url": "https://example.com/character.png"
  }'
```

双图：

```bash
curl -X POST "https://linksky.top/v1/video/async-generations" \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "kling-v3",
    "prompt": "根据两张参考图生成一段连贯的电影感运动镜头",
    "duration": 10,
    "aspect_ratio": "16:9",
    "generate_audio": true,
    "generateAudio": true,
    "async": true,
    "images": [
      "https://example.com/start.png",
      "https://example.com/end.png"
    ]
  }'
```

兼容字段：服务端也接受 `image`、`input_reference`、`image_reference`，会自动转换为上游需要的参考图字段。新接入建议直接使用 `image_url` 或 `images`。

## 5. 查询任务

```bash
curl -X GET "https://linksky.top/v1/video/async-generations/task_xxx" \
  -H "Authorization: Bearer sk-你的令牌"
```

生成中：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "status": "in_progress",
  "progress": "35%"
}
```

成功：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "status": "completed",
  "video_url": "https://example.com/result.mp4",
  "url": "https://example.com/result.mp4"
}
```

失败：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "status": "failed",
  "error": {
    "message": "upstream error message"
  }
}
```

## 6. 常见错误

| 状态码 | 场景 | 处理建议 |
| --- | --- | --- |
| `400` | `duration` 不在 `3-15` 秒 | 调整为合法时长 |
| `400` | `aspect_ratio` 不是 `16:9` 或 `9:16` | 使用支持的比例 |
| `401` | 令牌无效 | 检查 `Authorization` |
| `429` | 并发或速率限制 | 降低并发或稍后重试 |
| `500/502/503` | 上游临时异常 | 稍后重试，必要时联系管理员 |
