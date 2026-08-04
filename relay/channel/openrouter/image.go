package openrouter

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// ── OpenAI → OpenRouter Request Conversion ────────────────────────────

// ConvertImageGenerationRequest converts an OpenAI image generation request
// to the OpenRouter dedicated Image API format.
func ConvertImageGenerationRequest(req *dto.ImageRequest) *ImageGenerationRequest {
	or := &ImageGenerationRequest{
		Model:        req.Model,
		Prompt:       req.Prompt,
		Quality:      MapQuality(req.Quality),
		OutputFormat: MapOutputFormat(req.ResponseFormat),
		ExtraFields:  req.ExtraFields,
	}

	aspectRatio, resolution := MapSize(req.Size)
	or.AspectRatio = aspectRatio
	or.Resolution = resolution

	if req.N != nil && *req.N > 0 {
		or.N = req.N
	} else {
		or.N = lo.ToPtr(uint(1))
	}

	return or
}

// ConvertImageEditRequest converts an OpenAI image edit multipart request
// to the OpenRouter dedicated Image API format with input_references.
func ConvertImageEditRequest(c *gin.Context, req *dto.ImageRequest) (*ImageGenerationRequest, error) {
	or := ConvertImageGenerationRequest(req)

	mf := c.Request.MultipartForm
	if mf == nil {
		form, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return nil, fmt.Errorf("failed to parse multipart form: %w", err)
		}
		c.Request.MultipartForm = form
		mf = form
	}

	var references []ImageReferenceInput

	imageFiles := collectImageFiles(mf)
	if len(imageFiles) == 0 {
		return nil, fmt.Errorf("image file is required for edits")
	}

	for _, fh := range imageFiles {
		ref, err := fileHeaderToReference(fh)
		if err != nil {
			return nil, fmt.Errorf("failed to read image file %q: %w", fh.Filename, err)
		}
		references = append(references, ref)
	}

	maskFiles := collectFieldFiles(mf, "mask")
	for _, fh := range maskFiles {
		ref, err := fileHeaderToReference(fh)
		if err != nil {
			return nil, fmt.Errorf("failed to read mask file %q: %w", fh.Filename, err)
		}
		references = append(references, ref)
	}

	or.InputReferences = references
	return or, nil
}

// ── OpenRouter Response → OpenAI Response Conversion ─────────────────

// ParseImageResponse parses raw OpenRouter Image API response bytes and
// returns an OpenAI-compatible ImageResponse plus usage.
func ParseImageResponse(responseBody []byte, statusCode int) (*dto.ImageResponse, *dto.Usage, *types.NewAPIError) {
	// Check for error in response body first
	var errResp dto.GeneralErrorResponse
	if err := common.Unmarshal(responseBody, &errResp); err == nil {
		if oaiError := errResp.TryToOpenAIError(); oaiError != nil && oaiError.Type != "" {
			return nil, nil, types.WithOpenAIError(*oaiError, statusCode)
		}
	}

	var orResp ImageGenerationResponse
	if err := common.Unmarshal(responseBody, &orResp); err != nil {
		return nil, nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	imgResp := &dto.ImageResponse{
		Created: orResp.Created,
	}
	for _, d := range orResp.Data {
		imgResp.Data = append(imgResp.Data, dto.ImageData{
			B64Json: d.B64Json,
		})
	}

	usage := &dto.Usage{}
	if orResp.Usage.Tokens > 0 {
		usage.TotalTokens = orResp.Usage.Tokens
	}

	return imgResp, usage, nil
}

// ── Parameter Mapping Helpers ────────────────────────────────────────

// MapSize maps OpenAI size strings to OpenRouter aspect_ratio and resolution values.
func MapSize(size string) (aspectRatio, resolution string) {
	switch size {
	case "256x256", "512x512", "1024x1024":
		return "1:1", "1K"
	case "1792x1024":
		return "16:9", "2K"
	case "1024x1792":
		return "9:16", "2K"
	case "1536x1024":
		return "3:2", "2K"
	case "1024x1536":
		return "2:3", "2K"
	case "1K", "2K", "4K", "512":
		return "", size
	default:
		if size != "" {
			var w, h int
			if _, err := fmt.Sscanf(size, "%dx%d", &w, &h); err == nil && w > 0 && h > 0 {
				g := gcd(w, h)
				return fmt.Sprintf("%d:%d", w/g, h/g), mapResolutionFromWidth(max(w, h))
			}
		}
		return "1:1", "1K"
	}
}

func mapResolutionFromWidth(width int) string {
	switch {
	case width <= 512:
		return "512"
	case width <= 1024:
		return "1K"
	case width <= 2048:
		return "2K"
	default:
		return "4K"
	}
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// MapQuality maps OpenAI quality values to OpenRouter quality values.
func MapQuality(quality string) string {
	switch strings.ToLower(quality) {
	case "standard", "low":
		return "low"
	case "hd", "high":
		return "high"
	case "medium":
		return "medium"
	default:
		if quality != "" {
			return quality
		}
		return "auto"
	}
}

// MapOutputFormat maps OpenAI response_format to OpenRouter output_format.
func MapOutputFormat(format string) string {
	switch strings.ToLower(format) {
	case "b64_json", "url":
		return "png"
	case "png":
		return "png"
	case "jpeg", "jpg":
		return "jpeg"
	case "webp":
		return "webp"
	default:
		return "png"
	}
}

// ── Multipart File Helpers ────────────────────────────────────────────

func collectImageFiles(mf *multipart.Form) []*multipart.FileHeader {
	var files []*multipart.FileHeader

	if fhs := mf.File["image"]; len(fhs) > 0 {
		files = append(files, fhs...)
	}
	if fhs := mf.File["image[]"]; len(fhs) > 0 {
		files = append(files, fhs...)
	}
	for fieldName, fhs := range mf.File {
		if fieldName != "image[]" && strings.HasPrefix(fieldName, "image[") && strings.HasSuffix(fieldName, "]") {
			files = append(files, fhs...)
		}
	}

	return files
}

func collectFieldFiles(mf *multipart.Form, fieldName string) []*multipart.FileHeader {
	return mf.File[fieldName]
}

func fileHeaderToReference(fh *multipart.FileHeader) (ImageReferenceInput, error) {
	file, err := fh.Open()
	if err != nil {
		return ImageReferenceInput{}, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return ImageReferenceInput{}, fmt.Errorf("failed to read file: %w", err)
	}

	mimeType := detectMimeType(fh.Filename, data)
	b64 := base64.StdEncoding.EncodeToString(data)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

	return ImageReferenceInput{
		Type: "image_url",
		ImageURL: ImageReferenceURL{
			URL: dataURL,
		},
	}, nil
}

func detectMimeType(filename string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if mt := mime.TypeByExtension(ext); mt != "" {
		if strings.HasPrefix(mt, "image/") {
			return mt
		}
	}
	if len(data) > 0 {
		return http.DetectContentType(data)
	}
	return "image/png"
}
