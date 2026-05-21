# ko3 异步视频 API 下游调用文档

本文档面向下游系统调用 `ko3` 视频模型。下游统一调用 `POST /v1/video/async-generations`，`model` 固定传 `ko3`。

兼容别名：`kling-o3`、`kling-video-o-3` 仅作为兼容保留，新接入建议统一使用 `ko3`。

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

调用流程：提交任务 -> 获取 `task_id` -> 每 `3-5` 秒轮询 -> `completed` 后读取 `video_url`、`url` 或 `data[0].url`。

## 2. 通用规则

- 所有请求都需要传 `prompt`。
- `size` 支持 `1440x1440`、`1080x1920`、`1920x1080`，分别对应 `1:1`、`9:16`、`16:9`。
- 文生视频、单图生视频、多图生视频、首尾帧模式可传 `duration` 和 `size`。
- `duration` 支持 `3-15` 秒。
- 图片大小不能超过 `5MB`。
- 多图模式最多上传 `7` 张图。
- 视频只能上传 `1` 个，视频时长需在 `3-10` 秒内，大小不能超过 `200MB`。
- 多图 + 视频模式下，图片最多上传 `4` 张。
- 视频参考、图片 + 视频、多图 + 视频模式不需要传 `duration` 和 `size`，默认按 `9:16` 处理，生成时长由视频素材决定。
- 远程图片优先使用 `image_url` / `image_urls` / `start_image_url` / `end_image_url`。
- 远程视频优先使用 `video_url`。

## 3. 推荐字段

| 模式 | 必填字段 | 可选字段 | 说明 |
| --- | --- | --- | --- |
| 文生视频 | `prompt`, `model` | `duration`, `size` | 不传默认 `duration=3`、`size=1080x1920` |
| 单图生视频 | `prompt`, `model`, `image_url` | `duration`, `size` | 图片作为参考图 |
| 多图生视频 | `prompt`, `model`, `image_urls` | `duration`, `size` | `image_urls` 按数组顺序作为多图参考，最多 `7` 张 |
| 首尾帧 | `prompt`, `model`, `start_image_url`, `end_image_url` | `duration`, `size` | 服务会分别转成首帧和尾帧参考 |
| 视频生视频 | `prompt`, `model`, `video_url` | 不建议传 `duration`, `size` | 默认使用上游视频参考参数 |
| 图片 + 视频生视频 | `prompt`, `model`, `image_url`, `video_url` | 不建议传 `duration`, `size` | 图片作为参考图，视频作为参考视频 |
| 多图 + 视频生视频 | `prompt`, `model`, `image_urls`, `video_url` | 不建议传 `duration`, `size` | 多张图片作为参考图，视频作为参考视频，图片最多 `4` 张 |

## 4. 文生视频

```json
{
  "prompt": "龟兔赛跑，电影级运动镜头，真实光影，动作自然",
  "model": "ko3",
  "duration": 3,
  "size": "1080x1920"
}
```

## 5. 单图生视频

```json
{
  "prompt": "猫咪跳舞，镜头轻微推进，动作自然",
  "model": "ko3",
  "duration": 3,
  "size": "1080x1920",
  "image_url": "https://example.com/cat.png"
}
```

## 6. 多图生视频

```json
{
  "prompt": "动物世界，将多张参考图中的主体组合成自然运动的视频",
  "model": "ko3",
  "duration": 3,
  "size": "1080x1920",
  "image_urls": [
    "https://example.com/a.png",
    "https://example.com/b.png",
    "https://example.com/c.png"
  ]
}
```

## 7. 首尾帧

```json
{
  "prompt": "从图一过渡到图二，镜头平滑推进，自然转场",
  "model": "ko3",
  "duration": 3,
  "size": "1080x1920",
  "start_image_url": "https://example.com/start.png",
  "end_image_url": "https://example.com/end.png"
}
```

## 8. 视频生视频

视频参考模式不需要传 `duration` 和 `size`。

```json
{
  "prompt": "把视频中的香水替换成牙膏，保持原视频运动节奏",
  "model": "ko3",
  "video_url": "https://example.com/source.mp4"
}
```

## 9. 图片 + 视频生视频

图片 + 视频模式不需要传 `duration` 和 `size`。

```json
{
  "prompt": "把视频中的香水替换成图片里的小熊，保持镜头运动",
  "model": "ko3",
  "image_url": "https://example.com/bear.png",
  "video_url": "https://example.com/source.mp4"
}
```

## 10. 多图 + 视频生视频

多图 + 视频模式不需要传 `duration` 和 `size`，图片最多 `4` 张。

```json
{
  "prompt": "用多张图片替换视频主体，保持视频中的运动和构图",
  "model": "ko3",
  "image_urls": [
    "https://example.com/a.png",
    "https://example.com/b.png"
  ],
  "video_url": "https://example.com/source.mp4"
}
```

## 11. 查询任务

```bash
curl -X GET "https://linksky.top/v1/video/async-generations/task_xxx" \
  -H "Authorization: Bearer sk-你的令牌"
```

处理中：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "status": "in_progress",
  "progress": "35%"
}
```

完成：

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

## 12. 常见错误

| 状态码 | 场景 | 处理建议 |
| --- | --- | --- |
| `400` | `duration` 不在 `3-15` 秒 | 调整为合法时长 |
| `400` | `size` 不是 `1440x1440`、`1080x1920`、`1920x1080` | 使用支持的尺寸 |
| `400` | 多图数量超过限制 | 纯多图最多 `7` 张，多图 + 视频最多 `4` 张 |
| `400` | 视频数量超过限制 | 只上传 `1` 个视频 |
| `401` | 令牌无效 | 检查 `Authorization` |
| `429` | 并发或速率限制 | 降低并发或稍后重试 |
| `500/502/503` | 上游临时异常 | 稍后重试，必要时联系管理员 |
