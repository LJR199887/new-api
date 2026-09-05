# 933 视频模型接入

参考本地 fa2api README / model-call-guide；素材数量以本次需求为准，覆盖旧文档中的 4 图 / 1 视频 / 1 音频。

## 渠道与接口

使用现有 OpenAI 视频任务适配链路，渠道 Base URL 指向 fa2api 服务根地址，Key 使用 fa2api 的公共 API Key，而不是 Fish Token。渠道中添加以下模型，**不要映射成 Seedance 或旧的 video-2.0 名称**：

| 模型 | 固定分辨率 | 生成时长 |
| --- | --- | --- |
| `933-video2.0` | 720p | 4–15 秒（整数） |
| `933-video2.0-480p` | 480p | 4–15 秒（整数） |
| `933-video2.0-mini` | 720p | 4–15 秒（整数） |
| `933-video2.0-mini-480p` | 480p | 4–15 秒（整数） |

生成时长按运营者最新确认支持 4–15 秒的全部整数值，覆盖参考文档旧的 4、5 秒限制；默认仍为 5 秒。比例：16:9、9:16、4:3、3:4、1:1、21:9，默认 16:9。显式分辨率与模型不一致会报错，不静默降档。

- 提交：`POST /v1/video/generations`，`Content-Type: application/json`。
- 查询：`GET /v1/video/generations/{task_id}`，使用 new-api 返回的公开任务 ID。
- 上游 HTTP 202 异步任务复用现有 new-api 提交响应与轮询格式；不要使用 fa2api 的内部任务 ID 查询 new-api。
- 不新增渠道类型；异步任务不使用 StreamOptions。

## 按秒计费

在模型定价中给四个模型分别选择「按时长」并填写每秒价格（`ModelPriceBySeconds` 的 `per_second`）。不配置会拒绝提交，不能退回按次或按 Token 计费。

下面仅为配置结构示例，**0.1 不是建议售价，也不是 Fish 积分换算结果**：

```json
{
  "933-video2.0": {"per_second": 0.1},
  "933-video2.0-480p": {"per_second": 0.1},
  "933-video2.0-mini": {"per_second": 0.1},
  "933-video2.0-mini-480p": {"per_second": 0.1}
}
```

费用 = 每秒单价 × **生成**秒数 × 分组倍率。配置分组专属每秒价格时优先使用该价格，不重复乘分组倍率。参考素材时长、图片数量不额外乘入费用。分辨率由不同模型各自的单价体现。失败任务沿用现有退款链路。

## 文生视频

```json
{"model":"933-video2.0-mini","prompt":"无人机飞跃原始森林","duration":5,"aspect_ratio":"9:16"}
```

## 首尾帧

必须同时提供首、尾两张图片；不能与普通图片、视频、音频参考混用。

```json
{
  "model":"933-video2.0-480p",
  "prompt":"从清晨逐渐过渡到黄昏",
  "duration":4,
  "start_image_url":"https://example.com/start.png",
  "end_image_url":"https://example.com/end.png"
}
```

## 多图 / 视频 / 音频参考

```json
{
  "model":"933-video2.0-mini-480p",
  "prompt":"按参考角色、运动和音乐生成短片",
  "duration":5,
  "image_urls":["https://example.com/character.png"],
  "video_urls":["https://example.com/motion.mp4"],
  "audio_urls":["https://example.com/music.mp3"]
}
```

- 图片最多 9 张，单张不大于 `20 × 1024 × 1024` 字节，包含恰好 20 MiB。HTTP(S) URL 或图片 Base64 Data URL；校验实际下载 / 解码后的字节数。
- 视频最多 3 个，单个不大于 200 MiB，每个 2–15 秒，视频总时长不大于 15 秒。
- 音频只在「多模态」模式中提供，最多 3 个，单个不大于 15 MiB，每个 2–15 秒，音频总时长不大于 15 秒；与视频总时长分别统计。
- 视频和音频使用 HTTP(S) URL。兼容 `video_reference` / `audio_reference` 的 `[{"url":"...","duration":5}]` 格式；显式时长先校验，但不能替代后端的实际媒体探测。上游统一收到 `video_urls` / `audio_urls`。
- 同类字段仅选一种表示，不能混传单数 URL、数组和 reference 别名以规避上限。
- 后端复用纯 Go 媒体解析器，支持 MP4/MOV/WebM、MP3/WAV/M4A/FLAC/OGG 等已有格式；不能探测时长的文件拒绝提交，不信任客户端填写的时长。视频下载限 200 MiB，音频下载限 15 MiB，单个下载超时 30 秒；顺序使用临时文件，结束即删除，沿用现有 SSRF、重定向和 Worker 设置。
- 创作中心不再提供独立「音频参考」模式；音频归入「多模态」，音频和视频均复用 WAN3.0 的「上传素材」交互和外部图床直传链路。常规图片上传仍保留原 10 MiB 上限，只有选择这四个模型时允许 20 MiB。

WAN3 系列在多模态模式下最多 5 个音频，MiniMax H3 系列最多 3 个音频；两者均为单音频 1–15 秒、音频总时长不超过 15 秒、单个不大于 15 MiB。

## 上游状态与验证边界

提供的 fa2api 文档明确注明：参考模式的高层生成仍可能拒绝调用，待补齐媒体检查 / 提交协议。new-api 只负责本仓库的接入、校验、计费和轮询，**不能替 fa2api 解锁尚未实现的能力**。本次不改动 fa2api 仓库，也未进行付费生成测试；上线前应使用支持这些模式的 fa2api 版本验证四类参考请求。

本地验证：

```text
go test ./relay/channel/task/sora ./relay/helper ./relay ./common ./controller ./setting/ratio_setting ./service
cd web
bun test tests/video933.test.js
bun run build
```

### 本次本地验证结果

- Go 相关包回归通过（适配器、计费、任务、公共工具、Controller、配置和 Service）。
- 前端能力与参数测试通过，包括新模型参数与修改模块的 JSX 解析。
- 完整前端构建未通过：当前安装的 `react-icons@5.7.0` 不导出既有代码 `web/src/helpers/render.jsx` 引用的 `SiLinkedin`。这是本次未修改的代码 / 依赖组合问题；未通过删除图标或绕开打包检查掩盖它。
