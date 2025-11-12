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

func TestNewPlugin(t *testing.T) {
	p := NewPlugin()
	require.NotNil(t, p)
	assert.Nil(t, p.handler)
	assert.Nil(t, p.userSvc)
	assert.Nil(t, p.sessionSvc)
	assert.Nil(t, p.auditSvc)
	assert.Nil(t, p.rbacSvc)
	assert.Nil(t, p.apikeyService)
	assert.Nil(t, p.orgService)
}
