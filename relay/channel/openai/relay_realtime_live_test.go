package openai

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// TestOpenaiRealtimeHandlerLiveDashScopeASR 针对真实 DashScope 的端到端冒烟测试。
// 需要环境变量 DASHSCOPE_WS_TOKEN（短期有效的 sk-ws- token），否则跳过。
func TestOpenaiRealtimeHandlerLiveDashScopeASR(t *testing.T) {
	token := os.Getenv("DASHSCOPE_WS_TOKEN")
	if token == "" {
		t.Skip("DASHSCOPE_WS_TOKEN not set, skipping live test")
	}

	pcm, err := os.ReadFile("testdata/asr_sample.pcm")
	require.NoError(t, err)

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	// 下游客户端 mock：发送 session.update + 分片音频 + session.finish，并收集回包
	receivedCh := make(chan string, 200)
	client := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteJSON(map[string]any{
			"event_id": "evt_update",
			"type":     "session.update",
			"session": map[string]any{
				"input_audio_format":        "pcm",
				"sample_rate":               16000,
				"input_audio_transcription": map[string]any{"language": "zh"},
				"turn_detection":            map[string]any{"type": "server_vad", "threshold": 0.0, "silence_duration_ms": 400},
			},
		})
		const chunkBytes = 3200 // 100ms @ 16kHz/16bit，模拟实时推流
		for i := 0; i < len(pcm); i += chunkBytes {
			end := i + chunkBytes
			if end > len(pcm) {
				end = len(pcm)
			}
			_ = conn.WriteJSON(map[string]any{
				"event_id": "evt_aud",
				"type":     "input_audio_buffer.append",
				"audio":    base64.StdEncoding.EncodeToString(pcm[i:end]),
			})
			time.Sleep(100 * time.Millisecond)
		}
		_ = conn.WriteJSON(map[string]any{"event_id": "evt_finish", "type": "session.finish"})
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			receivedCh <- string(msg)
			// 收到 session.finished 后主动断开，模拟真实客户端行为（DashScope 会等待客户端断开）
			if strings.Contains(string(msg), "session.finished") {
				return
			}
		}
	}))
	defer client.Close()
	clientURL := "ws" + strings.TrimPrefix(client.URL, "http")

	targetURL := "wss://dashscope.aliyuncs.com/api-ws/v1/realtime?model=qwen3-asr-flash-realtime-2026-02-10"
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	targetConn, _, err := websocket.DefaultDialer.Dial(targetURL, header)
	require.NoError(t, err)
	defer targetConn.Close()

	clientConn, _, err := websocket.DefaultDialer.Dial(clientURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/realtime?model=qwen3-asr-flash-realtime-2026-02-10", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName:  "qwen3-asr-flash-realtime-2026-02-10",
		UsePrice:         true,
		ClientWs:         clientConn,
		TargetWs:         targetConn,
		InputAudioFormat: "pcm",
		IsFirstRequest:   true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "qwen3-asr-flash-realtime-2026-02-10",
		},
	}

	type result struct {
		err   *types.NewAPIError
		usage *dto.RealtimeUsage
	}
	resultCh := make(chan result, 1)
	go func() {
		e, u := OpenaiRealtimeHandler(c, info)
		resultCh <- result{err: e, usage: u}
	}()

	var received []string
	var transcript string
	deadline := time.After(25 * time.Second)
collectLoop:
	for {
		select {
		case m := <-receivedCh:
			received = append(received, m)
			if strings.Contains(m, "conversation.item.input_audio_transcription.completed") {
				var ev dto.RealtimeEvent
				if err := json.Unmarshal([]byte(m), &ev); err == nil && ev.Transcript != "" {
					transcript = ev.Transcript
				}
			}
			if strings.Contains(m, "session.finished") {
				break collectLoop
			}
		case <-deadline:
			t.Fatalf("timed out waiting for transcription; received=%v", received)
		}
	}

	require.NotEmpty(t, transcript, "expected a non-empty transcript; received=%v", received)
	t.Logf("live transcript=%q", transcript)

	select {
	case res := <-resultCh:
		require.Nil(t, res.err)
		require.NotNil(t, res.usage)
		require.True(t, res.usage.TotalTokens > 0, "expected captured usage > 0, got %+v", res.usage)
		t.Logf("live usage total=%d input=%d output=%d audio_in=%d text_out=%d",
			res.usage.TotalTokens, res.usage.InputTokens, res.usage.OutputTokens,
			res.usage.InputTokenDetails.AudioTokens, res.usage.OutputTokenDetails.TextTokens)
	case <-time.After(10 * time.Second):
		t.Fatal("realtime handler did not return")
	}
}
