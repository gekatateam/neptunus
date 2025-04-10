package logger

import (
	"context"
	"log/slog"
	"math"
)

var _ slog.Handler = (*DynamicLevelHandler)(nil)

type LevelOverride int

const (
	OverridePipeline LevelOverride = iota + 1
	OverrideCore
	OverrideKeykeeper
	OverrideInput
	OverrideProcessor
	OverrideOutput
	OverrideFilter
	OverrideParser
	OverrideSerializer
)

const LevelUnassigned slog.Level = math.MinInt

type DynamicLevelHandler struct {
	basic slog.Handler

	assignedLevel  slog.Level
	levelsHandlers map[slog.Level]slog.Handler
}

func (h *DynamicLevelHandler) Override(newLevel slog.Level) slog.Handler {
	return &DynamicLevelHandler{
		basic:          h.basic,
		assignedLevel:  newLevel,
		levelsHandlers: h.levelsHandlers,
	}
}

func (h *DynamicLevelHandler) Handle(ctx context.Context, record slog.Record) error {
	if h.assignedLevel == LevelUnassigned {
		return h.basic.Handle(ctx, record)
	}

	return h.levelsHandlers[record.Level].Handle(ctx, record)
}

func (h *DynamicLevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.assignedLevel == LevelUnassigned {
		return h.basic.Enabled(ctx, level)
	}

	return level >= h.assignedLevel
}

func (h *DynamicLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var handlers = make(map[slog.Level]slog.Handler)
	for l := range h.levelsHandlers {
		handlers[l] = handlers[l].WithAttrs(attrs)
	}

	return &DynamicLevelHandler{
		basic:          h.basic.WithAttrs(attrs),
		assignedLevel:  h.assignedLevel,
		levelsHandlers: handlers,
	}
}

func (h *DynamicLevelHandler) WithGroup(name string) slog.Handler {
	var handlers = make(map[slog.Level]slog.Handler)
	for l := range h.levelsHandlers {
		handlers[l] = handlers[l].WithGroup(name)
	}

	return &DynamicLevelHandler{
		basic:          h.basic.WithGroup(name),
		assignedLevel:  h.assignedLevel,
		levelsHandlers: handlers,
	}
}
