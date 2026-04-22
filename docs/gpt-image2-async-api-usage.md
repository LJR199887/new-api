# GPT-Image2 异步图片 API 下游调用文档

本文档面向下游系统调用 `gpt-image2` 图片模型，统一使用异步任务模式。

适用模型：
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

接口：
| 用途 | 方法与路径 |
| --- | --- |
| 提交任务 | `POST /v1/images/async-generations` |
| 查询任务 | `GET /v1/images/async-generations/{task_id}` |

调用流程：提交任务 -> 获取 `task_id` -> 每 `3-5` 秒轮询 -> `completed` 后读取 `result_url` 或 `data[0].url`。

## 2. 提交参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 固定传 `gpt-image2` |
| `prompt` | string | 是 | 生成或编辑指令 |
| `output_resolution` | string | 否 | 固定为 `1K`；不传时服务端默认补 `1K` |
| `aspect_ratio` | string | 否 | 图片比例，支持 `1:1`、`16:9`、`9:16`、`4:3`、`3:4`、`3:2`、`2:3` |
| `messages` | array | 否 | 参考图输入。文生图不传；图生图或多图参考时传 OpenAI 多模态消息结构 |

说明：
- `gpt-image2` 的输出档位固定为 `1K`，不要传 `2K` 或 `4K`。
- 参考图最多支持 `6` 张。
- 图生图、多图参考图建议使用 `messages[].content[].image_url`，与上游 `gpt-image2` 调用保持一致。
- 兼容旧 JSON 调用时也可传 `image_urls` 或 `image`，服务端会自动转为 `messages`，但新接入建议直接使用 `messages`。

## 3. 文生图

不传参考图，只传 `prompt`。

```bash
curl https://linksky.top/v1/images/async-generations \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image2",
    "prompt": "一张未来城市夜景海报，霓虹灯，电影感，细节丰富",
    "output_resolution": "1K",
    "aspect_ratio": "1:1"
  }'
```

## 4. 图生图

传 1 张参考图到 `messages`。

```bash
curl https://linksky.top/v1/images/async-generations \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
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
  }'
```

## 5. 多图参考

传多张参考图到 `messages[].content`，数组顺序就是参考图顺序。建议在 `prompt` 中明确“图一”“图二”等用途。

```bash
curl https://linksky.top/v1/images/async-generations \
  -H "Authorization: Bearer sk-你的令牌" \
  -H "Content-Type: application/json" \
  -d '{
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
  }'
```

## 6. 响应格式

提交成功：
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

查询任务：
```bash
curl https://linksky.top/v1/images/async-generations/task_xxx \
  -H "Authorization: Bearer sk-你的令牌"
```

处理中：
```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "image.task",
  "model": "gpt-image2",
  "status": "in_progress",
  "progress": 30,
  "created_at": 1776843503
}
```

完成：
```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "image.task",
  "model": "gpt-image2",
  "status": "completed",
  "progress": 100,
  "created_at": 1776843503,
  "completed_at": 1776843536,
  "result_url": "https://example.com/result.png",
  "data": [
    {
      "url": "https://example.com/result.png",
      "presignedUrl": "",
      "presigned_url": "",
      "b64_json": "",
      "revised_prompt": ""
    }
  ]
}
```

失败：
```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "image.task",
  "model": "gpt-image2",
  "status": "failed",
  "progress": 100,
  "error": {
    "message": "image generation failed",
    "code": "bad_response"
  }
}
```

## 7. 状态处理

| status | 处理方式 |
| --- | --- |
| `queued` | 继续轮询 |
| `in_progress` | 继续轮询 |
| `completed` | 读取 `result_url`，没有则读取 `data[0].url` |
| `failed` | 展示 `error.message` |

## 8. 常见错误

### HTTP 状态码

| 状态码 | 含义 | 常见消息 |
| --- | --- | --- |
| `400` | 请求参数错误 | `model is required`、`prompt is required`、`output_resolution must be 1K for gpt-image2`、`aspect_ratio must be one of 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, or 2:3 for gpt-image2`、`gpt-image2 supports at most 6 uploaded images` |
| `401` | 认证失败 | `Token invalid or expired`、`Invalid API key`、`Unauthorized` |
| `404` | 资源不存在 | `task_not_exist`、`task not found` |
| `429` | 限额或配额问题 | `Token quota exhausted` |
| `500` | 服务内部错误 | 上游请求失败、结果解析失败、图片下载失败 |

### 任务接口错误格式

```json
{
  "code": "invalid_request",
  "message": "output_resolution must be 1K for gpt-image2",
  "data": null
}
```
