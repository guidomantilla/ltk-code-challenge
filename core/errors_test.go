package core

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	t.Parallel()

	t.Run("NewError", func(t *testing.T) {
		t.Parallel()

		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		e := NewError("base message", err1, err2)

		assert.Equal(t, "base message", e.Message)
		assert.Equal(t, []string{"error 1", "error 2"}, e.Err)
		assert.Equal(t, []string{"error 1", "error 2"}, e.Messages())
	})

	t.Run("Error method", func(t *testing.T) {
		t.Parallel()

		e := NewError("test", errors.New("internal"))
		got := e.Error()
		assert.Contains(t, got, "test")
		assert.Contains(t, got, "internal")
	})

	t.Run("Unwrap", func(t *testing.T) {
		t.Parallel()

		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		e := NewError("base", err1, err2)

		unwrapped := e.Unwrap()
		require.Error(t, unwrapped)
		assert.Contains(t, unwrapped.Error(), "error 1")
		assert.Contains(t, unwrapped.Error(), "error 2")
	})

	t.Run("Unwrap nil or empty", func(t *testing.T) {
		t.Parallel()

		var e *Error
		require.NoError(t, e.Unwrap())

		e2 := &Error{Message: "no errors"}
		require.NoError(t, e2.Unwrap())
	})
}
