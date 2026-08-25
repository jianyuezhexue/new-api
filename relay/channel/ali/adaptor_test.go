package ali

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestGetAliRealtimeURL(t *testing.T) {
	cases := []struct {
		name     string
		baseURL  string
		model    string
		expected string
	}{
		{
			name:     "default dashscope",
			baseURL:  "https://dashscope.aliyuncs.com",
			model:    "qwen3-asr-flash-realtime-2026-02-10",
			expected: "wss://dashscope.aliyuncs.com/api-ws/v1/realtime?model=qwen3-asr-flash-realtime-2026-02-10",
		},
		{
			name:     "intl dashscope",
			baseURL:  "https://dashscope-intl.aliyuncs.com",
			model:    "qwen3-asr-flash-realtime",
			expected: "wss://dashscope-intl.aliyuncs.com/api-ws/v1/realtime?model=qwen3-asr-flash-realtime",
		},
		{
			name:     "workspace llm domain falls back to global dashscope",
			baseURL:  "https://llm-ghhs52oe6b4pzikx.cn-beijing.maas.aliyuncs.com",
			model:    "qwen3-asr-flash-realtime-2026-02-10",
			expected: "wss://dashscope.aliyuncs.com/api-ws/v1/realtime?model=qwen3-asr-flash-realtime-2026-02-10",
		},
		{
			name:     "workspace domain without llm- prefix falls back",
			baseURL:  "https://ghhs52oe6b4pzikx.cn-beijing.maas.aliyuncs.com",
			model:    "qwen3-asr-flash-realtime",
			expected: "wss://dashscope.aliyuncs.com/api-ws/v1/realtime?model=qwen3-asr-flash-realtime",
		},
		{
			name:     "non-dashscope base url falls back to dashscope",
			baseURL:  "https://example.com/v1",
			model:    "qwen3-asr-flash-realtime",
			expected: "wss://dashscope.aliyuncs.com/api-ws/v1/realtime?model=qwen3-asr-flash-realtime",
		},
		{
			name:     "empty base url falls back to dashscope",
			baseURL:  "",
			model:    "qwen3-asr-flash-realtime",
			expected: "wss://dashscope.aliyuncs.com/api-ws/v1/realtime?model=qwen3-asr-flash-realtime",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelBaseUrl:    tc.baseURL,
					UpstreamModelName: tc.model,
				},
			}
			got := getAliRealtimeURL(info)
			require.Equal(t, tc.expected, got)
		})
	}
}
