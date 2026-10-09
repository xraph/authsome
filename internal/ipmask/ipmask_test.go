package ipmask

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetwork(t *testing.T) {
	assert.Equal(t, "203.0.113.0/24", Network("203.0.113.77"))
	assert.Equal(t, "203.0.113.0/24", Network("::ffff:203.0.113.77"), "a mapped IPv4 is masked as IPv4")
	assert.Equal(t, "2001:db8:1:2::/64", Network("2001:db8:1:2:3:4:5:6"))
	assert.Equal(t, "203.0.113.0/24", Network("203.0.113.0/24"), "masking is idempotent")
	assert.Equal(t, "", Network("not-an-address"))
	assert.Equal(t, "", Network(""))
}
