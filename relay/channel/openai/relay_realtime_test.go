package openai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// TestOpenaiRealtimeHandlerRelaysAndCapturesASRUsage 端到端验证：
// 客户端 -> new-api(OpenaiRealtimeHandler) -> mock 上游 -> 回传，
// 并校验 Qwen-ASR-Realtime 的 usage（携带在 conversation.item.input_audio_transcription.completed 事件上）被正确采集。
func TestOpenaiRealtimeHandlerRelaysAndCapturesASRUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	// 上游 mock：模拟 DashScope 实时 ASR 服务端
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteJSON(map[string]any{
			"event_id": "evt_session_created",
			"type":     "session.created",
			"session": map[string]any{
				"object":             "realtime.session",
				"model":              "qwen3-asr-flash-realtime",
				"modalities":         []string{"text"},
				"input_audio_format": "pcm",
				"sample_rate":        16000,
			},
		})
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var ev struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(msg, &ev)
			if ev.Type == "session.finish" {
				_ = conn.WriteJSON(map[string]any{
					"event_id":      "evt_transcription_completed",
					"type":          "conversation.item.input_audio_transcription.completed",
					"item_id":       "item_1",
					"content_index": 0,
					"transcript":    "今天天气怎么样",
					"language":      "zh",
					"usage": map[string]any{
						"total_tokens":          20,
						"input_tokens":          15,
						"output_tokens":         5,
						"input_tokens_details":  map[string]any{"text_tokens": 3, "audio_tokens": 12},
						"output_tokens_details": map[string]any{"text_tokens": 5},
					},
				})
				_ = conn.WriteJSON(map[string]any{"event_id": "evt_session_finished", "type": "session.finished"})
				return
			}
		}
	}))
	defer upstream.Close()
	upstreamURL := "ws" + strings.TrimPrefix(upstream.URL, "http")

	// 下游客户端 mock：向 new-api 发送 session.update + session.finish，并收集回包
	receivedCh := make(chan string, 100)
	client := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteJSON(map[string]any{
			"event_id": "evt_update",
			"type":     "session.update",
			"session":  map[string]any{"input_audio_transcription": map[string]any{"language": "zh"}},
		})
		_ = conn.WriteJSON(map[string]any{"event_id": "evt_finish", "type": "session.finish"})
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			receivedCh <- string(msg)
		}
	}))
	defer client.Close()
	clientURL := "ws" + strings.TrimPrefix(client.URL, "http")

	targetConn, _, err := websocket.DefaultDialer.Dial(upstreamURL, nil)
	require.NoError(t, err)
	defer targetConn.Close()

	clientConn, _, err := websocket.DefaultDialer.Dial(clientURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/realtime?model=qwen3-asr-flash-realtime", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName:  "qwen3-asr-flash-realtime",
		UsePrice:         true, // 跳过真实 DB 计费，仅验证 usage 采集
		ClientWs:         clientConn,
		TargetWs:         targetConn,
		InputAudioFormat: "pcm",
		IsFirstRequest:   true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "qwen3-asr-flash-realtime",
		},
	}

	type result struct {
		err   *types.NewAPIError
		usage *dto.RealtimeUsage
	}
	resultCh := make(chan result, 1)
	go func() {
		newAPIErr, sumUsage := OpenaiRealtimeHandler(c, info)
		resultCh <- result{err: newAPIErr, usage: sumUsage}
	}()

	// 收集客户端收到的回包，直到收到 session.finished
	var received []string
	deadline := time.After(8 * time.Second)
collectLoop:
	for {
		select {
		case m := <-receivedCh:
			received = append(received, m)
			if strings.Contains(m, "session.finished") {
				break collectLoop
			}
		case <-deadline:
			t.Fatalf("timed out waiting for relayed events; received=%v", received)
		}
	}

	joined := strings.Join(received, "\n")
	require.Contains(t, joined, "session.created")
	require.Contains(t, joined, "conversation.item.input_audio_transcription.completed")
	require.Contains(t, joined, "今天天气怎么样")

	// 校验 handler 返回的累计 usage
	select {
	case res := <-resultCh:
		require.Nil(t, res.err)
		require.NotNil(t, res.usage)
		require.Equal(t, 20, res.usage.TotalTokens)
		require.Equal(t, 15, res.usage.InputTokens)
		require.Equal(t, 5, res.usage.OutputTokens)
		require.Equal(t, 12, res.usage.InputTokenDetails.AudioTokens)
		require.Equal(t, 3, res.usage.InputTokenDetails.TextTokens)
		require.Equal(t, 5, res.usage.OutputTokenDetails.TextTokens)
	case <-time.After(5 * time.Second):
		t.Fatal("realtime handler did not return")
	}
}
