package suggest_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"

	"inventar/internal/suggest"
)

// fakeAPI answers like the Messages API with the given body and records the request.
type fakeAPI struct {
	header http.Header
	body   map[string]any
}

func (f *fakeAPI) start(t *testing.T, answer string) *suggest.Claude {
	return f.startWith(t, "", answer)
}

func (f *fakeAPI) startWith(t *testing.T, model, answer string) *suggest.Claude {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.header = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &f.body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, answer)
	}))
	t.Cleanup(srv.Close)
	return suggest.NewClaude("test-key", model, option.WithBaseURL(srv.URL), option.WithMaxRetries(0))
}

func message(stopReason, text string) string {
	content, _ := json.Marshal([]map[string]any{{"type": "text", "text": text}})
	return `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5",` +
		`"stop_reason":"` + stopReason + `","content":` + string(content) +
		`,"usage":{"input_tokens":1,"output_tokens":1}}`
}

func TestRequestUsesDefaultModelEffortAndSchema(t *testing.T) {
	var api fakeAPI
	c := api.start(t, message("end_turn", `{"name":" Stehlampe Messing ","description":"Mit rundem Fuß."}`))

	s, err := c.Suggest(context.Background(), []byte("jpeg"))
	if err != nil {
		t.Fatal(err)
	}
	if s != (suggest.Suggestion{Name: "Stehlampe Messing", Description: "Mit rundem Fuß."}) {
		t.Fatalf("suggestion %+v", s)
	}

	// Claude Haiku 5.5 has no server-side fallback, so neither the parameter nor the beta is sent.
	if api.header.Get("X-Api-Key") != "test-key" || api.header.Get("Anthropic-Beta") != "" {
		t.Errorf("headers: key %q, beta %q", api.header.Get("X-Api-Key"), api.header.Get("Anthropic-Beta"))
	}
	b := api.body
	if b["model"] != "claude-haiku-5-5" || b["fallbacks"] != nil || b["thinking"] != nil || b["tool_choice"] != nil {
		t.Errorf("model %v, fallbacks %v, thinking %v, tool_choice %v", b["model"], b["fallbacks"], b["thinking"], b["tool_choice"])
	}
	oc, _ := b["output_config"].(map[string]any)
	format, _ := oc["format"].(map[string]any)
	schema, _ := format["schema"].(map[string]any)
	if oc["effort"] != "low" || format["type"] != "json_schema" || schema["additionalProperties"] != false {
		t.Errorf("output_config %v", oc)
	}
	content := b["messages"].([]any)[0].(map[string]any)["content"].([]any)
	image := content[0].(map[string]any)["source"].(map[string]any)
	if image["type"] != "base64" || image["media_type"] != "image/jpeg" || image["data"] != "anBlZw==" {
		t.Errorf("image source %v", image)
	}
}

func TestModelsWithFallbackOptIntoDefaultFallbacks(t *testing.T) {
	for _, model := range []string{"claude-opus-5-5", "claude-opus-5", "claude-fable-5-1", "claude-sonnet-5-5"} {
		var api fakeAPI
		c := api.startWith(t, model, message("end_turn", `{"name":"Lampe","description":""}`))
		if _, err := c.Suggest(context.Background(), []byte("jpeg")); err != nil {
			t.Fatal(err)
		}
		if api.body["model"] != model || api.body["fallbacks"] != "default" || api.header.Get("Anthropic-Beta") != "server-side-fallback-2026-07-01" {
			t.Errorf("%s: model %v, fallbacks %v, beta %q", model, api.body["model"], api.body["fallbacks"], api.header.Get("Anthropic-Beta"))
		}
	}
}

func TestRefusalAndCutOffAnswersAreErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		answer string
		want   error
	}{
		"refusal":    {message("refusal", ""), suggest.ErrRefused},
		"max tokens": {message("max_tokens", `{"name":"Steh`), suggest.ErrIncomplete},
		"no JSON":    {message("end_turn", "Eine Lampe"), suggest.ErrIncomplete},
		"empty name": {message("end_turn", `{"name":"","description":""}`), suggest.ErrIncomplete},
	} {
		var api fakeAPI
		_, err := api.start(t, tc.answer).Suggest(context.Background(), []byte("jpeg"))
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: error %v, want %v", name, err, tc.want)
		}
	}
}
