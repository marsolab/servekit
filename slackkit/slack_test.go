package slackkit

import (
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestError_Error(t *testing.T) {
	f := func(name string, err Error, want string) {
		t.Helper()

		t.Run(name, func(t *testing.T) {
			td.Cmp(t, err.Error(), want)
		})
	}

	f("ErrNilBlock", ErrNilBlock, "nil block")
	f("ErrNilText", ErrNilText, "nil text")
	f("ErrInvalidBlockType", ErrInvalidBlockType, "invalid block type")
	f("ErrInvalidBlockTextType", ErrInvalidBlockTextType, "invalid block text type")
	f("ErrInvalidBlockTextStyle", ErrInvalidBlockTextStyle, "invalid block text style")
	f("ErrInvalidBlockTextEmoji", ErrInvalidBlockTextEmoji, "invalid block text emoji")
	f("Custom", Error("test error"), "test error")
}

func TestNewNotification(t *testing.T) {
	td.NewT(t)

	t.Run("empty blocks", func(t *testing.T) {
		notif, err := NewNotification()
		td.CmpNil(t, err)
		td.Cmp(t, notif.Blocks, []Block(nil))
	})

	t.Run("single valid header block", func(t *testing.T) {
		block := NewHeader("Test Header", true)
		notif, err := NewNotification(block)
		td.CmpNil(t, err)
		td.Cmp(t, notif.Blocks, []Block{block})
	})

	t.Run("multiple valid blocks", func(t *testing.T) {
		header1 := NewHeader("Header 1", true)
		header2 := NewHeader("Header 2", false)
		notif, err := NewNotification(header1, header2)
		td.CmpNil(t, err)
		td.Cmp(t, notif.Blocks, []Block{header1, header2})
	})

	t.Run("nil block", func(t *testing.T) {
		var nilBlock Block
		notif, err := NewNotification(nilBlock)
		td.Cmp(t, err, td.NotNil())
		td.Cmp(t, notif, td.Nil())
		td.Cmp(t, err.Error(), td.Contains("block 0 is invalid"))
		td.Cmp(t, err.Error(), td.Contains("invalid block type"))
	})

	t.Run("invalid block type", func(t *testing.T) {
		block := Block{Type: BlockType("invalid")}
		notif, err := NewNotification(block)
		td.Cmp(t, err, td.NotNil())
		td.Cmp(t, notif, td.Nil())
		td.Cmp(t, err.Error(), td.Contains("block 0 is invalid"))
		td.Cmp(t, err.Error(), td.Contains("invalid block type"))
	})

	t.Run("header block with nil text", func(t *testing.T) {
		block := Block{Type: Header, Text: nil}
		notif, err := NewNotification(block)
		td.Cmp(t, err, td.NotNil())
		td.Cmp(t, notif, td.Nil())
		td.Cmp(t, err.Error(), td.Contains("block 0 is invalid"))
		td.Cmp(t, err.Error(), td.Contains("nil text"))
	})

	t.Run("header block with empty text", func(t *testing.T) {
		block := Block{
			Type: Header,
			Text: &Text{
				Type: PlainText,
				Text: "",
			},
		}
		notif, err := NewNotification(block)
		td.Cmp(t, err, td.NotNil())
		td.Cmp(t, notif, td.Nil())
		td.Cmp(t, err.Error(), td.Contains("block 0 is invalid"))
		td.Cmp(t, err.Error(), td.Contains("nil text"))
	})

	t.Run("header block with invalid text type", func(t *testing.T) {
		block := Block{
			Type: Header,
			Text: &Text{
				Type: Markdown,
				Text: "Header Text",
			},
		}
		notif, err := NewNotification(block)
		td.Cmp(t, err, td.NotNil())
		td.Cmp(t, notif, td.Nil())
		td.Cmp(t, err.Error(), td.Contains("block 0 is invalid"))
		td.Cmp(t, err.Error(), td.Contains("invalid block text type"))
	})

	t.Run("header block with style", func(t *testing.T) {
		bold := true
		block := Block{
			Type: Header,
			Text: &Text{
				Type:  PlainText,
				Text:  "Header Text",
				Style: &Style{Bold: &bold},
			},
		}
		notif, err := NewNotification(block)
		td.Cmp(t, err, td.NotNil())
		td.Cmp(t, notif, td.Nil())
		td.Cmp(t, err.Error(), td.Contains("block 0 is invalid"))
		td.Cmp(t, err.Error(), td.Contains("invalid block text style"))
	})

	t.Run("multiple blocks with one invalid", func(t *testing.T) {
		validBlock := NewHeader("Valid", true)
		invalidBlock := Block{Type: BlockType("invalid")}
		notif, err := NewNotification(validBlock, invalidBlock)
		td.Cmp(t, err, td.NotNil())
		td.Cmp(t, notif, td.Nil())
		td.Cmp(t, err.Error(), td.Contains("block 1 is invalid"))
	})

	t.Run("divider block", func(t *testing.T) {
		divider := NewDivider()
		notif, err := NewNotification(divider)
		td.CmpNil(t, err)
		td.Cmp(t, notif, td.NotNil())
		td.Cmp(t, len(notif.Blocks), 1)
	})

	t.Run("section block", func(t *testing.T) {
		section := NewSection("Section", false)
		notif, err := NewNotification(section)
		td.CmpNil(t, err)
		td.Cmp(t, notif, td.NotNil())
		td.Cmp(t, len(notif.Blocks), 1)
	})
}

func TestBlock_validate(t *testing.T) {
	td.NewT(t)

	t.Run("nil block", func(t *testing.T) {
		var block *Block
		err := block.validate()
		td.Cmp(t, err, ErrNilBlock)
	})

	t.Run("invalid block type", func(t *testing.T) {
		block := Block{Type: BlockType("invalid")}
		err := block.validate()
		td.Cmp(t, err, ErrInvalidBlockType)
	})

	t.Run("header block with nil text", func(t *testing.T) {
		block := Block{Type: Header, Text: nil}
		err := block.validate()
		td.Cmp(t, err, ErrNilText)
	})

	t.Run("header block with empty text", func(t *testing.T) {
		block := Block{
			Type: Header,
			Text: &Text{
				Type: PlainText,
				Text: "",
			},
		}
		err := block.validate()
		td.Cmp(t, err, ErrNilText)
	})

	t.Run("header block with invalid text type", func(t *testing.T) {
		block := Block{
			Type: Header,
			Text: &Text{
				Type: Markdown,
				Text: "Header Text",
			},
		}
		err := block.validate()
		td.Cmp(t, err, ErrInvalidBlockTextType)
	})

	t.Run("header block with style", func(t *testing.T) {
		bold := true
		block := Block{
			Type: Header,
			Text: &Text{
				Type:  PlainText,
				Text:  "Header Text",
				Style: &Style{Bold: &bold},
			},
		}
		err := block.validate()
		td.Cmp(t, err, ErrInvalidBlockTextStyle)
	})

	t.Run("valid header block", func(t *testing.T) {
		block := NewHeader("Valid Header", true)
		err := block.validate()
		td.CmpNil(t, err)
	})

	t.Run("valid header block without emoji", func(t *testing.T) {
		block := NewHeader("Valid Header", false)
		err := block.validate()
		td.CmpNil(t, err)
	})
}

func TestNewHeader(t *testing.T) {
	td.NewT(t)

	t.Run("with emoji", func(t *testing.T) {
		block := NewHeader("Test Header", true)
		td.Cmp(t, block.Type, Header)
		td.Cmp(t, block.Text, td.NotNil())
		td.Cmp(t, block.Text.Type, PlainText)
		td.Cmp(t, block.Text.Text, "Test Header")
		td.Cmp(t, block.Text.Emoji, td.Ptr(true))
		td.Cmp(t, block.Text.Style, td.Nil())
	})

	t.Run("without emoji", func(t *testing.T) {
		block := NewHeader("Test Header", false)
		td.Cmp(t, block.Type, Header)
		td.Cmp(t, block.Text, td.NotNil())
		td.Cmp(t, block.Text.Type, PlainText)
		td.Cmp(t, block.Text.Text, "Test Header")
		td.Cmp(t, block.Text.Emoji, td.Ptr(false))
		td.Cmp(t, block.Text.Style, td.Nil())
	})

	t.Run("empty text", func(t *testing.T) {
		block := NewHeader("", true)
		td.Cmp(t, block.Type, Header)
		td.Cmp(t, block.Text, td.NotNil())
		td.Cmp(t, block.Text.Text, "")
	})
}

func TestNewDivider(t *testing.T) {
	td.NewT(t)

	block := NewDivider()
	td.Cmp(t, block.Type, Divider)
	td.Cmp(t, block.Text, td.Nil())
}

func TestNewSection(t *testing.T) {
	td.NewT(t)

	t.Run("with emoji", func(t *testing.T) {
		block := NewSection("Test Section", true)
		td.Cmp(t, block.Type, Section)
		td.Cmp(t, block.Text, td.NotNil())
		td.Cmp(t, block.Text.Type, PlainText)
		td.Cmp(t, block.Text.Text, "Test Section")
		td.Cmp(t, block.Text.Emoji, td.Ptr(true))
		td.Cmp(t, block.Text.Style, td.Nil())
	})

	t.Run("without emoji", func(t *testing.T) {
		block := NewSection("Test Section", false)
		td.Cmp(t, block.Type, Section)
		td.Cmp(t, block.Text, td.NotNil())
		td.Cmp(t, block.Text.Type, PlainText)
		td.Cmp(t, block.Text.Text, "Test Section")
		td.Cmp(t, block.Text.Emoji, td.Ptr(false))
		td.Cmp(t, block.Text.Style, td.Nil())
	})

	t.Run("empty text", func(t *testing.T) {
		block := NewSection("", true)
		td.Cmp(t, block.Type, Section)
		td.Cmp(t, block.Text, td.NotNil())
		td.Cmp(t, block.Text.Text, "")
	})
}
