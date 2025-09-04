package cre_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/chainlink-protos/cre/go/values"
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestConsensusMedianAggregation(t *testing.T) {
	descriptor := cre.ConsensusMedianAggregation[int]()
	require.NoError(t, descriptor.Err())
	assert.Equal(t, descriptor.Descriptor().GetAggregation(), sdk.AggregationType_AGGREGATION_TYPE_MEDIAN)
}

func TestConsensusIdenticalAggregation(t *testing.T) {
	t.Run("valid types", func(t *testing.T) {
		descriptor := cre.ConsensusIdenticalAggregation[int]()
		require.NoError(t, descriptor.Err())
		assert.Equal(t, descriptor.Descriptor().GetAggregation(), sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL)
	})

	t.Run("invalid types", func(t *testing.T) {
		descriptor := cre.ConsensusIdenticalAggregation[chan int]()
		require.Error(t, descriptor.Err())
	})
}

func TestConsensusCommonPrefixAggregation(t *testing.T) {
	t.Run("valid primitive types", func(t *testing.T) {
		descriptor, err := cre.ConsensusCommonPrefixAggregation[string]()()
		require.NoError(t, err)
		assert.Equal(t, descriptor.Descriptor().GetAggregation(), sdk.AggregationType_AGGREGATION_TYPE_COMMON_PREFIX)
	})

	t.Run("invalid primitive types", func(t *testing.T) {
		_, err := cre.ConsensusCommonPrefixAggregation[[]chan int]()()
		require.Error(t, err)
	})
}

func TestConsensusCommonSuffixAggregation(t *testing.T) {
	t.Run("valid primitive types", func(t *testing.T) {
		descriptor, err := cre.ConsensusCommonSuffixAggregation[string]()()
		require.NoError(t, err)
		assert.Equal(t, descriptor.Descriptor().GetAggregation(), sdk.AggregationType_AGGREGATION_TYPE_COMMON_SUFFIX)
	})

	t.Run("invalid primitive types", func(t *testing.T) {
		_, err := cre.ConsensusCommonSuffixAggregation[[]chan int]()()
		require.Error(t, err)
	})
}

func TestConsensusAggregationFromTags(t *testing.T) {
	t.Run("valid median - all numeric types", func(t *testing.T) {
		t.Run("int", func(t *testing.T) { testMedianField[int](t) })
		t.Run("int8", func(t *testing.T) { testMedianField[int8](t) })
		t.Run("int16", func(t *testing.T) { testMedianField[int16](t) })
		t.Run("int32", func(t *testing.T) { testMedianField[int32](t) })
		t.Run("int64", func(t *testing.T) { testMedianField[int64](t) })
		t.Run("uint", func(t *testing.T) { testMedianField[uint](t) })
		t.Run("uint8", func(t *testing.T) { testMedianField[uint8](t) })
		t.Run("uint16", func(t *testing.T) { testMedianField[uint16](t) })
		t.Run("uint32", func(t *testing.T) { testMedianField[uint32](t) })
		t.Run("uint64", func(t *testing.T) { testMedianField[uint64](t) })
		t.Run("float32", func(t *testing.T) { testMedianField[float32](t) })
		t.Run("float64", func(t *testing.T) { testMedianField[float64](t) })
		t.Run("*big.Int", func(t *testing.T) { testMedianField[*big.Int](t) })
		t.Run("decimal", func(t *testing.T) { testMedianField[decimal.Decimal](t) })
		t.Run("time", func(t *testing.T) { testMedianField[time.Time](t) })
	})

	t.Run("private fields are ignored", func(t *testing.T) {
		expected := &sdk.ConsensusDescriptor{
			Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
				FieldsMap: &sdk.FieldsMap{
					Fields: map[string]*sdk.ConsensusDescriptor{
						"Val": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
					},
				},
			},
		}

		t.Run("implicit ignore", func(t *testing.T) {
			type S struct {
				Val      string `consensus_aggregation:"identical"`
				privateV string
			}

			desc := cre.ConsensusAggregationFromTags[S]()
			require.NoError(t, desc.Err())
			assert.True(t, proto.Equal(expected, desc.Descriptor()))
		})

		t.Run("explicit ignore", func(t *testing.T) {
			type S struct {
				Val      string `consensus_aggregation:"identical"`
				privateV string `consensus_aggregation:"ignore"`
			}

			desc := cre.ConsensusAggregationFromTags[S]()
			require.NoError(t, desc.Err())
			assert.True(t, proto.Equal(expected, desc.Descriptor()))
		})
	})

	t.Run("private fields error if they are tagged but not ignored", func(t *testing.T) {
		type S struct {
			Val      string `consensus_aggregation:"identical"`
			privateV string `consensus_aggregation:"identical"`
		}

		desc := cre.ConsensusAggregationFromTags[S]()
		// unexported field privateV with consensus tag on type S accessed via
		require.ErrorContains(t, desc.Err(), "unexported field privateV with consensus tag on type S")
	})

	t.Run("valid identical", func(t *testing.T) {
		type S struct {
			Val   string    `consensus_aggregation:"identical"`
			PVal  *string   `consensus_aggregation:"identical"`
			Slice []string  `consensus_aggregation:"identical"`
			Array [2]string `consensus_aggregation:"identical"`
			Bi    *big.Int  `consensus_aggregation:"identical"`
		}
		desc := cre.ConsensusAggregationFromTags[S]()
		require.NoError(t, desc.Err())
		expected := &sdk.ConsensusDescriptor{
			Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
				FieldsMap: &sdk.FieldsMap{
					Fields: map[string]*sdk.ConsensusDescriptor{
						"Val": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
						"PVal": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
						"Slice": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
						"Array": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
						"Bi": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
					},
				},
			},
		}
		require.True(t, proto.Equal(desc.Descriptor(), expected))
	})

	t.Run("valid common prefix", func(t *testing.T) {
		type S struct {
			Val []string `consensus_aggregation:"common_prefix"`
		}
		desc := cre.ConsensusAggregationFromTags[S]()
		require.NoError(t, desc.Err())
		expected := &sdk.ConsensusDescriptor{
			Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
				FieldsMap: &sdk.FieldsMap{
					Fields: map[string]*sdk.ConsensusDescriptor{
						"Val": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_COMMON_PREFIX,
							},
						},
					},
				},
			},
		}
		require.True(t, proto.Equal(desc.Descriptor(), expected))
	})

	t.Run("valid common suffix", func(t *testing.T) {
		type S struct {
			Val [2]string `consensus_aggregation:"common_suffix"`
		}
		desc := cre.ConsensusAggregationFromTags[S]()
		require.NoError(t, desc.Err())
		expected := &sdk.ConsensusDescriptor{
			Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
				FieldsMap: &sdk.FieldsMap{
					Fields: map[string]*sdk.ConsensusDescriptor{
						"Val": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_COMMON_SUFFIX,
							},
						},
					},
				},
			},
		}
		require.True(t, proto.Equal(desc.Descriptor(), expected))
	})

	t.Run("valid nested", func(t *testing.T) {
		type Inner struct {
			Score int32 `consensus_aggregation:"median"`
		}
		type Outer struct {
			In Inner `consensus_aggregation:"nested"`
		}
		desc := cre.ConsensusAggregationFromTags[Outer]()
		require.NoError(t, desc.Err())
		expected := &sdk.ConsensusDescriptor{
			Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
				FieldsMap: &sdk.FieldsMap{
					Fields: map[string]*sdk.ConsensusDescriptor{
						"In": {
							Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
								FieldsMap: &sdk.FieldsMap{
									Fields: map[string]*sdk.ConsensusDescriptor{
										"Score": {
											Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
												Aggregation: sdk.AggregationType_AGGREGATION_TYPE_MEDIAN,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		require.True(t, proto.Equal(desc.Descriptor(), expected))
	})

	t.Run("valid identical nested", func(t *testing.T) {
		type Inner struct {
			Score int32
		}

		type Outer struct {
			In  Inner  `consensus_aggregation:"identical"`
			PIn *Inner `consensus_aggregation:"identical"`
		}
		desc := cre.ConsensusAggregationFromTags[Outer]()
		require.NoError(t, desc.Err())
		expected := &sdk.ConsensusDescriptor{
			Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
				FieldsMap: &sdk.FieldsMap{
					Fields: map[string]*sdk.ConsensusDescriptor{
						"In": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
						"PIn": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
					},
				},
			},
		}
		require.True(t, proto.Equal(desc.Descriptor(), expected))
	})

	t.Run("valid naming aligns with mapstructure rename", func(t *testing.T) {
		type Inner struct {
			Val string `consensus_aggregation:"identical" mapstructure:"renamed_val_inner"`
		}

		type MapstructureFields struct {
			Val  string `consensus_aggregation:"identical" mapstructure:"renamed_val"`
			Val2 Inner  `consensus_aggregation:"identical" mapstructure:",squash"`
		}

		desc := cre.ConsensusAggregationFromTags[*MapstructureFields]()
		require.NoError(t, desc.Err())

		expected := &sdk.ConsensusDescriptor{
			Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
				FieldsMap: &sdk.FieldsMap{
					Fields: map[string]*sdk.ConsensusDescriptor{
						"renamed_val": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
						"renamed_val_inner": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
					},
				},
			},
		}
		assert.True(t, proto.Equal(desc.Descriptor(), expected))

		wrapped, err := values.Wrap(&MapstructureFields{Val: "anything", Val2: Inner{Val: "anything_else"}})
		require.NoError(t, err)
		actual := &MapstructureFields{}
		require.NoError(t, wrapped.UnwrapTo(actual))
		assert.Equal(t, "anything", actual.Val)
	})

	t.Run("invalid identical nested", func(t *testing.T) {
		type Inner struct {
			Ch chan int32 `consensus_aggregation:"identical"`
		}

		type Outer struct {
			In Inner `consensus_aggregation:"identical"`
		}
		desc := cre.ConsensusAggregationFromTags[Outer]()
		require.Error(t, desc.Err())
	})

	t.Run("invalid nested field", func(t *testing.T) {
		type Inner struct {
			Ch chan int `consensus_aggregation:"median"`
		}
		type Outer struct {
			In Inner `consensus_aggregation:"nested"`
		}
		desc := cre.ConsensusAggregationFromTags[Outer]()
		require.Error(t, desc.Err())
	})

	t.Run("valid pointer", func(t *testing.T) {
		type S struct {
			Val string `consensus_aggregation:"identical"`
		}

		desc := cre.ConsensusAggregationFromTags[*S]()
		require.NoError(t, desc.Err())
		expected := &sdk.ConsensusDescriptor{
			Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
				FieldsMap: &sdk.FieldsMap{
					Fields: map[string]*sdk.ConsensusDescriptor{
						"Val": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
					},
				},
			},
		}
		require.True(t, proto.Equal(desc.Descriptor(), expected))
	})

	t.Run("invalid median", func(t *testing.T) {
		type S struct {
			Val string `consensus_aggregation:"median"`
		}
		desc := cre.ConsensusAggregationFromTags[S]()
		require.ErrorContains(t, desc.Err(), "not a numeric type")
	})

	t.Run("invalid not a struct", func(t *testing.T) {
		desc := cre.ConsensusAggregationFromTags[int]()
		require.ErrorContains(t, desc.Err(), "expects a struct type")
	})

	t.Run("ignore fields", func(t *testing.T) {
		type S struct {
			Val          string `consensus_aggregation:"identical"`
			IgnoredField string `consensus_aggregation:"ignore"`
		}
		desc := cre.ConsensusAggregationFromTags[S]()
		require.NoError(t, desc.Err())
		expected := &sdk.ConsensusDescriptor{
			Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
				FieldsMap: &sdk.FieldsMap{
					Fields: map[string]*sdk.ConsensusDescriptor{
						"Val": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_IDENTICAL,
							},
						},
					},
				},
			},
		}
		require.True(t, proto.Equal(desc.Descriptor(), expected))
	})

	t.Run("invalid missing field tag specifies full path", func(t *testing.T) {
		type Nested struct {
			Foo string
		}
		type S struct {
			Val    string `consensus_aggregation:"identical"`
			Nested Nested `consensus_aggregation:"nested"`
		}
		desc := cre.ConsensusAggregationFromTags[S]()
		require.ErrorContains(t, desc.Err(), "missing consensus tag on type Nested accessed via Nested.Foo")
	})

	t.Run("invalid identical", func(t *testing.T) {
		t.Run("channel", func(t *testing.T) { testInvalidIdenticalField[chan string](t) })
		t.Run("non string key map", func(t *testing.T) { testInvalidIdenticalField[map[int]int](t) })
	})

	t.Run("common prefix for valid types", func(t *testing.T) {
		descriptor := cre.ConsensusAggregationFromTags[struct {
			Val []int `consensus_aggregation:"common_prefix"`
		}]()

		require.NoError(t, descriptor.Err())
		expected := &sdk.ConsensusDescriptor{
			Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
				FieldsMap: &sdk.FieldsMap{
					Fields: map[string]*sdk.ConsensusDescriptor{
						"Val": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_COMMON_PREFIX,
							},
						},
					},
				},
			},
		}

		require.True(t, proto.Equal(descriptor.Descriptor(), expected))
	})

	t.Run("common prefix invalid types", func(t *testing.T) {
		desc := cre.ConsensusAggregationFromTags[struct {
			Val chan int `consensus_aggregation:"common_prefix"`
		}]()

		require.Error(t, desc.Err())
	})

	t.Run("common suffix for valid types", func(t *testing.T) {
		descriptor := cre.ConsensusAggregationFromTags[struct {
			Val []int `consensus_aggregation:"common_suffix"`
		}]()

		require.NoError(t, descriptor.Err())
		expected := &sdk.ConsensusDescriptor{
			Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
				FieldsMap: &sdk.FieldsMap{
					Fields: map[string]*sdk.ConsensusDescriptor{
						"Val": {
							Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
								Aggregation: sdk.AggregationType_AGGREGATION_TYPE_COMMON_SUFFIX,
							},
						},
					},
				},
			},
		}

		require.True(t, proto.Equal(descriptor.Descriptor(), expected))
	})

	t.Run("common suffix invalid types", func(t *testing.T) {
		desc := cre.ConsensusAggregationFromTags[struct {
			Val chan int `consensus_aggregation:"common_suffix"`
		}]()

		require.Error(t, desc.Err())
	})

	t.Run("invalid tag", func(t *testing.T) {
		type Invalid struct {
			In int `consensus_aggregation:"not_real"`
		}
		desc := cre.ConsensusAggregationFromTags[Invalid]()
		require.Error(t, desc.Err())
	})
}

func testMedianField[T any](t *testing.T) {
	t.Helper()
	desc := cre.ConsensusAggregationFromTags[struct {
		Val  T  `consensus_aggregation:"median"`
		PVal *T `consensus_aggregation:"median"`
	}]()
	require.NoError(t, desc.Err())
	expected := &sdk.ConsensusDescriptor{
		Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
			FieldsMap: &sdk.FieldsMap{
				Fields: map[string]*sdk.ConsensusDescriptor{
					"Val": {
						Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
							Aggregation: sdk.AggregationType_AGGREGATION_TYPE_MEDIAN,
						},
					},
					"PVal": {
						Descriptor_: &sdk.ConsensusDescriptor_Aggregation{
							Aggregation: sdk.AggregationType_AGGREGATION_TYPE_MEDIAN,
						},
					},
				},
			},
		},
	}
	require.True(t, proto.Equal(desc.Descriptor(), expected))
}

func testInvalidIdenticalField[T any](t *testing.T) {
	t.Helper()
	testInvalidIdenticalFieldHelper[T](t)
	testInvalidIdenticalFieldHelper[*T](t)
}

func testInvalidIdenticalFieldHelper[T any](t *testing.T) {
	t.Helper()
	desc := cre.ConsensusAggregationFromTags[struct {
		Val T `consensus_aggregation:"identical"`
	}]()
	require.ErrorContains(t, desc.Err(), "field Val marked as identical but is not a valid type")
}
