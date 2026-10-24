// Package suggest asks Claude for a name and a short description of an item from a photo
// (ADR-0006).
package suggest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Suggestion is a proposed name and description in German.
type Suggestion struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var (
	// ErrRefused means Claude (and its fallback) declined the request.
	ErrRefused = errors.New("suggestion refused")
	// ErrIncomplete means the answer was cut off or unusable.
	ErrIncomplete = errors.New("suggestion incomplete")
)

// DefaultModel is used unless INVENTAR_ANTHROPIC_MODEL names another one.
const DefaultModel = "claude-haiku-5-5"

// fallbackModels support server-side refusal fallbacks with fallbacks "default". Other models,
// Claude Haiku 5.5 among them, have none and get the plain request.
var fallbackModels = map[string]bool{
	"claude-opus-5-5":   true,
	"claude-opus-5":     true,
	"claude-fable-5-1":  true,
	"claude-sonnet-5-5": true,
}

const (
	// maxTokens covers the (always on, short at effort low) thinking plus the small JSON answer.
	maxTokens = 4096
)

// instruction is the German task given next to the photo.
const instruction = `Das Foto zeigt einen Gegenstand aus einem Haushalt, der in eine Inventarliste aufgenommen wird.
Schlage dafür einen Namen und eine kurze Beschreibung auf Deutsch vor.

- name: wie ein Eintrag in einer Haushaltsinventarliste, 2 bis 5 Wörter, z. B. „Stehlampe Messing“ oder „Kochtopf Edelstahl 24 cm“. Nenne eine Marke nur, wenn sie eindeutig zu erkennen ist, rate sie nicht.
- description: ein kurzer Satz mit sichtbaren Merkmalen, die den Gegenstand von ähnlichen unterscheiden (Farbe, Material, Form, Größe), und seinem Zustand, falls erkennbar. Sachlich, keine Werbesprache.

Sind mehrere Gegenstände zu sehen, beschreibe den, der im Mittelpunkt steht.`

// schema is the structured output format of the answer.
var schema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"name": {"type": "string"},
		"description": {"type": "string"}
	},
	"required": ["name", "description"],
	"additionalProperties": false
}`)

// Claude makes suggestions with the Claude API.
type Claude struct {
	client anthropic.Client
	model  string
}

// NewClaude creates a client with the given API key and model (DefaultModel if empty). The key
// is only sent to the API.
func NewClaude(apiKey, model string, opts ...option.RequestOption) *Claude {
	if model == "" {
		model = DefaultModel
	}
	return &Claude{
		client: anthropic.NewClient(append([]option.RequestOption{option.WithAPIKey(apiKey)}, opts...)...),
		model:  model,
	}
}

// Model is the Claude model the suggestions come from.
func (c *Claude) Model() string { return c.model }

// Suggest sends the JPEG photo to Claude and returns its suggestion. The context bounds the
// whole call including retries.
func (c *Claude) Suggest(ctx context.Context, jpeg []byte) (Suggestion, error) {
	params := anthropic.BetaMessageNewParams{
		Model:     c.model,
		MaxTokens: maxTokens,
		// Thinking stays adaptive (it cannot be disabled on current models); effort low keeps it short for this simple extraction.
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: anthropic.BetaOutputConfigEffortLow,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: schema},
		},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(
				anthropic.NewBetaImageBlock(anthropic.BetaBase64ImageSourceParam{
					Data:      base64.StdEncoding.EncodeToString(jpeg),
					MediaType: anthropic.BetaBase64ImageSourceMediaTypeImageJPEG,
				}),
				anthropic.NewBetaTextBlock(instruction),
			),
		},
	}
	if fallbackModels[c.model] {
		// A request declined by the classifiers is retried server-side on the recommended model.
		params.Fallbacks = anthropic.BetaFallbacksParamOfDefault()
		params.Betas = []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01}
	}
	msg, err := c.client.Beta.Messages.New(ctx, params)
	if err != nil {
		return Suggestion{}, fmt.Errorf("claude request: %w", err)
	}
	return parse(msg)
}

func parse(msg *anthropic.BetaMessage) (Suggestion, error) {
	switch msg.StopReason {
	case anthropic.BetaStopReasonEndTurn:
	case anthropic.BetaStopReasonRefusal:
		return Suggestion{}, fmt.Errorf("%w: category %q", ErrRefused, msg.StopDetails.Category)
	case anthropic.BetaStopReasonMaxTokens:
		return Suggestion{}, fmt.Errorf("%w: max_tokens reached", ErrIncomplete)
	default:
		return Suggestion{}, fmt.Errorf("%w: stop reason %q", ErrIncomplete, msg.StopReason)
	}
	var text strings.Builder
	for _, block := range msg.Content {
		if t, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	var s Suggestion
	if err := json.Unmarshal([]byte(text.String()), &s); err != nil {
		return Suggestion{}, fmt.Errorf("%w: %v", ErrIncomplete, err)
	}
	s.Name, s.Description = strings.TrimSpace(s.Name), strings.TrimSpace(s.Description)
	if s.Name == "" {
		return Suggestion{}, fmt.Errorf("%w: empty name", ErrIncomplete)
	}
	return s, nil
}
