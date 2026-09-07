# 933 Video 2.0 系列异步视频 API 下游对接文档

本文档面向调用 new-api 公开 API 的下游客户端，适用于以下四个模型：

- `933-video2.0`
- `933-video2.0-480p`
- `933-video2.0-mini`
- `933-video2.0-mini-480p`

四个模型均通过异步任务生成视频，按生成时长计费。

> 版本要求：若使用 fa2api 作为上游，需部署已接通全量参考素材的版本（至少包含 fa2api commit `04a3054`）。旧版只支持 4 秒 480p 图片参考，会返回 `image-reference video currently requires...` 或音视频参考 `501` 错误。

## 1. 接入信息

```text
Base URL: https://你的网关域名
提交任务: POST /v1/video/generations
查询任务: GET  /v1/video/generations/{task_id}
```

所有请求都必须携带 new-api 下游 API Key：

```http
Authorization: Bearer sk-你的令牌
Content-Type: application/json
```

基本流程：

1. `POST /v1/video/generations` 提交生成任务。
2. 保存响应中的 `task_id`。
3. 每 3–5 秒调用查询接口。
4. `status=completed` 时读取结果 URL；`status=failed` 时停止轮询并记录 `error`。

## 2. 模型与分辨率

| 模型 | 类型 | 固定分辨率 | 生成时长 | 默认时长 |
| --- | --- | ---: | ---: | ---: |
| `933-video2.0` | Standard | 720p | 4–15 秒整数 | 5 秒 |
| `933-video2.0-480p` | Standard | 480p | 4–15 秒整数 | 5 秒 |
| `933-video2.0-mini` | Mini | 720p | 4–15 秒整数 | 5 秒 |
| `933-video2.0-mini-480p` | Mini | 480p | 4–15 秒整数 | 5 秒 |

分辨率由模型名固定。推荐省略 `resolution`；若显式传入，必须与模型匹配，否则返回 `400`，网关不会静默降级。

## 3. 请求参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 上述四个模型之一 |
| `prompt` | string | 是 | 视频提示词，建议至少 3 个字符 |
| `duration` | integer | 否 | 生成视频时长，4–15，默认 5 |
| `seconds` | integer/string | 否 | `duration` 的兼容别名，不要与 `duration` 传不同值 |
| `aspect_ratio` | string | 否 | 默认 `16:9`，可选值见下文 |
| `resolution` | string | 否 | 只能传模型对应的 `720p` 或 `480p` |
| `image_url` | string | 否 | 单张参考图片 URL |
| `image_urls` | string[] | 否 | 多图参考，最多 9 张 |
| `start_image_url` | string | 否 | 首帧图片，必须与 `end_image_url` 同时传入 |
| `end_image_url` | string | 否 | 尾帧图片，必须与 `start_image_url` 同时传入 |
| `video_url` | string | 否 | 单个参考视频 URL |
| `video_urls` | string[] | 否 | 多视频参考，最多 3 个 |
| `video_reference` | object[] | 否 | 对象形式的视频参考；推荐只传 `url`，`duration` 仅为可选校验提示 |
| `audio_url` | string | 否 | 单个参考音频 URL |
| `audio_urls` | string[] | 否 | 多音频参考，最多 3 个 |
| `audio_reference` | object[] | 否 | 对象形式的音频参考；推荐只传 `url`，`duration` 仅为可选校验提示 |
| `n` / `count` | integer | 否 | 只支持 `1` |

`aspect_ratio` 支持：

```text
16:9, 9:16, 4:3, 3:4, 1:1, 21:9
```

## 4. 参考模式与素材限制

网关会根据所传素材自动判断模式，下游通常不需要传 `generation_mode` 或 `reference_mode`。

| 模式 | 请求特征 | 关键规则 |
| --- | --- | --- |
| 文生视频 | 不传任何参考素材 | 只需 `prompt` |
| 多图参考 | 传 `image_url` 或 `image_urls` | 最多 9 张 |
| 首尾帧 | 同时传 `start_image_url` 和 `end_image_url` | 必须各 1 张，不能混入其他参考素材 |
| 视频参考 | 传 `video_url` / `video_urls` / `video_reference` | 最多 3 个 |
| 多模态 | 图片、视频、音频组合 | 音频必须搭配至少 1 个图片或视频 |

### 4.1 素材上限

| 素材 | 数量 | 单文件大小 | 单文件时长 | 同类总时长 | 建议格式 |
| --- | ---: | ---: | ---: | ---: | --- |
| 图片 | 最多 9 张 | 小于 20 MiB | - | - | PNG、JPG、WebP |
| 视频 | 最多 3 个 | 小于 200 MiB | 2–15 秒 | 不超过 15 秒 | MP4、MOV |
| 音频 | 最多 3 个 | 小于 15 MiB | 2–15 秒 | 不超过 15 秒 | MP3、WAV |

注意：

- 视频总时长和音频总时长分别统计。
- 大小校验以实际下载/解码后的字节数为准。为兼容所有上游版本，不要让文件大小恰好等于上限。
- 推荐下游只提交 URL，不提交参考素材的 `duration`。对象形式中的 `duration` 仅为可选校验提示，服务端仍会探测实际时长。
- 素材必须使用上游能访问的 HTTP(S) URL。不能传本地路径、浏览器 `blob:` URL、带认证才能访问的临时链接。
- 同一类素材不要同时传单数、数组和 `*_reference` 别名，否则会当作冲突请求。

## 5. 请求示例

### 5.1 文生视频

```bash
curl -X POST "https://你的网关域名/v1/video/generations" \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "933-video2.0",
    "prompt": "无人机穿越云海与雪山，电影感运镜",
    "duration": 8,
    "aspect_ratio": "16:9"
  }'
```

### 5.2 多图参考

```json
{
  "model": "933-video2.0-mini",
  "prompt": "保持角色外观，让两个角色在森林小路上并排行走",
  "duration": 6,
  "aspect_ratio": "9:16",
  "image_urls": [
    "https://example.com/character-a.png",
    "https://example.com/character-b.png",
    "https://example.com/scene.png"
  ]
}
```

### 5.3 首尾帧

```json
{
  "model": "933-video2.0-480p",
  "prompt": "画面从清晨自然过渡到黄昏",
  "duration": 5,
  "aspect_ratio": "16:9",
  "start_image_url": "https://example.com/start.png",
  "end_image_url": "https://example.com/end.png"
}
```

### 5.4 多模态参考

```json
{
  "model": "933-video2.0-mini-480p",
  "prompt": "保持参考角色外观，参考运镜、动作节奏和音乐节拍生成广告短片",
  "duration": 5,
  "aspect_ratio": "9:16",
  "image_urls": [
    "https://example.com/character.png"
  ],
  "video_urls": [
    "https://example.com/motion.mp4"
  ],
  "audio_urls": [
    "https://example.com/music.mp3"
  ]
}
```

推荐使用 `video_urls` 和 `audio_urls`，请求里只提交素材 URL。服务端会自行探测参考素材时长；顶层 `duration` 必须保留，它是输出视频时长和计费时长。

## 6. 提交响应

通过 new-api 提交成功通常返回 HTTP `200 OK`（上游自身可能使用 `202 Accepted`，下游不应依赖上游状态码）：

```json
{
  "id": "task_public_xxx",
  "task_id": "task_public_xxx",
  "object": "video.generation",
  "model": "933-video2.0-mini-480p",
  "status": "queued",
  "progress": 0,
  "created": 1780000000
}
```

`task_id` 是 new-api 对外公开任务 ID。查询时不要换成日志中的上游 ID。

收到 `202` 仅表示任务已接收，不代表生成已成功。

## 7. 查询任务

```bash
curl "https://你的网关域名/v1/video/generations/task_public_xxx" \
  -H "Authorization: Bearer sk-你的令牌"
```

生成中示例：

```json
{
  "id": "task_public_xxx",
  "task_id": "task_public_xxx",
  "object": "video.generation",
  "model": "933-video2.0-mini-480p",
  "status": "running",
  "progress": 30
}
```

完成示例：

```json
{
  "id": "task_public_xxx",
  "task_id": "task_public_xxx",
  "object": "video.generation",
  "model": "933-video2.0-mini-480p",
  "status": "completed",
  "progress": 100,
  "url": "https://example.com/generated-video.mp4"
}
```

失败示例：

```json
{
  "id": "task_public_xxx",
  "task_id": "task_public_xxx",
  "object": "video.generation",
  "model": "933-video2.0-mini-480p",
  "status": "failed",
  "error": "video generation failed"
}
```

状态可能为 `submitted`、`queued`、`running`、`processing`、`in_progress`、`completed` 或 `failed`。下游应将前五种视为未完成状态。

结果链接可能位于 `url`、`video_url`、`result_url` 或 `data[0].url`，客户端建议做兼容读取。

## 8. JavaScript 对接示例

```js
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function generate933Video({ baseUrl, apiKey, body }) {
  const submitResponse = await fetch(`${baseUrl}/v1/video/generations`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  });
  const submitted = await submitResponse.json();
  if (!submitResponse.ok) {
    throw new Error(
      submitted?.error?.message || submitted?.detail || JSON.stringify(submitted),
    );
  }

  const taskId = submitted.task_id || submitted.id;
  if (!taskId) throw new Error('submit response does not contain task_id');

  for (let attempt = 0; attempt < 240; attempt += 1) {
    await sleep(4000);
    const pollResponse = await fetch(
      `${baseUrl}/v1/video/generations/${encodeURIComponent(taskId)}`,
      { headers: { Authorization: `Bearer ${apiKey}` } },
    );
    const task = await pollResponse.json();
    if (!pollResponse.ok) {
      throw new Error(task?.error?.message || task?.detail || 'task query failed');
    }

    const status = String(task.status || '').toLowerCase();
    if (status === 'completed' || status === 'succeeded' || status === 'success') {
      const url =
        task.url || task.video_url || task.result_url || task.data?.[0]?.url;
      if (!url) throw new Error('completed task does not contain a video URL');
      return { taskId, url, task };
    }
    if (status === 'failed' || status === 'failure' || status === 'error') {
      throw new Error(task?.error?.message || task?.error || 'video generation failed');
    }
  }
  throw new Error('video generation polling timed out');
}
```

调用示例：

```js
const result = await generate933Video({
  baseUrl: 'https://你的网关域名',
  apiKey: 'sk-你的令牌',
  body: {
    model: '933-video2.0-mini',
    prompt: '镜头穿过需弥漫的未来城市',
    duration: 8,
    aspect_ratio: '16:9',
  },
});

console.log(result.taskId, result.url);
```

## 9. 计费

四个模型必须在 new-api 中配置「按时长」单价。

```text
费用 = 模型每秒价格 × 生成时长 × 分组倍率
```

- 计费使用顶层 `duration`，不使用参考音视频时长。
- 参考图片、视频和音频数量不直接乘入 new-api 的按秒费用。
- 下游不要依赖 Fish 积分、上游报价版本或内部 Token 信息。
- 未配置每秒价格时，网关会拒绝提交，不回退到按次或 Token 计费。

## 10. 常见错误

| HTTP/错误 | 原因 | 处理方式 |
| --- | --- | --- |
| `400 unsupported duration` | `duration` 不是 4–15 范围的整数 | 修正输出时长 |
| `400 unsupported resolution` | 显式分辨率与模型名不匹配 | 省略 `resolution` 或更换模型 |
| `400 unsupported aspect_ratio` | 画面比例不在支持列表 | 使用文档列出的比例 |
| `frame mode requires exactly one start and one end image` | 首尾帧缺失或数量不对 | 各传 1 张图片 |
| `frame mode cannot be combined with references` | 首尾帧与普通图片/音视频混用 | 只保留首尾帧 |
| `supports at most 9 image references` | 参考图片超限 | 减少到 9 张以内 |
| `supports at most 3 video/audio references` | 参考音视频超限 | 每类减少到 3 个以内 |
| `duration must be between 2 and 15 seconds` | 参考音视频时长超限 | 修正参考素材 |
| `total ... reference duration must not exceed 15 seconds` | 同类参考素材总时长超限 | 裁剪或减少素材 |
| `audio reference requires at least one image or video reference` | 只传了音频 | 至少添加 1 张图片或 1 个视频 |
| `image-reference video currently requires a standard/Mini -480p model and duration=4` | fa2api 服务仍是旧版 | 紧急规避可改用 `-480p` + 4 秒；正式处理是升级 fa2api |
| `video/audio reference media inspection protocol is not configured` | fa2api 未部署音视频参考实现 | 升级至包含 `04a3054` 的版本 |
| `401` | API Key 无效 | 检查 `Authorization` 请求头 |
| `402` | 上游账号余额不足 | 更换有余额账号或充值 |
| `429` | 请求限流 | 按指数退避重试，不要立即高频重发 |

## 11. 下游实现注意事项

- 提交成功后必须持久化 `task_id`、模型、提示词和提交时间。
- 查询请求建议每 3–5 秒一次，不要高频轮询。
- 任务进入上游后，不要因查询超时就自动重新提交，否则可能重复扣费。
- 结果 URL 可能存在有效期，业务需长期保存时应在任务完成后及时转存。
- 对 `5xx`、`429` 可以做有界退避；对参数类 `400` 应直接修改请求，不要原样重试。
- 不要把上游 Fish Token、Team ID、Cookie、预签名上传 URL 暴露给下游或记录在公开日志中。
