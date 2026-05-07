# Seedance 异步视频 API 下游调用文档

本文档面向下游系统调用 `Seedance` 视频模型，统一使用异步任务模式。

适用模型：
- `video-2.0`
- `video-2.0-fast`

## 1. 接入信息

Base URL：
```text
https://你的域名
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

调用流程：提交任务 -> 获取 `task_id` -> 每 `3-5` 秒轮询 -> `completed` 后读取 `url`。

## 2. 模型参数范围

`video-2.0`：
- 时长：`4-15` 秒
- 比例：`9:16`、`16:9`、`1:1`
- 分辨率：`720p`

`video-2.0-fast`：
- 时长：`4-15` 秒
- 比例：`9:16`、`16:9`、`1:1`
- 分辨率：`720p`

## 3. 提交参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | `video-2.0` 或 `video-2.0-fast` |
| `prompt` | string | 是 | 视频生成提示词，不能为空，字符数不能超过 `1500` |
| `duration` | number | 否 | 视频时长，推荐 `4-15` |
| `size` | string | 否 | 直接指定输出尺寸，如 `1280x720`、`720x1280`、`720x720` |
| `aspect_ratio` | string | 否 | 视频比例，仅支持 `9:16`、`16:9`、`1:1` |
| `resolution` | string | 否 | 输出分辨率，当前仅支持 `720p` |
| `image_url` | string | 否 | 单图参考模式使用的图片 URL；不传即为文生视频 |
| `image_urls` | string[] | 否 | 多图参考模式使用的图片 URL 数组，最多 `4` 张 |
| `video_url` | string | 否 | 单视频参考模式使用的视频素材 URL |
| `video_reference` | object[] | 否 | 多视频参考模式使用的视频数组，格式为 `[{ "url": "..." }]`，最多 `3` 个 |
| `start_image_url` | string | 否 | 首尾帧模式的起始图 URL |
| `end_image_url` | string | 否 | 首尾帧模式的结束图 URL |
| `images` | string[] | 否 | `image_urls` 的兼容别名，最多 `4` 张 |
| `async` | boolean | 否 | 建议固定传 `true` |

说明：
- 下游调用时建议显式传 `duration`、`aspect_ratio`、`resolution`，不要依赖默认值。
- 如果你已经能明确给出尺寸，也可以直接传 `size`。
- 如果使用 `size`，建议只传与 `9:16`、`16:9`、`1:1` 对应的 `720p` 尺寸。
- 多图参考最多上传 `4` 张图片。
- 上传视频素材时，最多 `3` 个视频，总大小不能超过 `200MB`，总时长不能超过 `15` 秒。
- 视频参考模式下，单个参考视频的分辨率必须在 `720px` 到 `2160px` 之间，否则上游会返回“视频分辨率不支持”错误。
- `prompt` 字符数不能超过 `1500`，超过后可能会被上游拒绝或导致生成失败。
- `prompt` 建议避免违规、侵权、涉政、涉黄等高风险内容。

## 4. 文生视频示例

请求体：

```json
{
  "model": "video-2.0",
  "prompt": "一个霓虹夜景街头的时尚模特向前走来，镜头轻微跟拍，人物动作自然，无文字，无logo",
  "duration": 4,
  "aspect_ratio": "9:16",
  "resolution": "720p",
  "async": true
}
```

## 5. 图生视频示例

请求体：

```json
{
  "model": "video-2.0-fast",
  "prompt": "让图片中的主体自然动起来，镜头平稳推进，光影真实，无文字，无logo",
  "duration": 4,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "image_url": "https://example.com/source.png",
  "async": true
}
```

## 6. 首尾帧示例

请求体：

```json
{
  "model": "video-2.0-fast",
  "prompt": "图一自然过渡到图二，镜头平稳，动作自然，无文字，无logo",
  "duration": 4,
  "size": "720x1280",
  "start_image_url": "https://example.com/start.png",
  "end_image_url": "https://example.com/end.png",
  "async": true
}
```

## 7. 多图参考示例

请求体：

```json
{
  "model": "video-2.0-fast",
  "prompt": "结合两张参考图生成一段自然过渡的视频，主体动作自然，镜头平稳，无文字，无logo",
  "duration": 4,
  "size": "1280x720",
  "image_urls": [
    "https://example.com/image-1.png",
    "https://example.com/image-2.png"
  ],
  "async": true
}
```

## 8. 视频参考示例

请求体：

```json
{
  "model": "video-2.0-fast",
  "prompt": "广告视频",
  "duration": 4,
  "size": "720x1280",
  "video_url": "https://example.com/source.mp4",
  "async": true
}
```

## 9. 多视频参考示例

请求体：

```json
{
  "model": "video-2.0-fast",
  "prompt": "广告视频",
  "duration": 8,
  "size": "1280x720",
  "video_reference": [
    {
      "url": "https://example.com/video-1.mp4"
    },
    {
      "url": "https://example.com/video-2.mp4"
    }
  ],
  "async": true
}
```

## 10. 图片 + 视频混合参考示例

请求体：

```json
{
  "model": "video-2.0-fast",
  "prompt": "广告视频",
  "duration": 4,
  "size": "720x1280",
  "image_url": "https://example.com/source.png",
  "video_url": "https://example.com/source.mp4",
  "async": true
}
```

## 11. 多图 + 多视频混合参考示例

请求体：

```json
{
  "model": "video-2.0-fast",
  "prompt": "龟兔赛跑",
  "duration": 15,
  "size": "720x1280",
  "image_urls": [
    "https://example.com/image-1.png",
    "https://example.com/image-2.png"
  ],
  "video_reference": [
    {
      "url": "https://example.com/video-1.mp4"
    },
    {
      "url": "https://example.com/video-2.mp4"
    }
  ],
  "async": true
}
```

## 12. 提交响应

提交成功后会先返回异步任务信息：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "video-2.0",
  "status": "queued",
  "progress": 10,
  "created_at": 1777618428
}
```

下游必须保存 `task_id`，后续通过它查询任务结果。

## 13. 查询任务

处理中：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "video-2.0",
  "status": "queued",
  "progress": 10,
  "created_at": 1777618428
}
```

完成：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "video-2.0",
  "status": "completed",
  "url": "https://example.com/result.mp4",
  "progress": 100,
  "created_at": 1777618428,
  "completed_at": 1777618510
}
```

失败：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "video-2.0",
  "status": "failed",
  "progress": 100,
  "created_at": 1777618428,
  "completed_at": 1777618450,
  "error": {
    "message": "video generation failed",
    "code": "bad_response"
  }
}
```

补充说明：
- 部分任务结果里还可能附带 `seconds`、`size` 字段。
- 生成成功时优先读取顶层 `url`。

## 14. 状态说明

| status | 处理方式 |
| --- | --- |
| `submitted` | 继续轮询 |
| `queued` | 继续轮询 |
| `processing` | 继续轮询 |
| `in_progress` | 继续轮询 |
| `completed` | 读取 `url` |
| `failed` | 展示 `error.message` |

建议：
- 轮询间隔使用 `3-5` 秒。
- 最长轮询 `5-10` 分钟。
- 若长时间停留在 `queued`，通常表示上游仍在排队，不一定是请求失败。

## 15. cURL 示例

提交任务：

```bash
curl -X POST "https://你的域名/v1/video/async-generations" \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "video-2.0",
    "prompt": "一个卡通小狐狸在森林里奔跑，镜头跟随，动作流畅，无文字，无logo",
    "duration": 4,
    "aspect_ratio": "9:16",
    "resolution": "720p",
    "async": true
  }'
```

查询任务：

```bash
curl "https://你的域名/v1/video/async-generations/task_xxx" \
  -H "Authorization: Bearer sk-你的令牌"
```

## 16. JS 调用示例

```js
async function createSeedanceVideo() {
  const baseUrl = 'https://你的域名';
  const apiKey = 'sk-你的令牌';

  const submitRes = await fetch(`${baseUrl}/v1/video/async-generations`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      model: 'video-2.0',
      prompt: '一个小机器人在海边散步，镜头缓慢推进，无文字，无logo',
      duration: 4,
      aspect_ratio: '16:9',
      resolution: '720p',
      async: true,
    }),
  });

  if (!submitRes.ok) {
    throw new Error(`submit failed: ${submitRes.status}`);
  }

  const submitData = await submitRes.json();
  const taskId = submitData.task_id;
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

    if (pollData.status === 'completed' && pollData.url) {
      return pollData.url;
    }

    if (pollData.status === 'failed') {
      throw new Error(pollData.error?.message || 'video generation failed');
    }
  }

  throw new Error('poll timeout');
}
```

## 17. 推荐接入方式

- Web/H5 场景建议先提交任务，再在前端或服务端轮询结果。
- 如果需要更稳定的任务管理，建议下游自行落库保存 `task_id`、`status`、`url`。
- 如果同时支持文生、单图、首尾帧、多图，建议统一使用同一个接口，通过不同字段组合区分模式。
