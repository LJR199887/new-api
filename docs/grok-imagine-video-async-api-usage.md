# Grok Imagine Video 异步调用文档

本文面向下游系统调用 `grok-imagine-video`。旧模型名 `grok-imagine-1.0-video` 仍可传入，服务端会自动转为上游新模型名 `grok-imagine-video`。

参考上游 `grok2api` 文档：Grok 视频独立接口是 `POST /v1/videos`，使用 multipart 表单提交，创建后通过 `GET /v1/videos/{video_id}` 轮询。本站对外仍提供统一异步接口 `/v1/video/async-generations`，服务端会在后台转换并转发给上游。

## 接口

| 用途 | 方法与路径 |
| --- | --- |
| 提交异步视频任务 | `POST /v1/video/async-generations` |
| 查询异步视频任务 | `GET /v1/video/async-generations/{task_id}` |

认证：

```http
Authorization: Bearer sk-你的令牌
```

## 文生视频

下游可以直接传 JSON：

```bash
curl https://linksky.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video",
    "prompt": "霓虹雨夜街头，电影感慢镜头追拍",
    "seconds": 10,
    "size": "1792x1024",
    "resolution_name": "720p",
    "preset": "normal",
    "async": true
  }'
```

兼容旧模型名：

```json
{
  "model": "grok-imagine-1.0-video",
  "prompt": "A neon rainy street at night, cinematic slow tracking shot",
  "duration": 10,
  "size": "1792x1024",
  "resolution_name": "720p",
  "preset": "normal",
  "async": true
}
```

说明：`duration` 会兼容转换为上游需要的 `seconds`。

## 图生视频

上游 Grok 文档要求参考图使用 multipart 文件字段 `input_reference[]`。因此图生视频建议下游也使用 multipart：

```bash
curl https://linksky.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -F "model=grok-imagine-video" \
  -F "prompt=让参考图里的主体向镜头走来，电影感运镜" \
  -F "seconds=10" \
  -F "size=720x1280" \
  -F "resolution_name=720p" \
  -F "preset=normal" \
  -F "input_reference[]=@/path/to/reference.png"
```

为兼容现有客户端，服务端也会把 multipart 中的 `image`、`image[]`、`images`、`images[]`、`image_reference`、`image_reference[]` 文件字段转为上游的 `input_reference[]`。

## 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 推荐 `grok-imagine-video`；旧名 `grok-imagine-1.0-video` 兼容 |
| `prompt` | string | 是 | 视频提示词 |
| `seconds` | number/string | 否 | 视频长度，上游支持 `6`、`10`、`12`、`16`、`20` |
| `duration` | number/string | 否 | 兼容字段，会转为 `seconds` |
| `size` | string | 否 | `720x1280`、`1280x720`、`1024x1024`、`1024x1792`、`1792x1024` |
| `resolution_name` | string | 否 | `480p` 或 `720p` |
| `preset` | string | 否 | `fun`、`normal`、`spicy`、`custom` |
| `input_reference[]` | file | 否 | 图生视频参考图，multipart 文件字段 |
| `async` | boolean | 否 | 可传 `true`，服务端始终按异步任务处理 |

## 提交响应

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "grok-imagine-video",
  "status": "queued",
  "progress": 10,
  "created_at": 1777188292
}
```

## 查询任务

```bash
curl https://linksky.top/v1/video/async-generations/task_xxx \
  -H "Authorization: Bearer $LINKSKY_API_KEY"
```

完成后优先读取 `video_url`，没有则读取 `url` 或 `data[0].url`：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "grok-imagine-video",
  "status": "completed",
  "progress": 100,
  "video_url": "https://example.com/result.mp4",
  "url": "https://example.com/result.mp4",
  "completed_at": 1777188342
}
```

失败时：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "grok-imagine-video",
  "status": "failed",
  "progress": 100,
  "error": {
    "message": "video generation failed",
    "code": "bad_response"
  }
}
```

## 下游 JS 示例

```js
async function createGrokVideo() {
  const baseUrl = 'https://linksky.top';
  const apiKey = process.env.LINKSKY_API_KEY;

  const submitRes = await fetch(`${baseUrl}/v1/video/async-generations`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      model: 'grok-imagine-video',
      prompt: 'A neon rainy street at night, cinematic slow tracking shot',
      seconds: 10,
      size: '1792x1024',
      resolution_name: '720p',
      preset: 'normal',
      async: true,
    }),
  });

  const task = await submitRes.json();
  const taskId = task.task_id || task.id;

  for (let i = 0; i < 120; i += 1) {
    await new Promise((resolve) => setTimeout(resolve, 3000));
    const pollRes = await fetch(
      `${baseUrl}/v1/video/async-generations/${encodeURIComponent(taskId)}`,
      { headers: { Authorization: `Bearer ${apiKey}` } },
    );
    const data = await pollRes.json();
    const videoUrl = data.video_url || data.url || data.data?.[0]?.url;
    if (data.status === 'completed' && videoUrl) return videoUrl;
    if (data.status === 'failed') {
      throw new Error(data.error?.message || 'video generation failed');
    }
  }

  throw new Error('poll timeout');
}
```

## 注意

- 下游只需要调用本站 `/v1/video/async-generations`；不要直接调用上游 `/v1/videos`。
- 服务端会把 JSON 文生视频转换为上游 multipart 表单。
- 图生视频推荐直接传 multipart 文件，字段名用 `input_reference[]`。
- JSON 图片 URL 不是上游文档推荐格式；如需图生视频，优先上传文件。
