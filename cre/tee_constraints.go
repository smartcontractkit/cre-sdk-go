package cre

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
)

// TeeConstraint describes the set of (TEE, region) pairs a handler will accept.
// Sealed via the unexported toRequirements method: only AnyTee, AnyTeeInRegions,
// and OneOfTees (declared in this package) can implement it.
type TeeConstraint interface {
	toRequirements() *sdk.Requirements
}

// Tees is the config-friendly wrapper around a TeeConstraint. Its UnmarshalJSON
// inspects the JSON shape and produces the matching variant; null and unknown
// shapes are rejected to fail closed. Use *Tees in config structs when the
// field is optional.
type Tees struct {
	TeeConstraint
}

// AnyTee accepts any TEE in any region. JSON form: {}.
type AnyTee struct{}

func (AnyTee) toRequirements() *sdk.Requirements {
	return &sdk.Requirements{Tee: &sdk.Tee{Item: &sdk.Tee_AnyRegions{AnyRegions: &sdk.Regions{}}}}
}

// AnyTeeInRegions accepts any TEE provided it runs in one of the listed regions.
// JSON form: {"regions": [...]}.
type AnyTeeInRegions struct {
	Regions []Region `json:"regions"`
}

func (a AnyTeeInRegions) toRequirements() *sdk.Requirements {
	regs := make([]string, len(a.Regions))
	for i, r := range a.Regions {
		regs[i] = string(r)
	}
	return &sdk.Requirements{Tee: &sdk.Tee{Item: &sdk.Tee_AnyRegions{AnyRegions: &sdk.Regions{Regions: regs}}}}
}

// OneOfTees accepts any of the listed per-TEE bindings. Each binding may carry
// its own region set. JSON form: [{"tee": "...", "regions": [...]}, ...].
type OneOfTees []TeeBinding

func (o OneOfTees) toRequirements() *sdk.Requirements {
	out := make([]*sdk.TeeTypeAndRegions, len(o))
	for i, b := range o {
		out[i] = b.teeTypeAndRegions()
	}
	return &sdk.Requirements{Tee: &sdk.Tee{Item: &sdk.Tee_TeeTypesAndRegions{TeeTypesAndRegions: &sdk.TeeTypesAndRegions{TeeTypeAndRegions: out}}}}
}

// TeeBinding is the sealed interface for a single entry inside OneOfTees. Each
// implementing type owns the enum of regions that TEE supports (e.g. Nitro <->
// NitroRegion) so a region for the wrong TEE is a compile-time error. Sealed
// via the unexported teeTypeAndRegions method.
type TeeBinding interface {
	teeTypeAndRegions() *sdk.TeeTypeAndRegions
}

// Nitro is the AWS Nitro TEE. JSON tag: "nitro".
type Nitro struct {
	Regions []NitroRegion `json:"regions,omitempty"`
}

func (n Nitro) teeTypeAndRegions() *sdk.TeeTypeAndRegions {
	var regs []string
	if n.Regions != nil {
		regs = make([]string, len(n.Regions))
		for i, r := range n.Regions {
			regs[i] = string(r)
		}
	}
	return &sdk.TeeTypeAndRegions{Type: sdk.TeeType_TEE_TYPE_AWS_NITRO, Regions: regs}
}

// Region is the global region enum used by AnyTeeInRegions, where no specific
// TEE is pinned. Add a new region by adding a const plus a case in UnmarshalText.
type Region string

const (
	AwsUsWest2 Region = "us-west-2"
)

func (r *Region) UnmarshalText(b []byte) error {
	switch v := Region(b); v {
	case AwsUsWest2:
		*r = v
		return nil
	default:
		return fmt.Errorf("unknown region %q", b)
	}
}

// NitroRegion enumerates the regions AWS Nitro is supported in. Distinct from
// Region so that a non-Nitro region cannot appear inside a Nitro binding.
type NitroRegion string

const (
	NitroUsWest2 = NitroRegion(AwsUsWest2)
)

func (r *NitroRegion) UnmarshalText(b []byte) error {
	switch v := NitroRegion(b); v {
	case NitroUsWest2:
		*r = v
		return nil
	default:
		return fmt.Errorf("aws nitro does not support region %q", b)
	}
}

// teeBindingFactories maps the JSON "tee" discriminator to a fresh binding to
// unmarshal into. Add a new TEE by registering it here.
var teeBindingFactories = map[string]func() teeBindingUnmarshaler{
	"nitro": func() teeBindingUnmarshaler { return &Nitro{} },
}

// teeBindingUnmarshaler is the concrete pointer type that both implements
// TeeBinding and can be json.Unmarshal'd into. asValue returns the value form
// of the binding, which is what gets stored in OneOfTees.
type teeBindingUnmarshaler interface {
	TeeBinding
	json.Unmarshaler
	asValue() TeeBinding
}

func (n *Nitro) asValue() TeeBinding { return *n }

func (n *Nitro) UnmarshalJSON(b []byte) error {
	type raw struct {
		Tee     string          `json:"tee"`
		Regions json.RawMessage `json:"regions"`
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var r raw
	if err := dec.Decode(&r); err != nil {
		return err
	}
	if len(r.Regions) == 0 {
		n.Regions = nil
		return nil
	}
	var regions []NitroRegion
	if err := json.Unmarshal(r.Regions, &regions); err != nil {
		return err
	}
	if len(regions) == 0 {
		return fmt.Errorf(`"regions" must not be empty`)
	}
	n.Regions = regions
	return nil
}

// UnmarshalJSON dispatches on the JSON shape: {} -> AnyTee, {"regions":[...]} ->
// AnyTeeInRegions, [...] -> OneOfTees. null and any other shape error.
func (t *Tees) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		return fmt.Errorf("tee constraint is required; use *Tees for optional fields")
	}
	switch b[0] {
	case '[':
		return t.decodeArray(b)
	case '{':
		return t.decodeObject(b)
	default:
		return fmt.Errorf("tee constraint must be an object or array, got: %s", b)
	}
}

func (t *Tees) decodeArray(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		return fmt.Errorf("tee constraint list must not be empty")
	}
	out := make(OneOfTees, 0, len(raw))
	for i, entry := range raw {
		bind, err := unmarshalBinding(entry)
		if err != nil {
			return fmt.Errorf("tee constraint[%d]: %w", i, err)
		}
		out = append(out, bind)
	}
	t.TeeConstraint = out
	return nil
}

func (t *Tees) decodeObject(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		t.TeeConstraint = AnyTee{}
		return nil
	}
	regionsRaw, ok := raw["regions"]
	if !ok || len(raw) != 1 {
		return fmt.Errorf(`unrecognized tee constraint object; expected {} or {"regions":[...]}`)
	}
	var regions []Region
	if err := json.Unmarshal(regionsRaw, &regions); err != nil {
		return err
	}
	if len(regions) == 0 {
		return fmt.Errorf(`"regions" must not be empty`)
	}
	t.TeeConstraint = AnyTeeInRegions{Regions: regions}
	return nil
}

func unmarshalBinding(b []byte) (TeeBinding, error) {
	var probe struct {
		Tee string `json:"tee"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return nil, err
	}
	if probe.Tee == "" {
		return nil, fmt.Errorf(`missing "tee" discriminator`)
	}
	factory, ok := teeBindingFactories[probe.Tee]
	if !ok {
		return nil, fmt.Errorf("unknown tee %q", probe.Tee)
	}
	target := factory()
	if err := target.UnmarshalJSON(b); err != nil {
		return nil, err
	}
	return target.asValue(), nil
}
