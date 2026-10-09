package mask

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMasks(t *testing.T) {
	assert.Equal(t, "a***@example.com", Email("ada@example.com"))
	assert.Equal(t, "***", Email("not-an-address"))
	assert.Equal(t, "", Email(""))
	assert.Equal(t, []string{"a***@x.io", "b***@y.io"}, Emails([]string{"ann@x.io", "bob@y.io"}))
	assert.Equal(t, "***4567", Phone("+15551234567"))
	assert.Equal(t, "****", Phone("123"))
	assert.Equal(t, "a***@example.com", Identifier("ada@example.com"))
	assert.Equal(t, "***4567", Identifier("+15551234567"))
	assert.Equal(t, "***", Identifier("adalovelace"))
}
