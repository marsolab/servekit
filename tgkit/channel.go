package tgkit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	telegram "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	maxInviteLinkNameLength = 32
	maxInviteLinkMembers    = 99999

	// MonthlySubscriptionPeriod is Telegram's currently supported recurring
	// billing period for bot and channel subscriptions.
	MonthlySubscriptionPeriod = 30 * 24 * time.Hour

	// MaxStarsSubscriptionPrice is Telegram's maximum monthly subscription
	// price in Stars.
	MaxStarsSubscriptionPrice = 10000
)

var (
	// ErrInvalidChatID is returned when a channel chat ID is empty or unsupported.
	ErrInvalidChatID = errors.New("invalid telegram chat id")

	// ErrInvalidInviteLink is returned when invite-link settings violate the
	// Telegram Bot API constraints.
	ErrInvalidInviteLink = errors.New("invalid telegram invite link")

	// ErrInvalidSubscription is returned when subscription settings violate the
	// Telegram Bot API constraints.
	ErrInvalidSubscription = errors.New("invalid telegram subscription")

	// ErrRequestRejected is returned when Telegram responds with a false result
	// without a more specific API error.
	ErrRequestRejected = errors.New("telegram request rejected")
)

// Channel provides channel-scoped Telegram Bot API operations.
//
// The bot must be a member or administrator with the rights required by each
// operation. Telegram's Bot API does not create the channel chat itself.
type Channel struct {
	bot    *telegram.Bot
	chatID any
}

// InviteLink configures a regular channel invite link.
type InviteLink struct {
	Name               string
	ExpireAt           time.Time
	MemberLimit        int
	CreatesJoinRequest bool
}

// ChannelSubscription configures a paid monthly channel invite link.
type ChannelSubscription struct {
	Name       string
	PriceStars int
}

// Channel creates a helper scoped to a channel ID or @username.
func (b *Bot) Channel(chatID any) (*Channel, error) {
	if err := validateChatID(chatID); err != nil {
		return nil, err
	}

	return &Channel{
		bot:    b.Bot,
		chatID: chatID,
	}, nil
}

// ID returns the channel's Telegram chat ID or @username.
func (c *Channel) ID() any {
	return c.chatID
}

// Send sends a message to the channel. The channel ID in params is ignored.
func (c *Channel) Send(ctx context.Context, params telegram.SendMessageParams) (*models.Message, error) {
	params.ChatID = c.chatID

	message, err := c.bot.SendMessage(ctx, &params)
	if err != nil {
		return nil, fmt.Errorf("send Telegram channel message: %w", err)
	}

	return message, nil
}

// DeleteMessage deletes a message from the channel.
func (c *Channel) DeleteMessage(ctx context.Context, messageID int) error {
	ok, err := c.bot.DeleteMessage(ctx, &telegram.DeleteMessageParams{
		ChatID:    c.chatID,
		MessageID: messageID,
	})

	return checkTelegramResult("delete Telegram channel message", telegramResult(ok), err)
}

// SetTitle changes the channel title.
func (c *Channel) SetTitle(ctx context.Context, title string) error {
	ok, err := c.bot.SetChatTitle(ctx, &telegram.SetChatTitleParams{
		ChatID: c.chatID,
		Title:  title,
	})

	return checkTelegramResult("set Telegram channel title", telegramResult(ok), err)
}

// SetDescription changes the channel description.
func (c *Channel) SetDescription(ctx context.Context, description string) error {
	ok, err := c.bot.SetChatDescription(ctx, &telegram.SetChatDescriptionParams{
		ChatID:      c.chatID,
		Description: description,
	})

	return checkTelegramResult("set Telegram channel description", telegramResult(ok), err)
}

// PinMessage pins a channel message.
func (c *Channel) PinMessage(ctx context.Context, messageID int, disableNotification bool) error {
	ok, err := c.bot.PinChatMessage(ctx, &telegram.PinChatMessageParams{
		ChatID:              c.chatID,
		MessageID:           messageID,
		DisableNotification: disableNotification,
	})

	return checkTelegramResult("pin Telegram channel message", telegramResult(ok), err)
}

// UnpinMessage unpins a channel message.
func (c *Channel) UnpinMessage(ctx context.Context, messageID int) error {
	ok, err := c.bot.UnpinChatMessage(ctx, &telegram.UnpinChatMessageParams{
		ChatID:    c.chatID,
		MessageID: messageID,
	})

	return checkTelegramResult("unpin Telegram channel message", telegramResult(ok), err)
}

// Info returns full information about the channel.
func (c *Channel) Info(ctx context.Context) (*models.ChatFullInfo, error) {
	info, err := c.bot.GetChat(ctx, &telegram.GetChatParams{ChatID: c.chatID})
	if err != nil {
		return nil, fmt.Errorf("get Telegram channel: %w", err)
	}

	return info, nil
}

// Administrators returns the channel's administrators.
func (c *Channel) Administrators(ctx context.Context, includeBots bool) ([]models.ChatMember, error) {
	admins, err := c.bot.GetChatAdministrators(ctx, &telegram.GetChatAdministratorsParams{
		ChatID:     c.chatID,
		ReturnBots: &includeBots,
	})
	if err != nil {
		return nil, fmt.Errorf("get Telegram channel administrators: %w", err)
	}

	return admins, nil
}

// MemberCount returns the number of channel members.
func (c *Channel) MemberCount(ctx context.Context) (int, error) {
	count, err := c.bot.GetChatMemberCount(ctx, &telegram.GetChatMemberCountParams{ChatID: c.chatID})
	if err != nil {
		return 0, fmt.Errorf("get Telegram channel member count: %w", err)
	}

	return count, nil
}

// Member returns a channel member by Telegram user ID.
func (c *Channel) Member(ctx context.Context, userID int64) (*models.ChatMember, error) {
	member, err := c.bot.GetChatMember(ctx, &telegram.GetChatMemberParams{
		ChatID: c.chatID,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("get Telegram channel member: %w", err)
	}

	return member, nil
}

// CreateInviteLink creates an additional channel invite link.
func (c *Channel) CreateInviteLink(ctx context.Context, invite InviteLink) (*models.ChatInviteLink, error) {
	if err := invite.validate(); err != nil {
		return nil, err
	}

	var expireDate int
	if !invite.ExpireAt.IsZero() {
		expireDate = int(invite.ExpireAt.Unix())
	}

	link, err := c.bot.CreateChatInviteLink(ctx, &telegram.CreateChatInviteLinkParams{
		ChatID:             c.chatID,
		Name:               invite.Name,
		ExpireDate:         expireDate,
		MemberLimit:        invite.MemberLimit,
		CreatesJoinRequest: invite.CreatesJoinRequest,
	})
	if err != nil {
		return nil, fmt.Errorf("create Telegram channel invite link: %w", err)
	}

	return link, nil
}

// CreateSubscriptionInviteLink creates a paid monthly channel invite link.
func (c *Channel) CreateSubscriptionInviteLink(
	ctx context.Context,
	subscription ChannelSubscription,
) (*models.ChatInviteLink, error) {
	if err := subscription.validate(); err != nil {
		return nil, err
	}

	link, err := c.bot.CreateChatSubscriptionInviteLink(
		ctx,
		&telegram.CreateChatSubscriptionInviteLinkParams{
			ChatID:             c.chatID,
			Name:               subscription.Name,
			SubscriptionPeriod: int(MonthlySubscriptionPeriod / time.Second),
			SubscriptionPrice:  subscription.PriceStars,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create Telegram channel subscription link: %w", err)
	}

	return link, nil
}

// EditSubscriptionInviteLink changes the display name of a paid channel invite
// link.
func (c *Channel) EditSubscriptionInviteLink(
	ctx context.Context,
	inviteLink string,
	name string,
) (*models.ChatInviteLink, error) {
	if strings.TrimSpace(inviteLink) == "" {
		return nil, fmt.Errorf("%w: empty link", ErrInvalidInviteLink)
	}

	if utf8.RuneCountInString(name) > maxInviteLinkNameLength {
		return nil, fmt.Errorf("%w: name exceeds %d characters", ErrInvalidInviteLink, maxInviteLinkNameLength)
	}

	link, err := c.bot.EditChatSubscriptionInviteLink(
		ctx,
		&telegram.EditChatSubscriptionInviteLinkParams{
			ChatID:     c.chatID,
			InviteLink: inviteLink,
			Name:       name,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("edit Telegram channel subscription link: %w", err)
	}

	return link, nil
}

// RevokeInviteLink revokes an invite link created by the bot.
func (c *Channel) RevokeInviteLink(ctx context.Context, inviteLink string) (*models.ChatInviteLink, error) {
	if strings.TrimSpace(inviteLink) == "" {
		return nil, fmt.Errorf("%w: empty link", ErrInvalidInviteLink)
	}

	link, err := c.bot.RevokeChatInviteLink(ctx, &telegram.RevokeChatInviteLinkParams{
		ChatID:     c.chatID,
		InviteLink: inviteLink,
	})
	if err != nil {
		return nil, fmt.Errorf("revoke Telegram channel invite link: %w", err)
	}

	return link, nil
}

// ApproveJoinRequest approves a user's pending request to join the channel.
func (c *Channel) ApproveJoinRequest(ctx context.Context, userID int64) error {
	ok, err := c.bot.ApproveChatJoinRequest(ctx, &telegram.ApproveChatJoinRequestParams{
		ChatID: c.chatID,
		UserID: userID,
	})

	return checkTelegramResult("approve Telegram channel join request", telegramResult(ok), err)
}

// DeclineJoinRequest declines a user's pending request to join the channel.
func (c *Channel) DeclineJoinRequest(ctx context.Context, userID int64) error {
	ok, err := c.bot.DeclineChatJoinRequest(ctx, &telegram.DeclineChatJoinRequestParams{
		ChatID: c.chatID,
		UserID: userID,
	})

	return checkTelegramResult("decline Telegram channel join request", telegramResult(ok), err)
}

func (invite InviteLink) validate() error {
	if utf8.RuneCountInString(invite.Name) > maxInviteLinkNameLength {
		return fmt.Errorf("%w: name exceeds %d characters", ErrInvalidInviteLink, maxInviteLinkNameLength)
	}

	if invite.MemberLimit < 0 || invite.MemberLimit > maxInviteLinkMembers {
		return fmt.Errorf("%w: member limit must be between 1 and %d", ErrInvalidInviteLink, maxInviteLinkMembers)
	}

	if invite.CreatesJoinRequest && invite.MemberLimit != 0 {
		return fmt.Errorf("%w: join requests and a member limit are mutually exclusive", ErrInvalidInviteLink)
	}

	return nil
}

func (subscription ChannelSubscription) validate() error {
	if utf8.RuneCountInString(subscription.Name) > maxInviteLinkNameLength {
		return fmt.Errorf("%w: name exceeds %d characters", ErrInvalidSubscription, maxInviteLinkNameLength)
	}

	if subscription.PriceStars < 1 || subscription.PriceStars > MaxStarsSubscriptionPrice {
		return fmt.Errorf(
			"%w: price must be between 1 and %d Stars",
			ErrInvalidSubscription,
			MaxStarsSubscriptionPrice,
		)
	}

	return nil
}

func validateChatID(chatID any) error {
	switch value := chatID.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: empty value", ErrInvalidChatID)
		}
	case int:
		if value == 0 {
			return fmt.Errorf("%w: zero value", ErrInvalidChatID)
		}
	case int64:
		if value == 0 {
			return fmt.Errorf("%w: zero value", ErrInvalidChatID)
		}
	default:
		return fmt.Errorf("%w: unsupported type %T", ErrInvalidChatID, chatID)
	}

	return nil
}

type telegramResult bool

func checkTelegramResult(operation string, result telegramResult, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}

	if !result {
		return fmt.Errorf("%s: %w", operation, ErrRequestRejected)
	}

	return nil
}
