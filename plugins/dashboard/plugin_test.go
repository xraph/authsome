package dashboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_ID(t *testing.T) {
	p := NewPlugin()
	assert.Equal(t, "dashboard", p.ID())
}

func TestPlugin_Init(t *testing.T) {
	tests := []struct {
		name    string
		dep     interface{}
		wantErr bool
	}{
		{
			name:    "nil dependency",
			dep:     nil,
			wantErr: true,
		},
		{
			name:    "invalid dependency type",
			dep:     "invalid",
			wantErr: true,
		},
		// TODO: Add test with valid auth instance once we have a mock
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlugin()
			err := p.Init(tt.dep)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPlugin_RegisterHooks(t *testing.T) {
	p := NewPlugin()
	err := p.RegisterHooks(nil)
	assert.NoError(t, err, "RegisterHooks should not return an error for dashboard plugin")
}

func TestPlugin_RegisterServiceDecorators(t *testing.T) {
	p := NewPlugin()
	err := p.RegisterServiceDecorators(nil)
	assert.NoError(t, err, "RegisterServiceDecorators should not return an error for dashboard plugin")
}

func TestPlugin_Migrate(t *testing.T) {
	p := NewPlugin()
	err := p.Migrate()
	assert.NoError(t, err, "Migrate should not return an error for dashboard plugin")
}

func TestTemplateFuncs(t *testing.T) {
	funcs := templateFuncs()

	t.Run("inc", func(t *testing.T) {
		inc := funcs["inc"].(func(int) int)
		assert.Equal(t, 2, inc(1))
		assert.Equal(t, 11, inc(10))
	})

	t.Run("dec", func(t *testing.T) {
		dec := funcs["dec"].(func(int) int)
		assert.Equal(t, 0, dec(1))
		assert.Equal(t, 9, dec(10))
	})

	t.Run("mul", func(t *testing.T) {
		mul := funcs["mul"].(func(int, int) int)
		assert.Equal(t, 6, mul(2, 3))
		assert.Equal(t, 20, mul(4, 5))
	})

	t.Run("slice", func(t *testing.T) {
		slice := funcs["slice"].(func(string, int, int) string)
		assert.Equal(t, "hel", slice("hello", 0, 3))
		assert.Equal(t, "lo", slice("hello", 3, 5))
		assert.Equal(t, "", slice("hello", 10, 15))
	})

	t.Run("upper", func(t *testing.T) {
		upper := funcs["upper"].(func(string) string)
		assert.Equal(t, "H", upper("hello"))
		assert.Equal(t, "W", upper("world"))
		assert.Equal(t, "", upper(""))
	})
}

func TestNewPlugin(t *testing.T) {
	p := NewPlugin()
	require.NotNil(t, p)
	assert.Nil(t, p.handler)
	assert.Nil(t, p.templates)
	assert.Nil(t, p.userSvc)
	assert.Nil(t, p.sessionSvc)
	assert.Nil(t, p.auditSvc)
	assert.Nil(t, p.rbacSvc)
}
