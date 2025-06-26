package registry_test

/*
import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	basicactionmock "github.com/smartcontractkit/cre-sdk-go/capabilities/test/basicaction/basic_actionmock"
	basictriggermock "github.com/smartcontractkit/cre-sdk-go/capabilities/test/basictrigger/basic_triggermock"
	"github.com/smartcontractkit/cre-sdk-go/sdk/testutils/registry"
)

func TestRegisterCapability(t *testing.T) {
	r := registry.GetRegistry(t)
	c := &basictriggermock.BasicCapability{}

	err := r.RegisterCapability(c)
	assert.NoError(t, err)

	c = &basictriggermock.BasicCapability{}
	err = r.RegisterCapability(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "capability already exists:")
}

func TestGetCapability(t *testing.T) {
	r := registry.GetRegistry(t)
	c1 := &basictriggermock.BasicCapability{}
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
*/
