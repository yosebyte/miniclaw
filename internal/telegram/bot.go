// MIT License - Copyright (c) 2026 yosebyte
package telegram

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/yosebyte/miniclaw/internal/agent"
	"github.com/yosebyte/miniclaw/internal/config"
)

// Bot is the Telegram long-polling bot.
type Bot struct {
	cfg    *config.Config
	loop   *agent.Loop
	api    *tgbotapi.BotAPI
	typing sync.Map // chat_id(int64) -> context.CancelFunc
}

// New creates a Bot.
func New(cfg *config.Config, loop *agent.Loop) *Bot {
	return &Bot{cfg: cfg, loop: loop}
}

// Run starts long polling and blocks until ctx is cancelled.
func (b *Bot) Run(ctx context.Context) error {
	token := b.cfg.Telegram.Token
	if token == "" {
		return fmt.Errorf("telegram.token not configured; run: miniclaw onboard")
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("creating telegram bot: %w", err)
	}
	b.api = api
	log.Printf("[INFO] telegram bot connected; username=%s", api.Self.UserName)

	cmds := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "start", Description: "Start the bot"},
		tgbotapi.BotCommand{Command: "new", Description: "Start a new conversation"},
		tgbotapi.BotCommand{Command: "help", Description: "Show available commands"},
	)
	if _, err := api.Request(cmds); err != nil {
		log.Printf("[WARN] could not set bot commands: %v", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	u.AllowedUpdates = []string{"message"}

	updates := api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			api.StopReceivingUpdates()
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message == nil {
				continue
			}
			go b.handleMessage(ctx, update.Message)
		}
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	user := msg.From
	if user == nil {
		return
	}

	if !b.isAllowed(user) {
		log.Printf("[WARN] message from unauthorised user; id=%d username=%s", user.ID, user.UserName)
		return
	}

	chatID := msg.Chat.ID
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}
	if text == "" {
		return
	}

	preview := text
	if len(preview) > 60 {
		preview = preview[:60] + "..."
	}
	log.Printf("[INFO] message; from=%d chat=%d preview=%q", user.ID, chatID, preview)

	if strings.ToLower(strings.TrimSpace(text)) == "/start" {
		reply := tgbotapi.NewMessage(chatID,
			fmt.Sprintf("👋 Hi %s! I'm miniclaw.\n\nSend me a message and I'll respond!\nType /help to see available commands.", user.FirstName))
		if _, err := b.api.Send(reply); err != nil {
			log.Printf("[WARN] send error: %v", err)
		}
		return
	}

	sessionKey := fmt.Sprintf("telegram_%d", chatID)

	typingCtx, typingCancel := context.WithCancel(ctx)
	defer typingCancel()
	b.stopTyping(chatID)
	b.typing.Store(chatID, typingCancel)
	go b.typingLoop(typingCtx, chatID)

	response, err := b.loop.ProcessMessage(ctx, sessionKey, text)

	typingCancel()
	b.typing.Delete(chatID)

	if err != nil {
		log.Printf("[ERROR] agent error: %v", err)
		response = "Sorry, I encountered an error: " + err.Error()
	}

	b.sendText(chatID, response)
}

func (b *Bot) typingLoop(ctx context.Context, chatID int64) {
	b.sendTyping(chatID)
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.sendTyping(chatID)
		}
	}
}

func (b *Bot) sendTyping(chatID int64) {
	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	if _, err := b.api.Request(action); err != nil {
		log.Printf("[DEBUG] typing action failed: %v", err)
	}
}

func (b *Bot) stopTyping(chatID int64) {
	if cancel, ok := b.typing.Load(chatID); ok {
		cancel.(context.CancelFunc)()
		b.typing.Delete(chatID)
	}
}

func (b *Bot) sendText(chatID int64, content string) {
	for _, chunk := range splitMessage(content, 4000) {
		html := markdownToHTML(chunk)
		m := tgbotapi.NewMessage(chatID, html)
		m.ParseMode = tgbotapi.ModeHTML
		if _, err := b.api.Send(m); err != nil {
			log.Printf("[WARN] HTML send failed, falling back to plain: %v", err)
			m2 := tgbotapi.NewMessage(chatID, chunk)
			if _, err2 := b.api.Send(m2); err2 != nil {
				log.Printf("[ERROR] send error: %v", err2)
			}
		}
	}
}

func (b *Bot) isAllowed(user *tgbotapi.User) bool {
	if len(b.cfg.Telegram.AllowFrom) == 0 {
		return true
	}
	userID := fmt.Sprintf("%d", user.ID)
	for _, allowed := range b.cfg.Telegram.AllowFrom {
		if allowed == userID || (user.UserName != "" && allowed == user.UserName) {
			return true
		}
	}
	return false
}

// ---- Markdown → Telegram HTML conversion ----

var (
	reCodeBlock  = regexp.MustCompile("(?s)```[\\w]*\n?([\\s\\S]*?)```")
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reHeader     = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
	reBlockquote = regexp.MustCompile(`(?m)^>\s*(.*)$`)
	reBold       = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reBoldUS     = regexp.MustCompile(`__(.+?)__`)
	reItalic     = regexp.MustCompile(`(?:^|[^a-zA-Z0-9])_([^_]+)_(?:[^a-zA-Z0-9]|$)`)
	reStrike     = regexp.MustCompile(`~~(.+?)~~`)
	reLink       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reBullet     = regexp.MustCompile(`(?m)^[-*]\s+`)
)

func markdownToHTML(text string) string {
	if text == "" {
		return ""
	}

	var codeBlocks []string
	text = reCodeBlock.ReplaceAllStringFunc(text, func(m string) string {
		sub := reCodeBlock.FindStringSubmatch(m)
		codeBlocks = append(codeBlocks, sub[1])
		return fmt.Sprintf("\x00CB%d\x00", len(codeBlocks)-1)
	})

	var inlineCodes []string
	text = reInlineCode.ReplaceAllStringFunc(text, func(m string) string {
		sub := reInlineCode.FindStringSubmatch(m)
		inlineCodes = append(inlineCodes, sub[1])
		return fmt.Sprintf("\x00IC%d\x00", len(inlineCodes)-1)
	})

	text = reHeader.ReplaceAllString(text, "$1")
	text = reBlockquote.ReplaceAllString(text, "$1")

	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")

	text = reLink.ReplaceAllString(text, `<a href="$2">$1</a>`)
	text = reBold.ReplaceAllString(text, "<b>$1</b>")
	text = reBoldUS.ReplaceAllString(text, "<b>$1</b>")
	text = reItalic.ReplaceAllStringFunc(text, func(m string) string {
		sub := reItalic.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		return strings.Replace(m, "_"+sub[1]+"_", "<i>"+sub[1]+"</i>", 1)
	})
	text = reStrike.ReplaceAllString(text, "<s>$1</s>")
	text = reBullet.ReplaceAllString(text, "• ")

	for i, code := range inlineCodes {
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00IC%d\x00", i),
			"<code>"+htmlEscape(code)+"</code>")
	}
	for i, code := range codeBlocks {
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00CB%d\x00", i),
			"<pre><code>"+htmlEscape(code)+"</code></pre>")
	}

	return text
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func splitMessage(content string, maxLen int) []string {
	if len(content) <= maxLen {
		return []string{content}
	}
	var chunks []string
	for len(content) > 0 {
		if len(content) <= maxLen {
			chunks = append(chunks, content)
			break
		}
		cut := content[:maxLen]
		pos := strings.LastIndexByte(cut, '\n')
		if pos == -1 {
			pos = strings.LastIndexByte(cut, ' ')
		}
		if pos == -1 {
			pos = maxLen
		}
		chunks = append(chunks, content[:pos])
		content = strings.TrimLeft(content[pos:], " \n")
	}
	return chunks
}
