package model

import (
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
)

type RealtimeRequest struct {
	Model    string                         `json:"model"`
	Messages []smodel.ChatCompletionMessage `json:"messages"`
}

type RealtimeResponse struct {
	Type         string `json:"type"`
	EventId      string `json:"event_id"`
	ItemId       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	Text         string `json:"text"`
	Transcript   string `json:"transcript"`
	ResponseId   string `json:"response_id"`
	OutputIndex  int    `json:"output_index"`
	Delta        string `json:"delta"`
	AudioEndMs   int    `json:"audio_end_ms"`

	Item struct {
		Id      string `json:"id"`
		Object  string `json:"object"`
		Type    string `json:"type"`
		Status  string `json:"status"`
		Role    string `json:"role"`
		Content []struct {
			Type       string `json:"type"`
			Text       string `json:"text"`
			Transcript string `json:"transcript"`
		} `json:"content"`
		Name      string `json:"name"`
		CallId    string `json:"call_id"`
		Arguments any    `json:"arguments"`
	} `json:"item"`

	Part struct {
		Type       string `json:"type"`
		Text       string `json:"text"`
		Transcript string `json:"transcript"`
	} `json:"part"`

	Session struct {
		Id            string   `json:"id"`
		Object        string   `json:"object"`
		Model         string   `json:"model"`
		ExpiresAt     int      `json:"expires_at"`
		Modalities    []string `json:"modalities"`
		Instructions  string   `json:"instructions"`
		Voice         string   `json:"voice"`
		TurnDetection struct {
			Type              string  `json:"type"`
			Threshold         float64 `json:"threshold"`
			PrefixPaddingMs   int     `json:"prefix_padding_ms"`
			SilenceDurationMs int     `json:"silence_duration_ms"`
		} `json:"turn_detection"`
		InputAudioFormat        string  `json:"input_audio_format"`
		OutputAudioFormat       string  `json:"output_audio_format"`
		InputAudioTranscription any     `json:"input_audio_transcription"`
		ToolChoice              string  `json:"tool_choice"`
		Temperature             float64 `json:"temperature"`
		MaxResponseOutputTokens any     `json:"max_response_output_tokens"`
		Tools                   []any   `json:"tools"`
	} `json:"session"`

	Response struct {
		Object        string `json:"object"`
		Id            string `json:"id"`
		Status        string `json:"status"`
		StatusDetails any    `json:"status_details"`
		Output        []struct {
			Id      string `json:"id"`
			Object  string `json:"object"`
			Type    string `json:"type"`
			Status  string `json:"status"`
			Role    string `json:"role"`
			Content []struct {
				Type       string `json:"type"`
				Text       string `json:"text"`
				Transcript string `json:"transcript"`
			} `json:"content"`
			Name      string `json:"name"`
			CallId    string `json:"call_id"`
			Arguments any    `json:"arguments"`
		} `json:"output"`
		Usage struct {
			TotalTokens       int `json:"total_tokens"`
			InputTokens       int `json:"input_tokens"`
			OutputTokens      int `json:"output_tokens"`
			InputTokenDetails struct {
				CachedTokens int `json:"cached_tokens"`
				TextTokens   int `json:"text_tokens"`
				AudioTokens  int `json:"audio_tokens"`
			} `json:"input_token_details"`
			OutputTokenDetails struct {
				TextTokens  int `json:"text_tokens"`
				AudioTokens int `json:"audio_tokens"`
			} `json:"output_token_details"`
		} `json:"usage"`
	} `json:"response"`

	RateLimits []struct {
		Name         string  `json:"name"`
		Limit        int     `json:"limit"`
		Remaining    int     `json:"remaining"`
		ResetSeconds float64 `json:"reset_seconds"`
	} `json:"rate_limits"`

	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Param   any    `json:"param"`
		EventId any    `json:"event_id"`
	} `json:"error"`
}
