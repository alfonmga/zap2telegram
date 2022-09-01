package zap2telegram

import (
	"context"
	"time"

	"go.uber.org/zap/zapcore"
)

type Option func(*TelegramCore) error

// WithLevel sends messages equal or above specified level
func WithLevel(l zapcore.Level) Option {
	return func(h *TelegramCore) error {
		levels := getLevelThreshold(l)
		h.levels = levels
		return nil
	}
}

// WithStrongLevel sends only messages with specified level
func WithStrongLevel(l zapcore.Level) Option {
	return func(h *TelegramCore) error {
		h.levels = []zapcore.Level{l}
		return nil
	}
}

// WithDisabledNotification disables Telegram message notification
func WithDisabledNotification() Option {
	return func(h *TelegramCore) error {
		h.telegramClient.disableNotification = true
		return nil
	}
}

// WithNotificationOn enables Telegram message notification on specified levels
func WithNotificationOn(levels []zapcore.Level) Option {
	return func(h *TelegramCore) error {
		h.telegramClient.disableNotification = true
		h.telegramClient.enableNotificationOnLevels = levels
		return nil
	}
}

// WithParseMode sets parse mode for Telegram messages
// (E.g: "ModeMarkdown", "ModeMarkdownV2" or "ModeHTML")
// https://core.telegram.org/bots/api#formatting-options
func WithParseMode(parseMode string) Option {
	return func(h *TelegramCore) error {
		h.telegramClient.parseMode = &parseMode
		return nil
	}
}

// WithFormatter sets a custom Telegram message formatter
func WithFormatter(f func(e zapcore.Entry, fields []zapcore.Field) string) Option {
	return func(h *TelegramCore) error {
		h.telegramClient.formatter = f
		return nil
	}
}

// WithoutAsyncOpt disables default asynchronous mode and enables synchronous mode for messages sending (blocking)
func WithoutAsyncOpt() Option {
	return func(h *TelegramCore) error {
		if h.queue {
			return ErrAsyncOpt
		}
		h.async = false
		return nil
	}
}

// WithQueue sends the messages to Telegram in batches (burst) at the specified interval
func WithQueue(ctx context.Context, interval time.Duration, queueSize int) Option {
	return func(h *TelegramCore) error {
		h.async = false
		h.queue = true
		h.intervalQueue = interval
		h.entriesChan = make(chan chanEntry, queueSize)
		go func() {
			_ = h.consumeEntriesQueue(ctx)
		}()
		return nil
	}
}
