package idkit

import (
	"errors"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/marsolab/servekit/errkit"
	"github.com/maxatome/go-testdeep/td"
	"github.com/rs/xid"
)

func TestULID(t *testing.T) {
	t.Run("ULID", func(t *testing.T) {
		got := ULID()
		if len(got) != 26 {
			t.Errorf("the len of ULID() = %v, doesn't equal to 26 characters", got)
		}
	})
}

func TestValidateULID(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		id := ULID()
		if err := ValidateULID(id); err != nil {
			t.Errorf("ULID should be valid but its not: %v", err)
		}
	})

	t.Run("Error", func(t *testing.T) {
		id := "invalid id"
		if err := ValidateULID(id); err == nil || !errors.Is(err, errkit.ErrInvalidID) {
			t.Errorf("ULID should not be valid. Expected ErrInvalidID")
		}
	})
}

func TestDigiCode(t *testing.T) {
	t.Run("DigiCode", func(t *testing.T) {
		got := DigiCode()
		if len(got) != 6 || utf8.RuneCountInString(got) != 6 {
			t.Errorf("invalid digicode length: %d", len(got))
		}

		for _, r := range got {
			if !unicode.IsNumber(r) {
				t.Errorf("digicode contains char which is not a number: %s", string(r))
			}
		}
	})
}

func TestValidateDigiCode(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		code := DigiCode()
		if err := ValidateDigiCode(code); err != nil {
			t.Errorf("digicode should be valid but its not: %v", err)
		}
	})

	t.Run("Error len", func(t *testing.T) {
		id := "65789"
		if err := ValidateDigiCode(id); err == nil || !errors.Is(err, errkit.ErrInvalidID) {
			t.Errorf("digicode should not be valid. Expected ErrInvalidID")
		}
	})

	t.Run("Error invalid char", func(t *testing.T) {
		id := "65789C"
		if err := ValidateDigiCode(id); err == nil || !errors.Is(err, errkit.ErrInvalidID) {
			t.Errorf("digicode should not be valid. Expected ErrInvalidID")
		}
	})
}

func TestNewULID(t *testing.T) {
	// NewULID should return a valid ULID string and no error.
	id, err := NewULID()
	td.CmpNoError(t, err)
	td.Cmp(t, len(id), 26, "ULID should be 26 characters long.")

	// The returned ULID should pass validation.
	td.CmpNoError(t, ValidateULID(id))
}

func TestNewULID_Uniqueness(t *testing.T) {
	// Two calls to NewULID should produce different identifiers.
	id1, err1 := NewULID()
	td.CmpNoError(t, err1)

	id2, err2 := NewULID()
	td.CmpNoError(t, err2)

	td.CmpNot(t, id1, id2, "Two generated ULIDs should be different.")
}

func TestXID(t *testing.T) {
	// XID should return a non-empty uppercase string.
	got := XID()
	td.Cmp(t, len(got) > 0, true, "XID should not be empty.")

	// XID should be all uppercase.
	for _, r := range got {
		if r >= 'a' && r <= 'z' {
			t.Errorf("XID should be uppercase, but got lowercase character: %s", string(r))
		}
	}
}

func TestXID_Uniqueness(t *testing.T) {
	// Two calls to XID should produce different identifiers.
	id1 := XID()
	id2 := XID()
	td.CmpNot(t, id1, id2, "Two generated XIDs should be different.")
}

func TestValidateXID(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		// A freshly generated XID should pass validation when lowercased.
		id := strings.ToLower(XID())
		err := ValidateXID(id)
		td.CmpNoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		// An invalid string should return ErrInvalidID.
		err := ValidateXID("not-a-valid-xid")
		td.CmpError(t, err)
		td.Cmp(t, errors.Is(err, errkit.ErrInvalidID), true)
	})

	t.Run("EmptyString", func(t *testing.T) {
		// An empty string should return ErrInvalidID.
		err := ValidateXID("")
		td.CmpError(t, err)
		td.Cmp(t, errors.Is(err, errkit.ErrInvalidID), true)
	})
}

func TestParseXID(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		// Generate a valid XID and parse it back using lowercase form.
		raw := strings.ToLower(XID())
		parsed, err := ParseXID(raw)
		td.CmpNoError(t, err)
		td.CmpNot(t, parsed, xid.ID{}, "Parsed XID should not be zero value.")
	})

	t.Run("Error", func(t *testing.T) {
		// Parsing an invalid string should return ErrInvalidID.
		_, err := ParseXID("invalid")
		td.CmpError(t, err)
		td.Cmp(t, errors.Is(err, errkit.ErrInvalidID), true)
	})

	t.Run("EmptyString", func(t *testing.T) {
		// Parsing an empty string should return ErrInvalidID.
		parsed, err := ParseXID("")
		td.CmpError(t, err)
		td.Cmp(t, errors.Is(err, errkit.ErrInvalidID), true)
		td.Cmp(t, parsed, xid.ID{}, "Parsed XID from empty string should be zero value.")
	})
}
