package registry_test

import (
	"testing"

	actionandtriggermock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/actionandtrigger/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/cre-sdk-go/cre/testutils/registry"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction/mock"
)

func TestRegisterCapability(t *testing.T) {
	r := registry.GetRegistry(t)
	c := &basicactionmock.BasicActionCapability{}

	err := r.RegisterCapability(c)
	assert.NoError(t, err)

	c = &basicactionmock.BasicActionCapability{}
	err = r.RegisterCapability(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "capability already exists:")
}

func TestForceRegisterCapability(t *testing.T) {
	r := registry.GetRegistry(t)
	c1 := &basicactionmock.BasicActionCapability{}
	c2 := &basicactionmock.BasicActionCapability{}

	r.ForceRegisterCapability(c1)
	r.ForceRegisterCapability(c2)

	actual, err := r.GetCapability(c1.ID())
	require.NoError(t, err)
	assert.Equal(t, c2, actual)
}

func TestGetCapability(t *testing.T) {
	r := registry.GetRegistry(t)
	c1 := &actionandtriggermock.BasicCapability{}
	c2 := &basicactionmock.BasicActionCapability{}

	err := r.RegisterCapability(c1)
	require.NoError(t, err)

	//  make sure that the same capability isn't always returned
	err = r.RegisterCapability(c2)
	require.NoError(t, err)

	got, err := r.GetCapability(c1.ID())
	require.NoError(t, err)
	assert.Equal(t, c1.ID(), got.ID())

	got, err = r.GetCapability(c2.ID())
	require.NoError(t, err)
	assert.Equal(t, c2.ID(), got.ID())

	notReal := "not" + c1.ID()
	_, err = r.GetCapability(notReal)
	require.Error(t, err)
}
