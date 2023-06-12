// Kudos to <github.com/strpc/zaptelegram> for the initial implementation
package zap2telegram

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"time"

	"go.uber.org/zap/zapcore"
)

// zap2telegram default options
const (
	defaultLevel    = zapcore.WarnLevel // send messages equal or above this level
	defaultAsyncOpt = true              // send messages asynchronously by default
	defaultQueueOpt = false             // disable queue by default
)

// All levels provided by zap
var AllLevels = [6]zapcore.Level{
	zapcore.DebugLevel,
	zapcore.InfoLevel,
	zapcore.WarnLevel,
	zapcore.ErrorLevel,
	zapcore.FatalLevel,
	zapcore.PanicLevel,
}

// Posible errors when creating a new Zap Core
var (
	ErrBotAccessToken = errors.New("bot access token not defined")
	ErrChatIDs        = errors.New("chat ids not defined")
	ErrAsyncOpt       = errors.New("async option not worked with queue option")
)

type TelegramCore struct {
	zapcore.Core
	// inheritedFields is a collection of fields that have been added to the logger
	// through the use of `.With()`. These fields should never be cleared after
	// logging a single entry.
	inheritedFields []zapcore.Field
	telegramClient  *telegramClient      // telegram client
	enabler         zapcore.LevelEnabler // only send message if level is in this list
	async           bool                 // send messages asynchronously
	queue           bool                 // use a queue to send messages
	intervalQueue   time.Duration        // queue interval between messages sending
	entriesChan     chan chanEntry       // channel to store messages in queue
}
type chanEntry struct {
	entry  zapcore.Entry
	fields []zapcore.Field
}

// NewTelegramCore returns a new zap2telegram instance configured with the given options
func NewTelegramCore(botAccessToken string, chatIDs []int64, opts ...Option) (zapcore.Core, error) {
	if botAccessToken == "" {
		return nil, ErrBotAccessToken
	} else if len(chatIDs) == 0 {
		return nil, ErrChatIDs
	}
	telegramClient, err := newTelegramClient(botAccessToken, chatIDs)
	if err != nil {
		return nil, err
	}
	c := &TelegramCore{
		inheritedFields: []zapcore.Field{},
		telegramClient:  telegramClient,
		enabler:         zap.NewAtomicLevelAt(defaultLevel),
		async:           defaultAsyncOpt,
		queue:           defaultQueueOpt,
	}
	// apply options
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *TelegramCore) Enabled(l zapcore.Level) bool {
	return c.enabler.Enabled(l)
}
func (c *TelegramCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}
func (c *TelegramCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	entryFields := append(fields, c.inheritedFields...) // fields passed for the current entry log entry + inherited fields
	if c.async {
		go func() {
			_ = c.telegramClient.sendMessage(entry, entryFields)
		}()
	} else if c.queue {
		c.entriesChan <- chanEntry{entry, entryFields}
	} else {
		// if async or queue option is not set, send message immediately synchronously (blocking)
		if err := c.telegramClient.sendMessage(entry, entryFields); err != nil {
			return err
		}
	}
	return nil
}
func (c *TelegramCore) With(fields []zapcore.Field) zapcore.Core {
	cloned := *c
	cloned.inheritedFields = append(cloned.inheritedFields, fields...)
	return &cloned
}
func (c *TelegramCore) Sync() error {
	if c.queue {
		c.handleNewQueueEntries()
	}
	return nil
}

// consumeEntriesQueue sends all the entries (messages) in the queue to telegram at the given interval
func (h TelegramCore) consumeEntriesQueue(ctx context.Context) error {
	ticker := time.NewTicker(h.intervalQueue)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.handleNewQueueEntries()
		case <-ctx.Done():
			h.handleNewQueueEntries()
			return ctx.Err()
		}
	}
}

// handleNewQueueEntries send all new message entries in queue to telegram
func (h TelegramCore) handleNewQueueEntries() {
	for len(h.entriesChan) > 0 {
		chanEntry := <-h.entriesChan
		_ = h.telegramClient.sendMessage(chanEntry.entry, chanEntry.fields)
	}
}

// getLevelThreshold returns all levels equal and above the given level
func getLevelThreshold(l zapcore.Level) []zapcore.Level {
	for i := range AllLevels {
		if AllLevels[i] == l {
			return AllLevels[i:]
		}
	}
	return []zapcore.Level{}
}
