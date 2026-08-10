# video-2.5 / video-2.5-480p 异步视频 API 下游调用文档

本文档仅适用于以下两个模型：

- `video-2.5`：默认 `720p`，可通过 `resolution` 或 `size` 指定输出尺寸。
- `video-2.5-480p`：固定 `480p`，即使请求中传入其他分辨率，网关也会按 `480p` 调用上游。

两个模型的请求格式与 `video-2.0` 相同，统一使用异步任务接口。

## 1. 接入信息

Base URL：

```text
https://你的域名
```

认证请求头：

```http
Authorization: Bearer sk-你的令牌
Content-Type: application/json
```

| 用途 | 方法与路径 |
| --- | --- |
| 提交视频任务 | `POST /v1/video/async-generations` |
| 查询任务结果 | `GET /v1/video/async-generations/{task_id}` |

调用流程：提交任务 -> 保存 `task_id` -> 每 `3-5` 秒轮询 -> `completed` 后读取 `url`。

## 2. 素材限制

`video-2.5` 和 `video-2.5-480p` 的限制完全相同：

| 素材 | 数量限制 | 单个时长 | 总时长 |
| --- | --- | --- | --- |
| 图片 | 最多 `30` 张 | - | - |
| 视频 | 最多 `10` 个 | `3-10` 秒 | 不超过 `30` 秒 |
| 音频 | 最多 `10` 个 | `3-30` 秒 | 不超过 `30` 秒 |

网关会校验请求中显式传入的 `video_reference[].duration` 和 `audio_reference[].duration`。如果只提供远程 URL 而没有时长字段，实际素材时长由上游校验。

## 3. 提交请求参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | `video-2.5` 或 `video-2.5-480p` |
| `prompt` | string | 是 | 视频提示词，不能为空，最多 `5000` 字符 |
| `duration` | number | 否 | 生成视频时长，支持 `4-15` 秒，也用于按秒计费 |
| `aspect_ratio` | string | 否 | 输出比例：`9:16`、`16:9` 或 `1:1` |
| `resolution` | string | 否 | `video-2.5` 默认 `720p`；`video-2.5-480p` 固定 `480p` |
| `size` | string | 否 | 直接指定输出尺寸，与 `aspect_ratio` + `resolution` 二选一即可 |
| `image_url` | string | 否 | 单张参考图 URL |
| `image_urls` | string[] | 否 | 多张参考图 URL，最多 `30` 张 |
| `images` | string[] | 否 | `image_urls` 的兼容别名 |
| `start_image_url` | string | 否 | 首帧图片 URL |
| `end_image_url` | string | 否 | 尾帧图片 URL |
| `video_url` | string | 否 | 单个参考视频 URL |
| `video_reference` | object[] | 否 | 多视频参考，最多 `10` 个 |
| `audio_url` | string | 否 | 单个参考音频 URL |
| `audio_reference` | object[] | 否 | 多音频参考，最多 `10` 个 |
| `guidances.audio_reference` | object[] | 否 | 兼容 Leonardo Web 原始音频参考结构 |
| `async` | boolean | 否 | 建议固定为 `true` |

### 3.1 480p 尺寸映射

`video-2.5-480p` 可直接传 `aspect_ratio`，网关会生成对应的固定尺寸：

| 比例 | 输出尺寸 |
| --- | --- |
| `9:16` | `496x864` |
| `16:9` | `864x496` |
| `1:1` | `640x640` |

## 4. 文生视频

### 4.1 video-2.5（720p）

```json
{
  "model": "video-2.5",
  "prompt": "一只白色机器人在海边散步，镜头缓慢推进，电影感光影，无文字，无 logo",
  "duration": 10,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "async": true
}
```

### 4.2 video-2.5-480p

```json
{
  "model": "video-2.5-480p",
  "prompt": "一只白色机器人在海边散步，镜头缓慢推进，电影感光影，无文字，无 logo",
  "duration": 10,
  "aspect_ratio": "9:16",
  "resolution": "480p",
  "async": true
}
```

`video-2.5-480p` 也可直接传递 `"size": "496x864"`。

## 5. 图片参考示例

```json
{
  "model": "video-2.5",
  "prompt": "让参考图中的人物自然向前行走，镜头稳定跟拍",
  "duration": 8,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "image_urls": [
    "https://example.com/image-1.png",
    "https://example.com/image-2.png"
  ],
  "async": true
}
```

首尾帧模式：

```json
{
  "model": "video-2.5-480p",
  "prompt": "从日出自然过渡到黄昏，镜头稳定",
  "duration": 10,
  "aspect_ratio": "16:9",
  "start_image_url": "https://example.com/start.png",
  "end_image_url": "https://example.com/end.png",
  "async": true
}
```

## 6. 视频与音频素材示例

```json
{
  "model": "video-2.5",
  "prompt": "结合参考素材生成一段节奏自然的广告视频",
  "duration": 10,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "video_reference": [
    {
      "url": "https://example.com/video-1.mp4",
      "duration": 6
    },
    {
      "url": "https://example.com/video-2.mp4",
      "duration": 8
    }
  ],
  "audio_reference": [
    {
      "url": "https://example.com/music.mp3",
      "duration": 14
    }
  ],
  "async": true
}
```

也可使用已上传的上游音频 ID：

```json
{
  "model": "video-2.5-480p",
  "prompt": "让图片中的主体随音乐自然动起来",
  "duration": 8,
  "aspect_ratio": "1:1",
  "image_url": "https://example.com/source.png",
  "audio_reference": [
    {
      "id": "9be72770-3a31-4791-84bb-5047fc0d1fa9",
      "type": "UPLOADED",
      "duration": 14.9
    }
  ],
  "async": true
}
```

## 7. 提交响应

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "video-2.5",
  "status": "queued",
  "progress": 10,
  "created_at": 1777618428
}
```

下游必须保存 `task_id`，用于后续查询。

## 8. 查询任务

处理中：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "video-2.5",
  "status": "processing",
  "progress": 42,
  "created_at": 1777618428
}
```

完成：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "video-2.5",
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
  "model": "video-2.5",
  "status": "failed",
  "progress": 100,
  "error": {
    "message": "video generation failed",
    "code": "bad_response"
  }
}
```

| `status` | 下游处理方式 |
| --- | --- |
| `submitted` / `queued` / `processing` / `in_progress` | 继续轮询 |
| `completed` | 读取 `url` |
| `failed` | 读取并展示 `error.message` |

## 9. cURL 完整示例

提交：

```bash
curl -X POST "https://你的域名/v1/video/async-generations" \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "video-2.5-480p",
    "prompt": "一只卡通小狐狸在森林里奔跑，镜头跟随，动作流畅",
    "duration": 10,
    "aspect_ratio": "9:16",
    "resolution": "480p",
    "async": true
  }'
```

查询：

```bash
curl "https://你的域名/v1/video/async-generations/task_xxx" \
  -H "Authorization: Bearer sk-你的令牌"
```

## 10. JavaScript 轮询示例

```js
async function createVideo25(model = 'video-2.5') {
  const baseUrl = 'https://你的域名';
  const apiKey = 'sk-你的令牌';

  const submitResponse = await fetch(`${baseUrl}/v1/video/async-generations`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      model,
      prompt: '一个机器人在海边散步，镜头缓慢推进',
      duration: 10,
      aspect_ratio: '16:9',
      resolution: model.endsWith('-480p') ? '480p' : '720p',
      async: true,
    }),
  });

  if (!submitResponse.ok) {
    throw new Error(`submit failed: ${submitResponse.status}`);
  }

  const submitted = await submitResponse.json();
  if (!submitted.task_id) {
    throw new Error('missing task_id');
  }

  for (let attempt = 0; attempt < 120; attempt += 1) {
    await new Promise((resolve) => setTimeout(resolve, 3000));
    const pollResponse = await fetch(
      `${baseUrl}/v1/video/async-generations/${encodeURIComponent(submitted.task_id)}`,
      { headers: { Authorization: `Bearer ${apiKey}` } },
    );

    if (!pollResponse.ok) {
      throw new Error(`poll failed: ${pollResponse.status}`);
    }

    const task = await pollResponse.json();
    if (task.status === 'completed' && task.url) {
      return task.url;
    }
    if (task.status === 'failed') {
      throw new Error(task.error?.message || 'video generation failed');
    }
  }

  throw new Error('poll timeout');
}
```

## 11. 计费说明

两个模型均按生成时长计费。管理端为每个模型配置单秒价格，实际费用为：

```text
费用 = 单秒价格 × duration
```

例如单秒价格为 `0.3`，请求生成 `10` 秒视频，则费用为 `3`（未计其他分组倍率时）。
