package tgkit_test

import (
	"context"
	"fmt"

	telegram "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/marsolab/servekit/tgkit"
)

func Example() {
	handler := func(ctx context.Context, bot *telegram.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}

		_, _ = bot.SendMessage(ctx, &telegram.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Hello from ServeKit.",
		})
	}

	bot, err := tgkit.New(
		"123456:development-token",
		tgkit.WithSkipIdentityCheck(),
		tgkit.WithDefaultHandler(handler),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(bot.Mode())
	// Output: 0
}

func ExampleMiniAppLink() {
	link, err := tgkit.MiniAppLink(
		"@example_bot",
		"store",
		"campaign-42",
		tgkit.MiniAppModeCompact,
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(link)
	// Output: https://t.me/example_bot/store?startapp=campaign-42&mode=compact
}
