# 异步媒体 API 下游调用文档示例

本文档将 `sora2`、`veo`、`ko3`、`kling-v3`、`grok-imagine-video`、`banana`、`gpt-image2` 的异步调用方式整理成一份统一示例，便于下游系统快速接入。

适用模型：

- 视频：
  - `sora2`
  - `veo31`
  - `veo31-fast`
  - `veo31-ref`
  - `ko3`
  - `kling-v3`
  - `grok-imagine-video`
- 图片：
  - `nano-banana`
  - `nano-banana2`
  - `nano-banana-pro`
  - `gpt-image2`

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

统一异步接口：

| 类型 | 用途 | 方法与路径 |
| --- | --- | --- |
| 视频 | 提交异步任务 | `POST /v1/video/async-generations` |
| 视频 | 查询异步任务 | `GET /v1/video/async-generations/{task_id}` |
| 图片 | 提交异步任务 | `POST /v1/images/async-generations` |
| 图片 | 查询异步任务 | `GET /v1/images/async-generations/{task_id}` |

统一调用流程：

1. 提交生成任务
2. 获取 `task_id`
3. 每 `3-5` 秒轮询一次
4. `completed` 后读取结果 URL

## 2. 模型与接口对应关系

| 模型 | 类型 | 提交接口 | 查询接口 |
| --- | --- | --- | --- |
| `sora2` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `veo31` / `veo31-fast` / `veo31-ref` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `ko3` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `kling-v3` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `grok-imagine-video` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `nano-banana*` | 图片 | `POST /v1/images/async-generations` | `GET /v1/images/async-generations/{task_id}` |
| `gpt-image2` | 图片 | `POST /v1/images/async-generations` | `GET /v1/images/async-generations/{task_id}` |

## 3. 视频模型调用示例

### 3.1 `sora2`

提交接口：

```bash
curl https://linksky.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sora2",
    "prompt": "一个美食节目镜头，厨师把意大利面装盘，暖色灯光，镜头平稳推进，无文字，无logo",
    "duration": 4,
    "aspect_ratio": "16:9",
    "async": true
  }'
```

图生视频：

```json
{
  "model": "sora2",
  "prompt": "让图片中的产品出现在一个高级广告视频里，镜头缓慢推进，光影自然，无文字，无logo",
  "duration": 4,
  "aspect_ratio": "16:9",
  "async": true,
  "image_url": "https://example.com/source-product.jpg"
}
```

说明：

- `prompt` 不能为空，且不能超过 `5000` 个字符。
- 不传 `image_url` 为文生视频，传 `image_url` 为图生视频。

提交参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 固定填写 `sora2` |
| `prompt` | string | 是 | 视频生成提示词，不能超过 `5000` 个字符 |
| `duration` | number | 否 | 视频时长，常用 `4`、`8`、`12`，单位秒 |
| `aspect_ratio` | string | 否 | 视频比例，常用 `16:9`、`9:16` |
| `async` | boolean | 否 | 建议传 `true` |
| `image_url` | string | 否 | 图生视频参考图 URL，不传则为文生视频 |

### 3.2 `veo31` / `veo31-fast` / `veo31-ref`

文生视频：

```bash
curl https://linksky.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "veo31-fast",
    "prompt": "A stylish product commercial shot, smooth dolly-in, cinematic lighting, no text, no logo",
    "duration": 4,
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "async": true
  }'
```

图生视频：

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

说明：

- `veo31` / `veo31-fast` 推荐 `reference_mode=frame`
- `veo31-ref` 推荐 `reference_mode=image`
- 不传 `image_url` 为文生视频

提交参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | `veo31`、`veo31-fast` 或 `veo31-ref` |
| `prompt` | string | 是 | 视频生成提示词 |
| `duration` | number | 否 | 视频时长，常用 `4`、`6`、`8`，单位秒 |
| `aspect_ratio` | string | 否 | 比例，常用 `16:9` / `9:16` |
| `resolution` | string | 否 | 分辨率档位，如 `720p` / `1080p` |
| `reference_mode` | string | 否 | 参考图模式，`frame` 或 `image` |
| `async` | boolean | 否 | 建议传 `true` |
| `image_url` | string | 否 | 图生视频参考图 URL，不传则为文生视频 |

### 3.3 `ko3`

文生视频：
```bash
curl https://linksky.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "ko3",
    "prompt": "龟兔赛跑，电影级运动镜头，真实光影，动作自然",
    "duration": 3,
    "size": "1080x1920"
  }'
```

单图生视频：
```json
{
  "model": "ko3",
  "prompt": "猫咪跳舞，镜头轻微推进，动作自然",
  "duration": 3,
  "size": "1080x1920",
  "image_url": "https://example.com/cat.png"
}
```

多图生视频：
```json
{
  "model": "ko3",
  "prompt": "动物世界，将多张参考图中的主体组合成自然运动的视频",
  "duration": 3,
  "size": "1080x1920",
  "image_urls": [
    "https://example.com/a.png",
    "https://example.com/b.png",
    "https://example.com/c.png"
  ]
}
```

首尾帧：
```json
{
  "model": "ko3",
  "prompt": "从图一过渡到图二，镜头平滑推进，自然转场",
  "duration": 3,
  "size": "1080x1920",
  "start_image_url": "https://example.com/start.png",
  "end_image_url": "https://example.com/end.png"
}
```

视频生视频：
```json
{
  "model": "ko3",
  "prompt": "把视频中的香水替换成牙膏，保持原视频运动节奏",
  "video_url": "https://example.com/source.mp4"
}
```

图片 + 视频生视频：
```json
{
  "model": "ko3",
  "prompt": "把视频中的香水替换成图片里的小熊，保持镜头运动",
  "image_url": "https://example.com/bear.png",
  "video_url": "https://example.com/source.mp4"
}
```

多图 + 视频生视频：
```json
{
  "model": "ko3",
  "prompt": "用多张图片替换视频主体，保持视频中的运动和构图",
  "image_urls": [
    "https://example.com/a.png",
    "https://example.com/b.png"
  ],
  "video_url": "https://example.com/source.mp4"
}
```

说明：
- `ko3` 下游统一使用 `POST /v1/video/generations`，对外仍调用本站 `POST /v1/video/async-generations`。
- 兼容别名：`kling-o3`、`kling-video-o-3`，新接入建议使用 `ko3`。
- `size` 支持 `1440x1440`、`1080x1920`、`1920x1080`，分别对应 `1:1`、`9:16`、`16:9`。
- 文生视频、单图生视频、多图生视频、首尾帧模式可传 `duration` 和 `size`；`duration` 支持 `3-15` 秒。
- 图片大小不能超过 `5MB`；多图模式最多上传 `7` 张图。
- 视频只能上传 `1` 个，视频时长需在 `3-10` 秒内，大小不能超过 `200MB`。
- 多图 + 视频模式下，图片最多上传 `4` 张。
- 视频参考、图片 + 视频、多图 + 视频模式不需要传 `duration` 和 `size`，默认 `9:16`，时长根据视频素材决定。

提交参数：
| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 固定传 `ko3` |
| `prompt` | string | 是 | 视频提示词 |
| `duration` | number | 否 | 文生/图生/首尾帧可传，允许 `3-15` 秒 |
| `size` | string | 否 | `1440x1440` / `1080x1920` / `1920x1080` |
| `image_url` | string | 否 | 单图参考图 URL |
| `image_urls` | array | 否 | 多图参考图 URL；纯多图最多 `7` 张，多图 + 视频最多 `4` 张 |
| `start_image_url` | string | 否 | 首帧图 URL |
| `end_image_url` | string | 否 | 尾帧图 URL |
| `video_url` | string | 否 | 参考视频 URL，仅支持 `1` 个视频 |

完整文档见：[`docs/ko3-async-api-usage.md`](./ko3-async-api-usage.md)

### 3.4 `kling-v3`

文生视频：

```bash
curl https://linksky.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
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

单图图生视频：

```json
{
  "model": "kling-v3",
  "prompt": "让画面中的角色向镜头走来，电影级运镜，环境音真实",
  "duration": 15,
  "aspect_ratio": "9:16",
  "generate_audio": true,
  "generateAudio": true,
  "async": true,
  "image_url": "https://example.com/character.png"
}
```

首尾帧双图：

```json
{
  "model": "kling-v3",
  "prompt": "让角色从第一张图自然运动到第二张图，镜头平滑推进，电影级运镜",
  "duration": 8,
  "aspect_ratio": "9:16",
  "generate_audio": true,
  "generateAudio": true,
  "async": true,
  "image_urls": [
    "https://example.com/first-frame.png",
    "https://example.com/last-frame.png"
  ]
}
```

说明：

- `duration` 允许 `3-15` 秒，默认 `5` 秒。
- `aspect_ratio` 支持 `16:9` / `9:16`。
- 不传图片为文生视频；传 `image_url` 为单图图生视频。
- 首尾帧双图请使用 `image_urls`，最多取前 `2` 张。
- `images`、`image`、`input_reference`、`image_reference` 为兼容字段，服务端会自动转换。
- `kling-v3` 按次计费，`duration` 不作为按秒倍率扣费。

提交参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 固定传 `kling-v3` |
| `prompt` | string | 是 | 视频提示词 |
| `duration` | number | 否 | 视频时长，允许 `3-15` 秒 |
| `aspect_ratio` | string | 否 | `16:9` 或 `9:16` |
| `generate_audio` | boolean | 否 | 是否生成音频，默认 `true` |
| `generateAudio` | boolean | 否 | 音频开关兼容字段，建议与 `generate_audio` 同时传 `true` |
| `image_url` | string | 否 | 单图参考图 URL |
| `image_urls` | array | 否 | 首尾帧参考图 URL，最多前 `2` 张 |
| `async` | boolean | 否 | 建议传 `true` |

### 3.5 `grok-imagine-video`

文生视频：

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

图生视频，推荐 `multipart/form-data`：

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

如果下游只能传 JSON，也可以传图片 URL：

```json
{
  "model": "grok-imagine-video",
  "prompt": "让参考图里的主体向镜头走来，电影感运镜",
  "seconds": 10,
  "size": "720x1280",
  "resolution_name": "720p",
  "preset": "normal",
  "image_reference": [
    "https://example.com/reference.png"
  ],
  "async": true
}
```

说明：

- `seconds` 只允许 `6` 或 `10`
- `duration` 兼容转换为 `seconds`
- 图生视频最稳妥的方式是直接传 multipart 文件

提交参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 固定传 `grok-imagine-video` |
| `prompt` | string | 是 | 视频提示词 |
| `seconds` | number/string | 否 | 视频长度，只允许 `6` 或 `10`，默认 `10` |
| `duration` | number/string | 否 | 兼容字段，会转为 `seconds` |
| `size` | string | 否 | 如 `720x1280`、`1280x720`、`1024x1024`、`1792x1024` |
| `resolution_name` | string | 否 | `480p` 或 `720p` |
| `preset` | string | 否 | `fun`、`normal`、`spicy`、`custom` |
| `input_reference[]` | file | 否 | 图生视频参考图，multipart 文件字段 |
| `image_reference` | array | 否 | JSON 模式下的参考图 URL 数组 |
| `async` | boolean | 否 | 建议传 `true` |

## 4. 图片模型调用示例

### 4.1 Banana 系列：`nano-banana` / `nano-banana2` / `nano-banana-pro`

文生图：

```bash
curl https://linksky.top/v1/images/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "nano-banana-pro",
    "prompt": "高端牙膏产品广告海报，银色包装，商业摄影，干净高级，柔和棚拍光，背景极简",
    "output_resolution": "2K",
    "aspect_ratio": "16:9"
  }'
```

图生图：

```json
{
  "model": "nano-banana-pro",
  "prompt": "保留原图主体和构图，将画面优化为高端商业广告风格，增强包装质感和边缘高光，不要改变产品形状，不要增加额外文字",
  "output_resolution": "2K",
  "aspect_ratio": "1:1",
  "image_urls": [
    "https://example.com/source-product.png"
  ]
}
```

多图编辑：

```json
{
  "model": "nano-banana-pro",
  "prompt": "使用图一作为主体产品，使用图二中的 logo，将 logo 自然贴合到产品正面，保持透视、弧度、光影和反光一致，像真实印刷在包装上，不要额外文字",
  "output_resolution": "2K",
  "aspect_ratio": "16:9",
  "image_urls": [
    "https://example.com/product.png",
    "https://example.com/logo.png"
  ]
}
```

说明：

- 不传 `image_urls` 为文生图
- 传 `1` 张图为图生图
- 传多张图为多图编辑
- 最多输入 `6` 张图

提交参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | Banana 系列模型名，如 `nano-banana-pro` |
| `prompt` | string | 是 | 生成或编辑指令 |
| `output_resolution` | string | 否 | 输出档位，建议 `1K`、`2K`、`4K` |
| `aspect_ratio` | string | 否 | 图片比例，如 `1:1`、`16:9`、`9:16`、`4:3`、`3:4` |
| `image_urls` | array | 否 | 参考图 URL 数组，最多 `6` 张 |

### 4.2 `gpt-image2`

文生图：

```bash
curl https://linksky.top/v1/images/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image2",
    "prompt": "一张未来城市夜景海报，霓虹灯，电影感，细节丰富",
    "output_resolution": "1K",
    "aspect_ratio": "1:1"
  }'
```

图生图：

```json
{
  "model": "gpt-image2",
  "prompt": "把参考图改成赛博朋克海报风格，保持主体构图，电影级光影",
  "output_resolution": "1K",
  "aspect_ratio": "1:1",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "把参考图改成赛博朋克海报风格，保持主体构图，电影级光影"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/source.png"
          }
        }
      ]
    }
  ]
}
```

多图参考：

```json
{
  "model": "gpt-image2",
  "prompt": "使用图一作为人物主体，参考图二的服装风格，生成一张商业大片海报，保持真实摄影质感",
  "output_resolution": "1K",
  "aspect_ratio": "16:9",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "使用图一作为人物主体，参考图二的服装风格，生成一张商业大片海报，保持真实摄影质感"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/person.png"
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/style.png"
          }
        }
      ]
    }
  ]
}
```

说明：

- `output_resolution` 固定为 `1K`
- 参考图最多支持 `6` 张
- 新接入建议直接使用 `messages[].content[].image_url`

提交参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 固定传 `gpt-image2` |
| `prompt` | string | 是 | 生成或编辑指令 |
| `output_resolution` | string | 否 | 固定为 `1K`；不传时服务端默认补 `1K` |
| `aspect_ratio` | string | 否 | 支持 `1:1`、`16:9`、`9:16`、`4:3`、`3:4`、`3:2`、`2:3` |
| `messages` | array | 否 | 参考图输入，推荐用 OpenAI 多模态消息结构 |
| `image_urls` | array | 否 | 兼容旧调用，服务端会自动转为 `messages` |
| `image` | string | 否 | 兼容旧调用，服务端会自动转为 `messages` |

## 5. 提交成功响应示例

### 5.1 视频任务

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "sora2",
  "status": "queued",
  "progress": 10,
  "created_at": 1776418394
}
```

### 5.2 图片任务

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "image.task",
  "model": "gpt-image2",
  "status": "queued",
  "progress": 10,
  "created_at": 1776843503
}
```

下游拿到响应后，保存 `task_id`，后续轮询任务结果。

## 6. 查询任务示例

### 6.1 查询视频任务

```bash
curl https://linksky.top/v1/video/async-generations/task_xxx \
  -H "Authorization: Bearer $LINKSKY_API_KEY"
```

### 6.2 查询图片任务

```bash
curl https://linksky.top/v1/images/async-generations/task_xxx \
  -H "Authorization: Bearer $LINKSKY_API_KEY"
```

处理中：

```json
{
  "task_id": "task_xxx",
  "status": "in_progress",
  "progress": 30
}
```

完成：

```json
{
  "task_id": "task_xxx",
  "status": "completed",
  "progress": 100,
  "result_url": "https://example.com/result.png",
  "video_url": "https://example.com/result.mp4",
  "url": "https://example.com/result.mp4",
  "data": [
    {
      "url": "https://example.com/result.png"
    }
  ]
}
```

失败：

```json
{
  "task_id": "task_xxx",
  "status": "failed",
  "progress": 100,
  "error": {
    "message": "generation failed",
    "code": "bad_response"
  }
}
```

结果读取建议：

- 视频：优先读 `video_url`，没有则读 `url`，再没有则读 `data[0].url`
- 图片：优先读 `result_url`，没有则读 `data[0].url`

## 7. 通用下游 JS 轮询示例

### 7.1 视频任务

```js
async function pollVideoTask(baseUrl, apiKey, taskId) {
  for (let i = 0; i < 120; i += 1) {
    await new Promise((resolve) => setTimeout(resolve, 3000));

    const res = await fetch(
      `${baseUrl}/v1/video/async-generations/${encodeURIComponent(taskId)}`,
      {
        headers: {
          Authorization: `Bearer ${apiKey}`,
        },
      },
    );

    if (!res.ok) {
      throw new Error(`poll failed: ${res.status}`);
    }

    const data = await res.json();
    const payload =
      data.data && !Array.isArray(data.data) ? data.data : data;
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

### 7.2 图片任务

```js
async function pollImageTask(baseUrl, apiKey, taskId) {
  for (let i = 0; i < 120; i += 1) {
    await new Promise((resolve) => setTimeout(resolve, 3000));

    const res = await fetch(
      `${baseUrl}/v1/images/async-generations/${encodeURIComponent(taskId)}`,
      {
        headers: {
          Authorization: `Bearer ${apiKey}`,
        },
      },
    );

    if (!res.ok) {
      throw new Error(`poll failed: ${res.status}`);
    }

    const data = await res.json();
    const payload =
      data.data && !Array.isArray(data.data) ? data.data : data;
    const imageUrl = payload.result_url || payload.data?.[0]?.url || '';

    if (payload.status === 'completed' && imageUrl) {
      return imageUrl;
    }

    if (payload.status === 'failed') {
      throw new Error(payload.error?.message || payload.error || 'image generation failed');
    }
  }

  throw new Error('poll timeout');
}
```

## 8. 接入建议

- 轮询间隔建议 `3-5` 秒
- 最长可轮询 `5-10` 分钟
- 图片 URL 参考图必须公网可访问
- `grok-imagine-video` 图生视频推荐直接传 multipart 文件
- `gpt-image2` 固定使用 `1K` 输出
- `sora2` 的 `prompt` 不能超过 `5000` 个字符

## 9. 常见错误码

### 9.1 通用 HTTP 状态码

| 状态码 | 含义 | 常见消息 |
| --- | --- | --- |
| `400` | 请求参数错误 | 参数缺失、字段格式错误、模型参数不支持 |
| `401` | 认证失败 | `Token invalid or expired`、`Invalid API key`、`Unauthorized` |
| `403` | 无模型或分组访问权限 | `token has no access to model`、`group access denied` |
| `404` | 任务或资源不存在 | `task_not_exist`、`task not found`、`file not found` |
| `413` | 请求体过大 | `request body too large` |
| `429` | 限额或频控 | `Token quota exhausted`、`Too many requests` |
| `500` | 服务内部错误 | 上游失败、结果解析失败、轮询失败 |
| `503` | 没有可用渠道或上游暂时不可用 | `No available channel for model ...` |

### 9.2 `sora2`

| 状态码 | 常见消息 |
| --- | --- |
| `400` | `model is required`、`messages or prompt is required`、`prompt must contain at least 3 characters`、`prompt is required`、`prompt is too long`、`unsupported duration`、`unsupported aspect_ratio`、`unsupported resolution`、`unsupported reference_mode`、`Invalid video model`、`invalid request` |
| `500` | `video submit failed`、`video poll failed`、`video generation timed out`、`Unhandled error` |
| `503` | `No available channel for model sora2 under group xxx` |

上游语义错误：

| error_code | 含义 |
| --- | --- |
| `video_unsafe` | 内容安全拦截，建议修改 `prompt` 后重试 |

### 9.3 `veo31` / `veo31-fast` / `veo31-ref`

| 状态码 | 常见消息 |
| --- | --- |
| `400` | `unsupported duration`、`unsupported aspect_ratio`、`unsupported resolution`、`unsupported reference_mode`、`Invalid video model` |
| `401` | `Token invalid or expired`、`Invalid API key` |
| `404` | `task_not_exist`、`video generation not found`、`task not found` |
| `500` | `video submit failed`、`video poll failed`、结果解析失败 |

### 9.4 `grok-imagine-video`

| 状态码 | 常见消息 |
| --- | --- |
| `400` | `prompt is required`、`unsupported duration`、`unsupported resolution`、`Invalid video model` |
| `401` | `Token invalid or expired`、`Invalid API key` |
| `404` | `task not found`、`file not found` |
| `500` | `imgbed source download failed`、`video generation failed` |

### 9.5 Banana 系列

| 状态码 | 常见消息 |
| --- | --- |
| `400` | `prompt is required`、`messages or prompt is required`、`prompt must contain at least 3 characters`、`unsupported duration`、`unsupported resolution`、`unsupported aspect_ratio`、`unsupported reference_mode`、`unsupported output_resolution`、`Use /v1/chat/completions for video generation`、`Invalid video mode` |
| `401` | `Token invalid or expired`、`Invalid API key` |
| `404` | `task not found`、`file not found`、`error code not found` |
| `429` | `Token quota exhausted` |
| `500` | `imgbed source download failed`、轮询失败 |

任务接口错误格式示例：

```json
{
  "code": "invalid_request",
  "message": "field prompt is required",
  "data": null
}
```

### 9.6 `gpt-image2`

| 状态码 | 常见消息 |
| --- | --- |
| `400` | `model is required`、`prompt is required`、`output_resolution must be 1K for gpt-image2`、`aspect_ratio must be one of 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, or 2:3 for gpt-image2`、`gpt-image2 supports at most 6 uploaded images` |
| `401` | `Token invalid or expired`、`Invalid API key` |
| `404` | `task_not_exist`、`task not found` |
| `429` | `Token quota exhausted` |
| `500` | 上游请求失败、结果解析失败、图片下载失败 |

任务接口错误格式示例：

```json
{
  "code": "invalid_request",
  "message": "output_resolution must be 1K for gpt-image2",
  "data": null
}
```

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
