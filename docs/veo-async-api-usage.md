# Veo 异步视频 API 下游调用文档

本文档面向下游系统调用 `veo31-fast` / `veo31-ref` 视频模型，统一使用异步任务模式。

适用模型：
- `veo31-fast`
- `veo31-ref`

## 1. 接入信息

Base URL：

```text
https://linksky.top
```

认证：

```http
Authorization: Bearer sk-你的令牌
Content-Type: application/json
```

接口：

| 用途 | 方法与路径 |
| --- | --- |
| 提交异步视频任务 | `POST /v1/video/async-generations` |
| 查询异步视频任务 | `GET /v1/video/async-generations/{task_id}` |

调用流程：提交任务 -> 获取 `task_id` -> 每 `3-5` 秒轮询 -> `completed` 后读取 `video_url`、`url` 或 `data[0].url`

## 2. 推荐请求格式

Veo 图生视频推荐直接使用上游实际格式：

```json
{
  "model": "veo31-fast",
  "prompt": "Create a smooth cinematic motion",
  "duration": 4,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "reference_mode": "frame",
  "async": true,
  "image_url": "https://example.com/a.png"
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | `veo31-fast` 或 `veo31-ref` |
| `prompt` | string | 是 | 视频生成提示词 |
| `duration` | number | 否 | 视频时长，常用 `4` |
| `aspect_ratio` | string | 否 | 比例，常用 `16:9` / `9:16` |
| `resolution` | string | 否 | 分辨率档位，如 `720p` / `1080p` |
| `reference_mode` | string | 否 | 参考图模式，`veo31-fast` 建议 `frame`，`veo31-ref` 建议 `image` |
| `async` | boolean | 否 | 建议显式传 `true` |
| `image_url` | string | 否 | 图生视频参考图 URL；不传则为文生视频 |

## 3. 兼容旧请求字段

为了兼容旧下游，服务端目前仍支持这些旧字段：

- `image`
- `input_reference`
- `images[0]`
- `image_url`

服务端会自动转换成上游最终需要的 `image_url`。

如果没有显式传 `reference_mode`，服务端会自动补默认值：

- `veo31-fast` -> `frame`
- `veo31-ref` -> `image`

新接入建议直接使用 `image_url + reference_mode`，这样最清晰，也最接近上游原始格式。

## 4. 文生视频

不传 `image_url`：

```json
{
  "model": "veo31-fast",
  "prompt": "A stylish product commercial shot, smooth dolly-in, cinematic lighting, no text, no logo",
  "duration": 4,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "async": true
}
```

## 5. 图生视频

### 5.1 veo31-fast

推荐使用 `reference_mode=frame`：

```json
{
  "model": "veo31-fast",
  "prompt": "Create a smooth cinematic motion",
  "duration": 4,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "reference_mode": "frame",
  "async": true,
  "image_url": "https://example.com/a.png"
}
```

### 5.2 veo31-ref

推荐使用 `reference_mode=image`：

```json
{
  "model": "veo31-ref",
  "prompt": "Animate this still image with subtle natural movement while keeping the original composition",
  "duration": 4,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "reference_mode": "image",
  "async": true,
  "image_url": "https://example.com/a.png"
}
```

## 6. 提交响应

提交成功后会先返回任务信息，通常还没有视频链接：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "veo31-fast",
  "status": "queued",
  "progress": 10,
  "created_at": 1776418394
}
```

下游需要保存 `task_id`，后续轮询任务结果。

## 7. 查询任务

请求：

```bash
curl https://linksky.top/v1/video/async-generations/task_xxx \
  -H "Authorization: Bearer sk-你的令牌"
```

处理中：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "veo31-fast",
  "status": "in_progress",
  "progress": 35,
  "created_at": 1776418394
}
```

完成：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "veo31-fast",
  "status": "completed",
  "video_url": "https://example.com/result.mp4",
  "url": "https://example.com/result.mp4",
  "progress": 100,
  "created_at": 1776418394,
  "completed_at": 1776418477,
  "data": [
    {
      "url": "https://example.com/result.mp4"
    }
  ]
}
```

失败：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "veo31-fast",
  "status": "failed",
  "progress": 100,
  "error": {
    "message": "video generation failed",
    "code": "bad_response"
  }
}
```

## 8. 状态处理

| status | 处理方式 |
| --- | --- |
| `queued` | 继续轮询 |
| `submitted` | 继续轮询 |
| `in_progress` | 继续轮询 |
| `running` | 继续轮询 |
| `completed` | 优先读 `video_url`，没有则读 `url`，再没有则读 `data[0].url` |
| `failed` | 展示 `error.message` 或 `error` |

建议轮询间隔：

- 普通场景每 `3-5` 秒轮询一次
- 最长可轮询 `5-10` 分钟
- 进度值可能长时间停留在较低值，最终再跳到 `100`，这通常是正常现象

## 9. JS 调用示例

```js
async function createVeoVideo() {
  const baseUrl = 'https://linksky.top';
  const apiKey = 'sk-你的令牌';

  const submitRes = await fetch(`${baseUrl}/v1/video/async-generations`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      model: 'veo31-fast',
      prompt: 'Create a smooth cinematic motion',
      duration: 4,
      aspect_ratio: '16:9',
      resolution: '720p',
      reference_mode: 'frame',
      async: true,
      image_url: 'https://example.com/a.png',
    }),
  });

  if (!submitRes.ok) {
    throw new Error(`submit failed: ${submitRes.status}`);
  }

  const submitData = await submitRes.json();
  const taskId = submitData.task_id || submitData.data?.task_id;
  if (!taskId) {
    throw new Error('missing task_id');
  }

  for (let i = 0; i < 120; i += 1) {
    await new Promise((resolve) => setTimeout(resolve, 3000));

    const pollRes = await fetch(
      `${baseUrl}/v1/video/async-generations/${encodeURIComponent(taskId)}`,
      {
        headers: {
          Authorization: `Bearer ${apiKey}`,
        },
      },
    );

    if (!pollRes.ok) {
      throw new Error(`poll failed: ${pollRes.status}`);
    }

    const pollData = await pollRes.json();
    const payload =
      pollData.data && !Array.isArray(pollData.data) ? pollData.data : pollData;
    const videoUrl =
      payload.video_url || payload.url || payload.data?.[0]?.url || '';

    if (payload.status === 'completed' && videoUrl) {
      return videoUrl;
    }

    if (payload.status === 'failed') {
      throw new Error(payload.error?.message || payload.error || 'video generation failed');
    }
  }

  throw new Error('poll timeout');
}
```

## 10. 常见错误

| 状态码 | 含义 | 常见消息 |
| --- | --- | --- |
| `400` | 请求参数错误 | `model is required`、`prompt is required`、`unsupported duration`、`unsupported aspect_ratio`、`unsupported resolution`、`unsupported reference_mode` |
| `401` | 认证失败 | `Token invalid or expired`、`Invalid API key`、`Unauthorized` |
| `403` | 无模型或分组访问权限 | `token has no access to model`、`group access denied` |
| `404` | 任务或资源不存在 | `task_not_exist`、`video generation not found`、`task not found` |
| `429` | 限额或频控 | `Token quota exhausted`、`Too many requests` |
| `500` | 服务内部错误 | `video submit failed`、`video poll failed`、`Unhandled error` |
| `503` | 没有可用渠道或上游临时不可用 | `No available channel for model veo31-fast under group xxx` |

错误响应通常类似：

```json
{
  "error": {
    "message": "unsupported resolution",
    "type": "invalid_request_error",
    "code": "ERR-XXXXXXXXXX"
  }
}
```

## 11. 注意事项

- 下游只调用 `/v1/...` 接口，不要调用内部任务接口
- 图生视频时，图片地址必须可公网访问，建议使用稳定的 HTTPS 图片 URL
- 建议由服务端代理调用，不要把 API Key 暴露在浏览器前端
- 新接入请优先使用 `image_url + reference_mode`
- 旧字段虽然仍兼容，但后续文档和示例都建议按新格式接入
