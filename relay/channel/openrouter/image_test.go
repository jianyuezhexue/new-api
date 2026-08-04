package openrouter

import (
	"bytes"
	"encoding/base64"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── MapSize ──────────────────────────────────────────────────────────

func TestMapSize(t *testing.T) {
	tests := []struct {
		size              string
		wantAspectRatio   string
		wantResolution    string
	}{
		{"256x256", "1:1", "1K"},
		{"512x512", "1:1", "1K"},
		{"1024x1024", "1:1", "1K"},
		{"1792x1024", "16:9", "2K"},
		{"1024x1792", "9:16", "2K"},
		{"1536x1024", "3:2", "2K"},
		{"1024x1536", "2:3", "2K"},
		{"1K", "", "1K"},
		{"2K", "", "2K"},
		{"4K", "", "4K"},
		{"512", "", "512"},
		// Unknown sizes
		{"", "1:1", "1K"},
		{"800x600", "4:3", "1K"},
		{"2048x2048", "1:1", "2K"},
		{"4096x2048", "2:1", "4K"},
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			ar, res := MapSize(tt.size)
			if ar != tt.wantAspectRatio {
				t.Errorf("aspect_ratio = %q, want %q", ar, tt.wantAspectRatio)
			}
			if res != tt.wantResolution {
				t.Errorf("resolution = %q, want %q", res, tt.wantResolution)
			}
		})
	}
}

// ── MapQuality ───────────────────────────────────────────────────────

func TestMapQuality(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"standard", "low"},
		{"low", "low"},
		{"hd", "high"},
		{"high", "high"},
		{"medium", "medium"},
		{"", "auto"},
		{"custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MapQuality(tt.input)
			if got != tt.want {
				t.Errorf("MapQuality(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ── MapOutputFormat ──────────────────────────────────────────────────

func TestMapOutputFormat(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"b64_json", "png"},
		{"url", "png"},
		{"png", "png"},
		{"jpeg", "jpeg"},
		{"jpg", "jpeg"},
		{"webp", "webp"},
		{"", "png"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MapOutputFormat(tt.input)
			if got != tt.want {
				t.Errorf("MapOutputFormat(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ── ConvertImageGenerationRequest ────────────────────────────────────

func TestConvertImageGenerationRequest(t *testing.T) {
	req := &dto.ImageRequest{
		Model:          "openai/gpt-5.4-image-2",
		Prompt:         "a red panda in space",
		Size:           "1024x1024",
		Quality:        "standard",
		ResponseFormat: "b64_json",
		N:              lo.ToPtr(uint(3)),
	}
	or := ConvertImageGenerationRequest(req)

	if or.Model != "openai/gpt-5.4-image-2" {
		t.Errorf("Model = %q, want %q", or.Model, "openai/gpt-5.4-image-2")
	}
	if or.Prompt != "a red panda in space" {
		t.Errorf("Prompt = %q, want %q", or.Prompt, "a red panda in space")
	}
	if or.AspectRatio != "1:1" {
		t.Errorf("AspectRatio = %q, want 1:1", or.AspectRatio)
	}
	if or.Resolution != "1K" {
		t.Errorf("Resolution = %q, want 1K", or.Resolution)
	}
	if or.Quality != "low" {
		t.Errorf("Quality = %q, want low", or.Quality)
	}
	if or.OutputFormat != "png" {
		t.Errorf("OutputFormat = %q, want png", or.OutputFormat)
	}
	if or.N == nil || *or.N != 3 {
		t.Errorf("N = %v, want 3", or.N)
	}
	if len(or.InputReferences) != 0 {
		t.Errorf("InputReferences should be empty for generation, got %d", len(or.InputReferences))
	}
}

func TestConvertImageGenerationRequestDefaults(t *testing.T) {
	req := &dto.ImageRequest{
		Model:  "test-model",
		Prompt: "test prompt",
	}
	or := ConvertImageGenerationRequest(req)

	if or.N == nil || *or.N != 1 {
		t.Errorf("default N = %v, want 1", or.N)
	}
	if or.Quality != "auto" {
		t.Errorf("default Quality = %q, want auto", or.Quality)
	}
	if or.OutputFormat != "png" {
		t.Errorf("default OutputFormat = %q, want png", or.OutputFormat)
	}
	if or.AspectRatio != "1:1" {
		t.Errorf("default AspectRatio = %q, want 1:1", or.AspectRatio)
	}
	if or.Resolution != "1K" {
		t.Errorf("default Resolution = %q, want 1K", or.Resolution)
	}
}

// ── ConvertImageEditRequest ──────────────────────────────────────────

func TestConvertImageEditRequest(t *testing.T) {
	// Setup gin context with multipart form
	ginC, _ := gin.CreateTestContext(httptest.NewRecorder())

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("model", "test-model")
	writer.WriteField("prompt", "add a hat")
	writer.WriteField("n", "2")
	writer.WriteField("size", "1024x1024")

	// Add a fake image file
	part, err := writer.CreateFormFile("image", "test.png")
	if err != nil {
		t.Fatal(err)
	}
	// Write a minimal valid PNG
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}
	part.Write(pngData)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	ginC.Request = req

	// Parse multipart form
	form, err := common.ParseMultipartFormReusable(ginC)
	if err != nil {
		t.Fatal(err)
	}
	ginC.Request.MultipartForm = form

	imgReq := &dto.ImageRequest{
		Model:  "test-model",
		Prompt: "add a hat",
	}

	or, err := ConvertImageEditRequest(ginC, imgReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if or.Model != "test-model" {
		t.Errorf("Model = %q, want test-model", or.Model)
	}
	if or.Prompt != "add a hat" {
		t.Errorf("Prompt = %q, want 'add a hat'", or.Prompt)
	}
	if len(or.InputReferences) != 1 {
		t.Fatalf("expected 1 input_reference, got %d", len(or.InputReferences))
	}
	ref := or.InputReferences[0]
	if ref.Type != "image_url" {
		t.Errorf("reference Type = %q, want image_url", ref.Type)
	}
	if !strings.HasPrefix(ref.ImageURL.URL, "data:image/png;base64,") {
		t.Errorf("reference URL should be base64 data URL, got: %s", ref.ImageURL.URL[:50])
	}
	// Verify the base64 decodes back to our png data
	b64 := strings.TrimPrefix(ref.ImageURL.URL, "data:image/png;base64,")
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	if !bytes.Equal(decoded, pngData) {
		t.Errorf("decoded data doesn't match original")
	}
}

func TestConvertImageEditRequestMissingImage(t *testing.T) {
	ginC, _ := gin.CreateTestContext(httptest.NewRecorder())

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("model", "test-model")
	writer.WriteField("prompt", "add a hat")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	ginC.Request = req

	form, err := common.ParseMultipartFormReusable(ginC)
	if err != nil {
		t.Fatal(err)
	}
	ginC.Request.MultipartForm = form

	imgReq := &dto.ImageRequest{Model: "test-model", Prompt: "add a hat"}
	_, err = ConvertImageEditRequest(ginC, imgReq)
	if err == nil {
		t.Error("expected error for missing image, got nil")
	}
	if !strings.Contains(err.Error(), "image file is required") {
		t.Errorf("error message = %q, want 'image file is required'", err.Error())
	}
}

// ── ParseImageResponse ───────────────────────────────────────────────

func TestParseImageResponse(t *testing.T) {
	responseBody := []byte(`{
		"created": 1700000000,
		"data": [
			{"b64_json": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}
		],
		"usage": {"tokens": 500, "cost_usd": 0.02}
	}`)

	imgResp, usage, err := ParseImageResponse(responseBody, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if imgResp.Created != 1700000000 {
		t.Errorf("Created = %d, want 1700000000", imgResp.Created)
	}
	if len(imgResp.Data) != 1 {
		t.Fatalf("expected 1 image, got %d", len(imgResp.Data))
	}
	if imgResp.Data[0].B64Json == "" {
		t.Error("B64Json should not be empty")
	}
	if usage.TotalTokens != 500 {
		t.Errorf("TotalTokens = %d, want 500", usage.TotalTokens)
	}
}

func TestParseImageResponseMultipleImages(t *testing.T) {
	responseBody := []byte(`{
		"created": 1700000000,
		"data": [
			{"b64_json": "aaa"},
			{"b64_json": "bbb"},
			{"b64_json": "ccc"}
		]
	}`)

	imgResp, _, err := ParseImageResponse(responseBody, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgResp.Data) != 3 {
		t.Fatalf("expected 3 images, got %d", len(imgResp.Data))
	}
}

func TestParseImageResponseWithError(t *testing.T) {
	responseBody := []byte(`{
		"error": {
			"message": "Model not found",
			"type": "invalid_request_error",
			"code": "model_not_found"
		}
	}`)

	_, _, err := ParseImageResponse(responseBody, 404)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseImageResponseNoUsage(t *testing.T) {
	responseBody := []byte(`{
		"data": [{"b64_json": "aaa"}]
	}`)

	_, usage, err := ParseImageResponse(responseBody, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0", usage.TotalTokens)
	}
}

// ── fileHeaderToReference ────────────────────────────────────────────

func TestFileHeaderToReference(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "test.jpg")
	if err != nil {
		t.Fatal(err)
	}
	jpegMagic := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	part.Write(jpegMagic)
	writer.Close()

	// Parse to get the file header
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer form.RemoveAll()

	files := form.File["image"]
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	ref, err := fileHeaderToReference(files[0])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ref.Type != "image_url" {
		t.Errorf("Type = %q, want image_url", ref.Type)
	}
	if !strings.HasPrefix(ref.ImageURL.URL, "data:image/jpeg;base64,") {
		t.Errorf("URL should start with data:image/jpeg;base64,, got: %s", ref.ImageURL.URL[:50])
	}
}

// ── detectMimeType ──────────────────────────────────────────────────

func TestDetectMimeType(t *testing.T) {
	tests := []struct {
		filename string
		data     []byte
		want     string
	}{
		{"test.png", nil, "image/png"},
		{"test.jpg", nil, "image/jpeg"},
		{"test.jpeg", nil, "image/jpeg"},
		{"test.webp", nil, "image/webp"},
		{"test.gif", nil, "image/gif"},
		// fallback to sniffing with sufficient data for detection
		{"test.unknown", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x00}, "image/jpeg"},
		{"test.unknown", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}, "image/png"},
		// no data
		{"unknown.bin", nil, "image/png"},
		{"unknown.bin", []byte{}, "image/png"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := detectMimeType(tt.filename, tt.data)
			if got != tt.want {
				t.Errorf("detectMimeType(%q, %v) = %q, want %q", tt.filename, tt.data, got, tt.want)
			}
		})
	}
}

// ── gcd ─────────────────────────────────────────────────────────────

func TestGcd(t *testing.T) {
	tests := []struct{ a, b, want int }{
		{4, 2, 2},
		{2, 4, 2},
		{15, 10, 5},
		{7, 13, 1},
		{100, 100, 100},
		{0, 5, 5},
		{5, 0, 5},
	}

	for _, tt := range tests {
		got := gcd(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("gcd(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// ── Integration helpers ─────────────────────────────────────────────

func TestCollectImageFiles(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add standard "image" field
	w, _ := writer.CreateFormFile("image", "img1.png")
	w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	// Add "image[]" field
	w, _ = writer.CreateFormFile("image[]", "img2.png")
	w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer form.RemoveAll()

	files := collectImageFiles(form)
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}
