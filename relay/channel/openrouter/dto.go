package openrouter

import "encoding/json"

// ── Chat Completions ────────────────────────────────────────────────

type RequestReasoning struct {
	Enabled bool `json:"enabled"`
	// One of the following (not both):
	Effort    string `json:"effort,omitempty"`     // Can be "high", "medium", or "low" (OpenAI-style)
	MaxTokens int    `json:"max_tokens,omitempty"` // Specific token limit (Anthropic-style)
	// Optional: Default is false. All models support this.
	Exclude bool `json:"exclude,omitempty"` // Set to true to exclude reasoning tokens from response
}

type OpenRouterEnterpriseResponse struct {
	Data    json.RawMessage `json:"data"`
	Success bool            `json:"success"`
}

// ── Unified Image API (POST /api/v1/images) ─────────────────────────

// ImageGenerationRequest is the OpenRouter dedicated Image API request body.
// Used for both text-to-image (generations) and image-to-image (edits).
type ImageGenerationRequest struct {
	Model           string                `json:"model"`
	Prompt          string                `json:"prompt"`
	InputReferences []ImageReferenceInput `json:"input_references,omitempty"` // for image-to-image / edits
	Resolution      string                `json:"resolution,omitempty"`       // "512", "1K", "2K", "4K"
	AspectRatio     string                `json:"aspect_ratio,omitempty"`     // "1:1", "16:9", "9:16", "3:2", "2:3", etc.
	N               *uint                 `json:"n,omitempty"`                // 1–10
	Quality         string                `json:"quality,omitempty"`          // "auto", "low", "medium", "high"
	OutputFormat    string                `json:"output_format,omitempty"`    // "png", "jpeg", "webp"
	OutputCompression *int                `json:"output_compression,omitempty"` // 0–100 for jpeg/webp
	Background      string                `json:"background,omitempty"`       // "auto", "transparent", "opaque"
	Seed            *int64                `json:"seed,omitempty"`
	Stream          *bool                 `json:"stream,omitempty"`
	Size            string                `json:"size,omitempty"`             // convenience shorthand
	Provider        json.RawMessage       `json:"provider,omitempty"`         // provider passthrough
	ExtraFields     json.RawMessage       `json:"extra_fields,omitempty"`     // passthrough
}

// ImageReferenceInput is a reference image for image-to-image generation.
type ImageReferenceInput struct {
	Type     string            `json:"type"`     // "image_url"
	ImageURL ImageReferenceURL `json:"image_url"`
}

// ImageReferenceURL holds the URL or data URL of a reference image.
type ImageReferenceURL struct {
	URL string `json:"url"` // HTTPS URL or "data:image/png;base64,..."
}

// ImageGenerationResponse is the OpenRouter dedicated Image API response body.
type ImageGenerationResponse struct {
	Data    []ImageGenerationData `json:"data"`
	Created int64                 `json:"created"`
	Usage   ImageGenerationUsage  `json:"usage,omitempty"`
}

// ImageGenerationData holds a single generated image.
type ImageGenerationData struct {
	B64Json    string `json:"b64_json"`
	MediaType  string `json:"media_type,omitempty"` // e.g. "image/svg+xml" for vector models
}

// ImageGenerationUsage holds the usage/cost info from the Image API.
type ImageGenerationUsage struct {
	Tokens  int     `json:"tokens"`
	CostUSD float64 `json:"cost_usd"`
}
