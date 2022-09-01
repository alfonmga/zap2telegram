package zap2telegram

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap/zapcore"
)

// telegramClient default options
var (
	defaultLoggerName          = "zap2telegram" // default logger name used by the default formatter in case of an unnamed Zap logger
	defaultDisableNotification = false          // enable Telegram message notification by default
)

// telegramCLient is a Telegram client
type telegramClient struct {
	botAPI                     *tgbotapi.BotAPI
	chatIDs                    []int64                                              // chat ids to send messages to
	disableNotification        bool                                                 // disable Telegram message notification
	enableNotificationOnLevels []zapcore.Level                                      // enable Telegram message notification on specified levels
	parseMode                  *string                                              // parse mode for Telegram message
	formatter                  func(e zapcore.Entry, fields []zapcore.Field) string // Telegram messages format
}

// newTelegramClient returns a new Telegram client with the specified options
func newTelegramClient(botAccesstoken string, chatIDs []int64) (*telegramClient, error) {
	bot, err := tgbotapi.NewBotAPI(botAccesstoken)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new Telegram bot API instance: %w", err)
	}
	return &telegramClient{
		botAPI:              bot,
		chatIDs:             chatIDs,
		disableNotification: defaultDisableNotification,
	}, nil
}

// Logger: zap2telegram
// 11:25:59 01.01.2007
// info
// Hello bar
func (c *telegramClient) formatMessage(e zapcore.Entry, fields []zapcore.Field) string {
	if c.formatter != nil {
		return c.formatter(e, fields)
	}
	loggerName := defaultLoggerName
	if e.LoggerName != "" {
		loggerName = e.LoggerName
	}
	return fmt.Sprintf("Logger: %s\n%s\n%s\n%s", loggerName, e.Time, e.Level, e.Message)
}

// sendMessage sends a message all specified chat ids
func (c *telegramClient) sendMessage(e zapcore.Entry, fields []zapcore.Field) error {
	for _, chatID := range c.chatIDs {
		msg := tgbotapi.NewMessage(chatID, c.formatMessage(e, fields))
		msg.DisableNotification = c.disableNotification
		if len(c.enableNotificationOnLevels) > 0 {
			for _, level := range c.enableNotificationOnLevels {
				if e.Level == level {
					msg.DisableNotification = false // enable notification for this message
					break
				}
			}
		}
		if c.parseMode != nil {
			msg.ParseMode = *c.parseMode
		}
		_, err := c.botAPI.Send(msg)
		if err != nil {
			err := fmt.Errorf("failed to send message to chat %d: %w", chatID, err)
			log.Println(err) // FIXME: how to log this error without using the default logger and avoid infinite recursion?
			return err
		}
	}
	return nil
}
