# minimax-h3 异步视频 API 下游调用文档

`minimax-h3` 统一通过异步视频接口调用，并按次计费。管理员需要在模型价格设置中配置单次价格；生成时长不会作为额外倍率参与扣费。

## 接口

```text
POST /v1/video/async-generations
GET  /v1/video/async-generations/{task_id}
```

也兼容：

```text
POST /v1/video/generations
GET  /v1/video/generations/{task_id}
```

## 基本参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 固定为 `minimax-h3` |
| `prompt` | string | 是 | 视频提示词 |
| `duration` | integer | 否 | 5–15 秒，默认 5 秒 |
| `aspect_ratio` | string | 否 | `16:9`、`9:16`、`1:1`、`4:3`、`3:4`、`21:9` |
| `size` | string | 否 | 显式输出尺寸（宽×高），优先级高于 `aspect_ratio` |

MiniMax H3 默认使用 2K，`aspect_ratio` 与上游尺寸的对应关系如下：

| `aspect_ratio` | `size` | 上游参数 |
| --- | --- | --- |
| `16:9` | `2560x1440` | `width=2560, height=1440` |
| `9:16` | `1440x2560` | `width=1440, height=2560` |
| `1:1` | `1440x1440` | `width=1440, height=1440` |
| `4:3` | `1920x1440` | `width=1920, height=1440` |
| `3:4` | `1440x1920` | `width=1440, height=1920` |
| `21:9` | `3360x1440` | `width=3360, height=1440` |

## 文生视频

```json
{
  "model": "minimax-h3",
  "prompt": "龟兔赛跑",
  "duration": 5,
  "aspect_ratio": "16:9"
}
```

## 多图参考

最多支持 5 张图片。单图可传 `image_url`，多图使用 `image_urls`。

```json
{
  "model": "minimax-h3",
  "prompt": "图一和图二的角色出现在图三的场景中",
  "duration": 8,
  "image_urls": [
    "https://example.com/character-1.png",
    "https://example.com/character-2.png",
    "https://example.com/scene.png"
  ]
}
```

## 首尾帧

首尾帧模式不能和图片参考模式混用，也不支持音频参考。

```json
{
  "model": "minimax-h3",
  "prompt": "恐龙逐渐变成兔子",
  "duration": 5,
  "start_image_url": "https://example.com/start.png",
  "end_image_url": "https://example.com/end.png"
}
```

## 图片加音频

音频参考必须同时提供至少一张图片参考。

```json
{
  "model": "minimax-h3",
  "prompt": "恐龙冒险",
  "duration": 5,
  "image_url": "https://example.com/dinosaur.png",
  "audio_url": "https://example.com/adventure.mp3"
}
```

## 限制

- 不接受 `h3` 或 `hailuo-03` 作为下游模型名。
- 不支持视频参考。
- 图片参考最多 5 张。
- 图片参考模式和首尾帧模式不能混用。
- 音频参考必须与图片参考一起使用。
- 按次计费，`duration` 不参与时长倍率计算。
