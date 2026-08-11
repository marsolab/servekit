package tgkit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	telegram "github.com/go-telegram/bot"
)

func TestBotChannelValidatesChatID(t *testing.T) {
	t.Parallel()

	bot := &Bot{}
	tests := map[string]struct {
		chatID  any
		wantErr bool
	}{
		"numeric":          {chatID: int64(-100123)},
		"username":         {chatID: "@channel"},
		"empty username":   {chatID: " ", wantErr: true},
		"zero numeric":     {chatID: int64(0), wantErr: true},
		"unsupported type": {chatID: 1.25, wantErr: true},
		"nil":              {wantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			channel, err := bot.Channel(test.chatID)
			if test.wantErr {
				if !errors.Is(err, ErrInvalidChatID) {
					t.Fatalf("Channel() error = %v, want %v", err, ErrInvalidChatID)
				}

				return
			}
			if err != nil {
				t.Fatalf("Channel() error = %v", err)
			}
			if channel.ID() != test.chatID {
				t.Errorf("ID() = %v, want %v", channel.ID(), test.chatID)
			}
		})
	}
}

func TestChannelSendScopesMessageToChannel(t *testing.T) {
	t.Parallel()

	bot, requests := newTelegramTestBot(t, map[string]any{"message_id": 42})
	channel, err := bot.Channel(int64(-100123))
	if err != nil {
		t.Fatalf("Channel() error = %v", err)
	}

	message, err := channel.Send(context.Background(), telegram.SendMessageParams{
		ChatID: "@ignored",
		Text:   "Release shipped",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if message.ID != 42 {
		t.Errorf("message ID = %d, want 42", message.ID)
	}

	request := receiveTelegramRequest(t, requests)
	if request.path != "/bot"+testBotToken+"/sendMessage" {
		t.Errorf("request path = %q", request.path)
	}
	if got := request.values.Get("chat_id"); got != "-100123" {
		t.Errorf("chat_id = %q, want -100123", got)
	}
	if got := request.values.Get("text"); got != "Release shipped" {
		t.Errorf("text = %q, want Release shipped", got)
	}
}

func TestChannelCreateSubscriptionInviteLink(t *testing.T) {
	t.Parallel()

	bot, requests := newTelegramTestBot(t, map[string]any{"invite_link": "https://t.me/+paid"})
	channel, err := bot.Channel("@members")
	if err != nil {
		t.Fatalf("Channel() error = %v", err)
	}

	link, err := channel.CreateSubscriptionInviteLink(context.Background(), ChannelSubscription{
		Name:       "Gold",
		PriceStars: 250,
	})
	if err != nil {
		t.Fatalf("CreateSubscriptionInviteLink() error = %v", err)
	}
	if link.InviteLink != "https://t.me/+paid" {
		t.Errorf("invite link = %q", link.InviteLink)
	}

	request := receiveTelegramRequest(t, requests)
	if request.path != "/bot"+testBotToken+"/createChatSubscriptionInviteLink" {
		t.Errorf("request path = %q", request.path)
	}
	wants := map[string]string{
		"chat_id":             "@members",
		"name":                "Gold",
		"subscription_period": "2592000",
		"subscription_price":  "250",
	}
	for key, want := range wants {
		if got := request.values.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestChannelCreateInviteLink(t *testing.T) {
	t.Parallel()

	bot, requests := newTelegramTestBot(t, map[string]any{"invite_link": "https://t.me/+invite"})
	channel, err := bot.Channel("@members")
	if err != nil {
		t.Fatalf("Channel() error = %v", err)
	}
	expiresAt := time.Unix(2_000_000_000, 0)

	link, err := channel.CreateInviteLink(context.Background(), InviteLink{
		Name:        "Launch",
		ExpireAt:    expiresAt,
		MemberLimit: 100,
	})
	if err != nil {
		t.Fatalf("CreateInviteLink() error = %v", err)
	}
	if link.InviteLink != "https://t.me/+invite" {
		t.Errorf("invite link = %q", link.InviteLink)
	}

	request := receiveTelegramRequest(t, requests)
	if request.path != "/bot"+testBotToken+"/createChatInviteLink" {
		t.Errorf("request path = %q", request.path)
	}
	wants := map[string]string{
		"chat_id":      "@members",
		"name":         "Launch",
		"expire_date":  "2000000000",
		"member_limit": "100",
	}
	for key, want := range wants {
		if got := request.values.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestInviteLinkValidate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		invite  InviteLink
		wantErr bool
	}{
		"empty is valid": {},
		"expiring limited link": {
			invite: InviteLink{
				Name:        "Launch",
				ExpireAt:    time.Now().Add(time.Hour),
				MemberLimit: 100,
			},
		},
		"name too long": {
			invite:  InviteLink{Name: strings.Repeat("x", maxInviteLinkNameLength+1)},
			wantErr: true,
		},
		"limit too high": {
			invite:  InviteLink{MemberLimit: maxInviteLinkMembers + 1},
			wantErr: true,
		},
		"join request with limit": {
			invite: InviteLink{
				MemberLimit:        10,
				CreatesJoinRequest: true,
			},
			wantErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := test.invite.validate()
			if test.wantErr && !errors.Is(err, ErrInvalidInviteLink) {
				t.Fatalf("validate() error = %v, want %v", err, ErrInvalidInviteLink)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("validate() error = %v", err)
			}
		})
	}
}

func TestChannelSubscriptionValidate(t *testing.T) {
	t.Parallel()

	tests := map[string]ChannelSubscription{
		"zero":          {PriceStars: 0},
		"above maximum": {PriceStars: MaxStarsSubscriptionPrice + 1},
		"name too long": {
			Name:       strings.Repeat("x", maxInviteLinkNameLength+1),
			PriceStars: 1,
		},
	}

	for name, subscription := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := subscription.validate(); !errors.Is(err, ErrInvalidSubscription) {
				t.Fatalf("validate() error = %v, want %v", err, ErrInvalidSubscription)
			}
		})
	}
}

func TestCheckTelegramResultRejectsFalse(t *testing.T) {
	t.Parallel()

	err := checkTelegramResult("test operation", telegramResult(false), nil)
	if !errors.Is(err, ErrRequestRejected) {
		t.Fatalf("checkTelegramResult() error = %v, want %v", err, ErrRequestRejected)
	}
}
