package openai

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// TestOpenaiRealtimeProductionASR 生产环境端到端测试：以纯客户端身份直连生产网关的 /v1/realtime，
// 走完整链路（生产 new-api -> 阿里渠道 getAliRealtimeURL -> DashScope），验证 qwen3-asr-flash-realtime 可正常转写。
//
// 需要环境变量：
//
//	NEW_API_KEY        生产网关的 API Key（必填，否则跳过）
//	NEW_API_BASE_URL   生产网关地址（默认 https://tokens.buildingblock.top）
//	NEW_API_MODEL      模型名（默认 qwen3-asr-flash-realtime-2026-02-10）
func TestOpenaiRealtimeProductionASR(t *testing.T) {
	apiKey := os.Getenv("NEW_API_KEY")
	if apiKey == "" {
		t.Skip("NEW_API_KEY not set, skipping production test")
	}
	baseURL := strings.TrimRight(os.Getenv("NEW_API_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "https://tokens.buildingblock.top"
	}
	model := os.Getenv("NEW_API_MODEL")
	if model == "" {
		model = "qwen3-asr-flash-realtime-2026-02-10"
	}

	pcm, err := os.ReadFile("testdata/asr_sample.pcm")
	require.NoError(t, err)

	host := strings.TrimPrefix(strings.TrimPrefix(baseURL, "https://"), "http://")
	wsURL := "wss://" + host + "/v1/realtime?model=" + model
	header := http.Header{}
	// 注意：不要设置 Sec-WebSocket-Protocol: realtime（不带 key 会被鉴权中间件覆盖 Authorization）
	header.Set("Authorization", "Bearer "+apiKey)

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("dial failed: %v, http status=%d, body=%s", err, resp.StatusCode, string(body))
		}
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode, "expected 101 handshake")
	t.Logf("connected to %s (subprotocol=%q)", wsURL, conn.Subprotocol())

	// 并发读取服务端回包（含可能的 error 事件）
	receivedCh := make(chan string, 500)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			receivedCh <- string(msg)
		}
	}()

	// 1. session.update
	require.NoError(t, conn.WriteJSON(map[string]any{
		"event_id": "evt_update",
		"type":     "session.update",
		"session": map[string]any{
			"input_audio_format":        "pcm",
			"sample_rate":               16000,
			"input_audio_transcription": map[string]any{"language": "zh"},
			"turn_detection":            map[string]any{"type": "server_vad", "threshold": 0.0, "silence_duration_ms": 400},
		},
	}))

	// 2. 按实时节奏分片推流（100ms/块 @ 16kHz/16bit）
	const chunkBytes = 3200
	for i := 0; i < len(pcm); i += chunkBytes {
		end := i + chunkBytes
		if end > len(pcm) {
			end = len(pcm)
		}
		if err := conn.WriteJSON(map[string]any{
			"event_id": "evt_aud",
			"type":     "input_audio_buffer.append",
			"audio":    base64.StdEncoding.EncodeToString(pcm[i:end]),
		}); err != nil {
			t.Logf("write audio failed at chunk %d: %v", i/chunkBytes, err)
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 3. session.finish
	if err := conn.WriteJSON(map[string]any{"event_id": "evt_finish", "type": "session.finish"}); err != nil {
		t.Logf("write session.finish failed: %v", err)
	}

	// 4. 收集回包直到 session.finished / error / 连接关闭
	var (
		transcript string
		usage      *dto.RealtimeUsage
		seenEvents []string
		errEvent   string
	)
	deadline := time.After(30 * time.Second)
collectLoop:
	for {
		select {
		case m := <-receivedCh:
			var ev dto.RealtimeEvent
			if err := json.Unmarshal([]byte(m), &ev); err != nil {
				t.Logf("raw event: %s", m)
				continue
			}
			seenEvents = append(seenEvents, ev.Type)
			switch ev.Type {
			case "error":
				errEvent = m
				t.Logf("ERROR event: %s", m)
			case dto.RealtimeEventTypeInputAudioTranscriptionCompleted:
				if ev.Transcript != "" {
					transcript = ev.Transcript
				}
				if ev.Usage != nil {
					usage = ev.Usage
				}
				t.Logf("transcription completed: %q", ev.Transcript)
			case "session.finished":
				t.Logf("session.finished received")
				break collectLoop
			}
		case <-deadline:
			break collectLoop
		}
	}

	if errEvent != "" {
		t.Fatalf("received error event from gateway: %s (events=%v)", errEvent, seenEvents)
	}
	require.NotEmpty(t, transcript, "expected a non-empty transcript; events=%v", seenEvents)
	t.Logf("RESULT transcript=%q", transcript)
	if usage != nil {
		t.Logf("RESULT usage total=%d input=%d output=%d audio_in=%d text_out=%d",
			usage.TotalTokens, usage.InputTokens, usage.OutputTokens,
			usage.InputTokenDetails.AudioTokens, usage.OutputTokenDetails.TextTokens)
	}
}
