# 实时语音识别（qwen3-asr-flash-realtime）接入文档

> 通过网关 **tokens.buildingblock.top** 调用阿里云百炼实时语音识别模型 qwen3-asr-flash-realtime-2026-02-10。
> 协议为 WebSocket，事件格式兼容 OpenAI Realtime（ASR 子集）。

---

## 1. 快速概览

| 项目 | 值 |
|---|---|
| 接口类型 | WebSocket（wss://，不是 HTTP） |
| 端点 | wss://tokens.buildingblock.top/v1/realtime?model=qwen3-asr-flash-realtime-2026-02-10 |
| 鉴权 | 请求头 Authorization: Bearer <API_KEY> |
| API Key | new-api 网关令牌（sk- 开头），**不是**阿里云 DashScope 的 key |
| 模型 | qwen3-asr-flash-realtime-2026-02-10（要写全，带日期后缀） |
| 输入 | 实时音频流（PCM16 16kHz 单声道，或 opus） |
| 输出 | 实时识别文本（流式中间结果 + 最终结果） |

---

## 2. WebSocket 连接

~~~text
GET wss://tokens.buildingblock.top/v1/realtime?model=qwen3-asr-flash-realtime-2026-02-10

Headers:
  Authorization: Bearer sk-xxxxxx
~~~

**重要**：不要额外设置 Sec-WebSocket-Protocol: realtime 这个头（除非你是 OpenAI 官方 SDK 的完整协议头格式）。网关鉴权默认读 Authorization，如果你手动只发了一个 realtime 子协议头，会导致鉴权被覆盖、返回 401。

---

## 3. 事件协议

所有消息都是 **JSON 文本帧**（文本 WebSocket 帧），结构类似：

~~~json
{ "event_id": "事件ID（自定义字符串）", "type": "事件类型", "...": "其它字段" }
~~~

### 3.1 客户端 → 服务端（发送）

| 事件 | 说明 |
|---|---|
| session.update | 配置会话：音频格式、采样率、语言、VAD 等（可选，建议连接后先发） |
| input_audio_buffer.append | 推送一段音频（base64 编码） |
| input_audio_buffer.commit | 手动模式（非 VAD）下触发一次识别 |
| session.finish | 通知服务端结束会话 |

### 3.2 服务端 → 客户端（接收）

| 事件 | 说明 |
|---|---|
| session.created | 连接成功后服务端发的第一个事件，含默认配置 |
| session.updated | 确认 session.update 已生效 |
| input_audio_buffer.speech_started | VAD 检测到语音开始 |
| input_audio_buffer.speech_stopped | VAD 检测到语音结束 |
| conversation.item.created | 创建了一个识别项（item） |
| conversation.item.input_audio_transcription.text | 流式中间结果（text=已确认前缀，stash=仍在修正的草稿） |
| conversation.item.input_audio_transcription.completed | **最终识别结果**（含 transcript 和 usage） |
| conversation.item.input_audio_transcription.failed | 识别失败 |
| session.finished | 会话结束（收到后可断开连接） |
| error | 错误事件 |

---

## 4. 交互流程（VAD 模式，默认）

~~~text
1. 建立 WebSocket 连接
2. 收到 session.created
3. 发送 session.update（配置语言、VAD 等）  <- 可选
4. 收到 session.updated
5. 循环发送 input_audio_buffer.append（按实时节奏流式推音频）
6. 发送 session.finish
7. 收到 conversation.item.input_audio_transcription.completed（最终文本）
8. 收到 session.finished -> 客户端主动断开
~~~

**手动模式**：把 session.update 里的 turn_detection 设为 null，然后每段完整语音推完后自己发一次 input_audio_buffer.commit 来断句。

---

## 5. session.update 配置示例

~~~json
{
  "event_id": "evt_update",
  "type": "session.update",
  "session": {
    "input_audio_format": "pcm",
    "sample_rate": 16000,
    "input_audio_transcription": { "language": "zh" },
    "turn_detection": { "type": "server_vad", "threshold": 0.0, "silence_duration_ms": 400 }
  }
}
~~~

字段说明：

- input_audio_format：音频格式，pcm 或 opus，默认 pcm。
- sample_rate：16000 或 8000，默认 16000。
- input_audio_transcription.language：语种，如 zh（普通话）、en、yue（粤语）、ja、ko 等，见第 8 节。
- turn_detection：VAD 配置。type 固定 server_vad；threshold 灵敏度（-1~1，推荐 0.0）；silence_duration_ms 静音断句阈值（推荐 400）。设为 null 则关闭 VAD、进入手动模式。

---

## 6. 音频格式要求

- 编码：**PCM16**（16-bit 小端，单声道），或 opus
- 采样率：**16000 Hz**（也支持 8000，但不推荐）
- 分片：建议每片 **100ms**（16000Hz × 2 字节 × 0.1s = 3200 字节），base64 编码后放进 audio 字段
- 节奏：**必须按实时节奏流式发送**（比如每 100ms 发一片）。一次性把整段音频灌完 + 立刻 session.finish，会被 VAD 判定为无语音，拿不到识别结果

input_audio_buffer.append 示例：

~~~json
{
  "event_id": "evt_aud",
  "type": "input_audio_buffer.append",
  "audio": "base64编码的PCM16音频数据"
}
~~~

---

## 7. 最终结果与计费 usage

conversation.item.input_audio_transcription.completed 事件示例：

~~~json
{
  "event_id": "evt_xxx",
  "type": "conversation.item.input_audio_transcription.completed",
  "item_id": "item_xxx",
  "content_index": 0,
  "transcript": "今天天气怎么样？",
  "language": "zh",
  "emotion": "neutral",
  "usage": {
    "duration": 2,
    "total_tokens": 41,
    "input_tokens": 30,
    "output_tokens": 11,
    "input_tokens_details": { "text_tokens": 3, "audio_tokens": 27 },
    "output_tokens_details": { "text_tokens": 11 }
  }
}
~~~

- transcript：**最终识别文本**。
- usage：本次识别消耗，audio_tokens 是音频 token，text_tokens 是文本 token，duration 是音频时长（秒）。

流式中间结果 conversation.item.input_audio_transcription.text：

~~~json
{
  "event_id": "evt_xxx",
  "type": "conversation.item.input_audio_transcription.text",
  "item_id": "item_xxx",
  "content_index": 0,
  "text": "今天",
  "stash": "天气怎么样",
  "language": "zh",
  "emotion": "neutral"
}
~~~

实时预览 = text + stash（text 是已确认不再变的部分，stash 是还在修正的草稿）。

---

## 8. 支持语种

zh（普通话/四川话/闽南语/吴语）、yue（粤语）、en、ja、de、ko、ru、fr、pt、ar、it、es、hi、id、th、tr、uk、vi、cs、da、fil、fi、is、ms、no、pl、sv。

情绪 emotion 输出：surprised、neutral、happy、sad、disgusted、angry、fearful。

---

## 9. 完整示例（Python）

依赖：pip install websocket-client

~~~python
# -*- coding: utf-8 -*-
"""
调用 tokens.buildingblock.top 的 qwen3-asr-flash-realtime 实时语音识别
"""
import base64
import json
import time
import wave
import websocket  # websocket-client

API_BASE = "tokens.buildingblock.top"
API_KEY = "sk-xxxxxx"  # new-api 网关令牌
MODEL = "qwen3-asr-flash-realtime-2026-02-10"
WS_URL = "wss://" + API_BASE + "/v1/realtime?model=" + MODEL


def read_pcm16_from_wav(path, chunk_ms=100):
    """从 16kHz/16bit/单声道 WAV 读取 PCM16，按 chunk_ms 切块"""
    with wave.open(path, "rb") as wf:
        assert wf.getframerate() == 16000, "采样率必须为 16000"
        assert wf.getsampwidth() == 2, "必须为 16bit"
        assert wf.getnchannels() == 1, "必须为单声道"
        nbytes = int(16000 * 2 * chunk_ms / 1000)
        while True:
            data = wf.readframes(nbytes // 2)
            if not data:
                break
            yield data


def transcribe(wav_path, language="zh"):
    ws = websocket.create_connection(
        WS_URL,
        header=["Authorization: Bearer " + API_KEY],
        timeout=30,
    )

    # 1. 配置会话
    ws.send(json.dumps({
        "event_id": "evt_update",
        "type": "session.update",
        "session": {
            "input_audio_format": "pcm",
            "sample_rate": 16000,
            "input_audio_transcription": {"language": language},
            "turn_detection": {"type": "server_vad", "threshold": 0.0, "silence_duration_ms": 400},
        },
    }))

    # 2. 按实时节奏流式推音频
    for pcm in read_pcm16_from_wav(wav_path):
        ws.send(json.dumps({
            "event_id": "evt_aud",
            "type": "input_audio_buffer.append",
            "audio": base64.b64encode(pcm).decode(),
        }))
        time.sleep(0.1)

    # 3. 结束会话
    ws.send(json.dumps({"event_id": "evt_finish", "type": "session.finish"}))

    # 4. 收结果
    final_text = ""
    while True:
        msg = ws.recv()
        if not msg:
            break
        ev = json.loads(msg)
        t = ev.get("type")
        if t == "conversation.item.input_audio_transcription.completed":
            final_text += ev.get("transcript", "")
            print("识别:", ev.get("transcript"))
        elif t == "session.finished":
            break
        elif t == "error":
            raise RuntimeError("服务端错误: " + msg)
    ws.close()
    return final_text


if __name__ == "__main__":
    text = transcribe("audio_16k_mono.wav")
    print("最终文本:", text)
~~~

---

## 10. 注意事项（避坑清单）

1. **是 WebSocket，不是 HTTP**：地址是 wss://，别用普通的 HTTP POST。
2. **鉴权用网关令牌**：Authorization: Bearer <new-api 的 sk-...>，不是阿里云 DashScope 的 key。
3. **别发裸的 Sec-WebSocket-Protocol: realtime**：会让网关鉴权失败（401）。要么只发 Authorization，要么发完整的 OpenAI 风格子协议头（realtime, openai-insecure-api-key.<key>, openai-beta.realtime-v1）。
4. **音频格式要匹配**：PCM16、16kHz、单声道（或 opus）；8bit、立体声、48kHz 都会识别错或失败。
5. **按实时节奏推流**：每 ~100ms 推一片；一次性灌完会导致 VAD 判为无语音。
6. **结束要发 session.finish 并等 session.finished**：VAD 模式下直接断连会丢弃未完成的识别项。
7. **模型名要写全**：qwen3-asr-flash-realtime-2026-02-10，带日期后缀，别写成 qwen3-asr-flash-realtime。
8. **最终文本以 completed 事件的 transcript 为准**；text 事件只是流式预览。
9. **error 事件要处理**：识别/连接出错时服务端会发 error，记得读它而不是忽略。

---

## 11. 常见错误速查

| 现象 | 原因 | 处理 |
|---|---|---|
| 连接返回 401 Invalid token | API Key 错误/过期 | 换有效的网关令牌 |
| 握手返回 400 | 走的 HTTP 或代理没转发 WebSocket 头 | 确认用 wss://，检查网关代理 Upgrade/Connection |
| 连上但收不到 completed | 音频一次性灌完 / 格式不对 / VAD 没检测到语音 | 按 100ms 实时推流，确认 PCM16 16kHz 单声道 |
| error 事件 code=do_request_failed | 上游（阿里云）握手失败 | 检查渠道的 API Key 是否在全局 dashscope 域名下可用 |
