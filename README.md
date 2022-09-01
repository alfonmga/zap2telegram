# zap2telegram

[![Go Reference](https://pkg.go.dev/badge/github.com/alfonmga/zap2telegram.svg)](https://pkg.go.dev/github.com/alfonmga/zap2telegram)

`zap2telegram` is a fantastic way to centralize your program's [zap](https://github.com/uber-go/zap) logs by sending them to a Telegram chat.

![Screenshot 2022-09-01 at 18 48 06](https://user-images.githubusercontent.com/9363272/187970432-617fd4ad-5e65-49ab-aadb-0248b05e6637.png)

## Install

Via go get tool

```bash
$ go get -u github.com/alfonmga/zap2telegram
```

## Usage

Example usage:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alfonmga/zap2telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const appName = "acme-api"

const (
	tgBotAccessToken = "<telegram-bot-access-token>"
	tgChatID         = -1
	tgMsgParseMode   = "MarkdownV2"
)

func main() {
	zap2telegramCore, err := zap2telegram.NewTelegramCore(
		tgBotAccessToken,
		[]int64{tgChatID},
		zap2telegram.WithLevel(zapcore.InfoLevel), // Send only Info and above logs to Telegram
		zap2telegram.WithNotificationOn([]zapcore.Level{zap.ErrorLevel, zap.PanicLevel, zap.FatalLevel}), // Enable message notification only this levels
		zap2telegram.WithQueue(context.Background(), 11*time.Second, 330),                                // Use queue to send messages to Telegram every 11 seconds and set queue size to 330 messages at most
		zap2telegram.WithParseMode(tgMsgParseMode),
		zap2telegram.WithFormatter(func(e zapcore.Entry, fields []zapcore.Field) string {
			escapedAppName := tgbotapi.EscapeText(tgMsgParseMode, appName)
			escapedLoggerName := tgbotapi.EscapeText(tgMsgParseMode, e.LoggerName)
			escapedCaller := tgbotapi.EscapeText(tgMsgParseMode, e.Caller.TrimmedPath())
			escapedMessage := tgbotapi.EscapeText(tgMsgParseMode, e.Message)
			// [INFO] acme-api
			// Logger: main
			// Caller: main.go:10
			// Message: Some message here
			// user_id=12345
			// Stacktrace: ...
			msg := fmt.Sprintf(
				"\\[%s\\] _%s_\nLogger: %s\nCaller: %s\nMessage: *%s*",
				strings.ToUpper(e.Level.String()),
				escapedAppName,
				escapedLoggerName,
				escapedCaller,
				escapedMessage,
			)
			// Add fields to the message
			msgFields := ""
			for _, field := range fields {
				enc := zapcore.NewMapObjectEncoder()
				field.AddTo(enc)
				for k, v := range enc.Fields {
					if k == "app_name" {
						continue // Skip app_name field because it is already in the message
					}
					escapedK := tgbotapi.EscapeText(tgMsgParseMode, k)
					escapedV := tgbotapi.EscapeText(tgMsgParseMode, fmt.Sprintf("%+v", v))
					msgField := fmt.Sprintf("%s\\=`%s`", escapedK, escapedV)
					if msgFields != "" {
						msgFields += " " // add leading space if there are already fields
					}
					msgFields += msgField
				}
			}
			if msgFields != "" {
				msg += fmt.Sprintf("\n%s", msgFields)
			}
			if e.Stack != "" {
				msg += fmt.Sprintf("\nLogger stacktrace: `%s`", tgbotapi.EscapeText(tgMsgParseMode, string(e.Stack)))
			}
			return msg
		}),
	)
	if err != nil {
		panic(fmt.Errorf("failed to initialize zap2telegramCore: %+v", err))
	}

	logger := zap.New(zapcore.NewTee(
		zap2telegramCore,
		// Console stderr output > warn level
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
			zapcore.Lock(os.Stderr),
			zap.LevelEnablerFunc(func(level zapcore.Level) bool {
				return level > zapcore.WarnLevel
			}),
		),
		// Console stdout output <= warn level
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
			zapcore.Lock(os.Stdout),
			zap.LevelEnablerFunc(func(level zapcore.Level) bool {
				return level <= zapcore.WarnLevel
			}),
		),
	)).WithOptions(zap.AddStacktrace(zap.ErrorLevel), zap.AddCaller()).With(zap.String("app_name", appName)).Named("main")
	defer logger.Sync() // send logs to Telegram before the program exit (this is only supported if you're using the `WithQueue` option). If you prefer, you can use the `WithoutAsyncOpt` option for synchronous sending (blocking)

	logger.Warn("take a look at this log message, something important may be happening!")
	logger.Error("something went wrong", zap.String("user_id", "12345"))
}
```
