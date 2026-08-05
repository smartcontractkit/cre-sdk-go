package cre

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
)

func TestRegionUnmarshalText(t *testing.T) {
	t.Run("known region", func(t *testing.T) {
		var r Region
		require.NoError(t, r.UnmarshalText([]byte("us-west-2")))
		assert.Equal(t, AwsUsWest2, r)
	})

	t.Run("unknown region", func(t *testing.T) {
		var r Region
		err := r.UnmarshalText([]byte("mars-central-1"))
		assert.ErrorContains(t, err, `unknown region "mars-central-1"`)
	})
}

func TestNitroRegionUnmarshalText(t *testing.T) {
	t.Run("known region", func(t *testing.T) {
		var r NitroRegion
		require.NoError(t, r.UnmarshalText([]byte("us-west-2")))
		assert.Equal(t, NitroUsWest2, r)
	})

	t.Run("unknown region", func(t *testing.T) {
		var r NitroRegion
		err := r.UnmarshalText([]byte("eu-central-1"))
		assert.ErrorContains(t, err, `aws nitro does not support region "eu-central-1"`)
	})
}

func TestAnyTeeToRequirements(t *testing.T) {
	got := AnyTee{}.toRequirements()
	want := &sdk.Requirements{Tee: &sdk.Tee{Item: &sdk.Tee_AnyRegions{AnyRegions: &sdk.Regions{}}}}
	assert.True(t, proto.Equal(want, got), "got %v want %v", got, want)
}

func TestAnyTeeInRegionsToRequirements(t *testing.T) {
	t.Run("with regions", func(t *testing.T) {
		got := AnyTeeInRegions{Regions: []Region{AwsUsWest2}}.toRequirements()
		want := &sdk.Requirements{Tee: &sdk.Tee{Item: &sdk.Tee_AnyRegions{AnyRegions: &sdk.Regions{Regions: []string{"us-west-2"}}}}}
		assert.True(t, proto.Equal(want, got))
	})

	t.Run("nil regions emits empty slice", func(t *testing.T) {
		got := AnyTeeInRegions{}.toRequirements()
		regions := got.Tee.GetAnyRegions().GetRegions()
		assert.Empty(t, regions)
	})
}

func TestOneOfTeesToRequirements(t *testing.T) {
	t.Run("nitro with regions", func(t *testing.T) {
		got := OneOfTees{Nitro{Regions: []NitroRegion{NitroUsWest2}}}.toRequirements()
		want := &sdk.Requirements{Tee: &sdk.Tee{Item: &sdk.Tee_TeeTypesAndRegions{
			TeeTypesAndRegions: &sdk.TeeTypesAndRegions{TeeTypeAndRegions: []*sdk.TeeTypeAndRegions{
				{Type: sdk.TeeType_TEE_TYPE_AWS_NITRO, Regions: []string{"us-west-2"}},
			}},
		}}}
		assert.True(t, proto.Equal(want, got), "got %v want %v", got, want)
	})

	t.Run("nitro without regions emits nil region slice", func(t *testing.T) {
		got := OneOfTees{Nitro{}}.toRequirements()
		entries := got.Tee.GetTeeTypesAndRegions().TeeTypeAndRegions
		require.Len(t, entries, 1)
		assert.Equal(t, sdk.TeeType_TEE_TYPE_AWS_NITRO, entries[0].Type)
		assert.Nil(t, entries[0].Regions)
	})
}

// Sealing of TeeConstraint and TeeBinding is enforced at compile time via the
// unexported toRequirements / teeTypeAndRegions methods. These assertions exist
// purely to fail compilation if a future change accidentally breaks the seal.
var (
	_ TeeConstraint = AnyTee{}
	_ TeeConstraint = AnyTeeInRegions{}
	_ TeeConstraint = OneOfTees{}
	_ TeeBinding    = Nitro{}
)

func TestDecodeArrayDirectJSONError(t *testing.T) {
	// Outer json.Unmarshal would normally reject malformed JSON before our
	// UnmarshalJSON sees it; we call decodeArray directly to exercise its
	// inner-parse error path.
	var got Tees
	err := got.decodeArray([]byte("not-json"))
	require.Error(t, err)
}

func TestDecodeObjectDirectJSONError(t *testing.T) {
	var got Tees
	err := got.decodeObject([]byte("not-json"))
	require.Error(t, err)
}

func TestTeesUnmarshalJSON_AnyTee(t *testing.T) {
	var got Tees
	require.NoError(t, json.Unmarshal([]byte(`{}`), &got))
	assert.Equal(t, AnyTee{}, got.TeeConstraint)
}

func TestTeesUnmarshalJSON_AnyTeeInRegions(t *testing.T) {
	var got Tees
	require.NoError(t, json.Unmarshal([]byte(`{"regions":["us-west-2"]}`), &got))
	assert.Equal(t, AnyTeeInRegions{Regions: []Region{AwsUsWest2}}, got.TeeConstraint)
}

func TestTeesUnmarshalJSON_OneOfTees(t *testing.T) {
	t.Run("nitro with regions", func(t *testing.T) {
		var got Tees
		require.NoError(t, json.Unmarshal([]byte(`[{"tee":"nitro","regions":["us-west-2"]}]`), &got))
		assert.Equal(t, OneOfTees{Nitro{Regions: []NitroRegion{NitroUsWest2}}}, got.TeeConstraint)
	})

	t.Run("nitro without regions", func(t *testing.T) {
		var got Tees
		require.NoError(t, json.Unmarshal([]byte(`[{"tee":"nitro"}]`), &got))
		assert.Equal(t, OneOfTees{Nitro{}}, got.TeeConstraint)
	})

}

func TestTeesUnmarshalJSON_Errors(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"null fails closed", `null`, "tee constraint is required"},
		{"binding regions explicit null", `[{"tee":"nitro","regions":null}]`, `"regions" must not be empty`},
		{"bare string", `"any"`, "must be an object or array"},
		{"bare number", `123`, "must be an object or array"},
		{"empty list", `[]`, "list must not be empty"},
		{"invalid array json", `[`, "unexpected end of JSON input"},
		{"binding missing tee key", `[{}]`, `missing "tee" discriminator`},
		{"binding unknown tee", `[{"tee":"sgx"}]`, `unknown tee "sgx"`},
		{"binding probe json error", `[{"tee":3}]`, "cannot unmarshal"},
		{"binding regions parse error", `[{"tee":"nitro","regions":["eu-central-1"]}]`, "aws nitro does not support region"},
		{"binding regions empty", `[{"tee":"nitro","regions":[]}]`, `"regions" must not be empty`},
		{"binding unknown field", `[{"tee":"nitro","extra":1}]`, `unknown field "extra"`},
		{"binding nitro json error", `[{"tee":"nitro","regions":"not-an-array"}]`, "cannot unmarshal"},
		{"object unknown shape", `{"foo":1}`, "unrecognized tee constraint object"},
		{"object with regions plus extra", `{"regions":["us-west-2"],"extra":1}`, "unrecognized tee constraint object"},
		{"object regions bad type", `{"regions":"not-array"}`, "cannot unmarshal"},
		{"object regions unknown", `{"regions":["mars-central-1"]}`, "unknown region"},
		{"object regions empty", `{"regions":[]}`, `"regions" must not be empty`},
		{"object invalid json", `{`, "unexpected end of JSON input"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got Tees
			err := json.Unmarshal([]byte(tc.input), &got)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestTeesPointerNullIsNil(t *testing.T) {
	type cfg struct {
		Tees *Tees `json:"tees"`
	}
	var c cfg
	require.NoError(t, json.Unmarshal([]byte(`{"tees":null}`), &c))
	assert.Nil(t, c.Tees)
}

func TestTeesPointerOmittedIsNil(t *testing.T) {
	type cfg struct {
		Tees *Tees `json:"tees"`
	}
	var c cfg
	require.NoError(t, json.Unmarshal([]byte(`{}`), &c))
	assert.Nil(t, c.Tees)
}

func TestTeesEmbedInConfigStruct(t *testing.T) {
	type cfg struct {
		Name string `json:"name"`
		Tees Tees   `json:"tees"`
	}
	var c cfg
	require.NoError(t, json.Unmarshal([]byte(`{"name":"x","tees":[{"tee":"nitro","regions":["us-west-2"]}]}`), &c))
	assert.Equal(t, "x", c.Name)
	assert.Equal(t, OneOfTees{Nitro{Regions: []NitroRegion{NitroUsWest2}}}, c.Tees.TeeConstraint)
}
