package cre

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
)

type NumericType interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64 | *big.Int | decimal.Decimal
}

type Primitive interface {
	NumericType | string | bool
}

func ConsensusMedianAggregation[T NumericType]() ConsensusAggregation[T] {
	return &consensusDescriptor[T]{Descriptor_: &pb.ConsensusDescriptor_Aggregation{Aggregation: pb.AggregationType_AGGREGATION_TYPE_MEDIAN}}
}

func ConsensusIdenticalAggregation[T any]() ConsensusAggregation[T] {
	var t T
	if isIdenticalType(reflect.TypeOf(t)) {
		return &consensusDescriptor[T]{Descriptor_: &pb.ConsensusDescriptor_Aggregation{Aggregation: pb.AggregationType_AGGREGATION_TYPE_IDENTICAL}}
	}

	return &consensusDescriptorError[T]{Error: fmt.Errorf("%T is not a valid type for identical consensus", t)}
}

func ConsensusCommonPrefixAggregation[T any]() func() (ConsensusAggregation[[]T], error) {
	return func() (ConsensusAggregation[[]T], error) {
		var t []T
		if isIdenticalSliceOrArray(reflect.TypeOf(t)) {
			return &consensusDescriptor[[]T]{Descriptor_: &pb.ConsensusDescriptor_Aggregation{Aggregation: pb.AggregationType_AGGREGATION_TYPE_COMMON_PREFIX}}, nil
		}

		return &consensusDescriptor[[]T]{}, fmt.Errorf("%T is not a valid type for common prefix consensus", t)
	}
}

func ConsensusCommonSuffixAggregation[T any]() func() (ConsensusAggregation[[]T], error) {
	return func() (ConsensusAggregation[[]T], error) {
		var t []T
		if isIdenticalSliceOrArray(reflect.TypeOf(t)) {
			return &consensusDescriptor[[]T]{Descriptor_: &pb.ConsensusDescriptor_Aggregation{Aggregation: pb.AggregationType_AGGREGATION_TYPE_COMMON_SUFFIX}}, nil
		}

		return &consensusDescriptor[[]T]{}, fmt.Errorf("%T is not a valid type for common prefix consensus", t)
	}
}

func ConsensusAggregationFromTags[T any]() ConsensusAggregation[T] {
	var zero T
	t := reflect.TypeOf(zero)
	descriptor, err := parseConsensusTag(t, "")
	if err != nil {
		return &consensusDescriptorError[T]{Error: err}
	}
	return (*consensusDescriptor[T])(descriptor)
}

var bigIntType = reflect.TypeOf((*big.Int)(nil))
var timeType = reflect.TypeOf(time.Time{})
var decimalType = reflect.TypeOf(decimal.Decimal{})

func parseConsensusTag(t reflect.Type, path string) (*pb.ConsensusDescriptor, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("ConsensusAggregationFromTags expects a struct type, got %s", t.Kind())
	}

	descriptors := make(map[string]*pb.ConsensusDescriptor)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("consensus_aggregation")
		if tag == "" {
			return nil, fmt.Errorf("missing consensus tag on type %s accessed via %s", t.Name(), path+field.Name)
		}
		if tag == "ignore" {
			continue
		}

		serializedName := field.Name
		mapstructureTagParts := strings.Split(field.Tag.Get("mapstructure"), ",")
		if mapstructureTagParts[0] != "" {
			serializedName = mapstructureTagParts[0]
		}

		if len(mapstructureTagParts) > 1 && mapstructureTagParts[1] == "squash" {
			inner, err := parseConsensusTag(field.Type, path+field.Name+".")
			if err != nil {
				return nil, fmt.Errorf("nested field %s: %w", field.Name, err)
			}

			for innerFieldName, innerDescriptor := range inner.GetFieldsMap().Fields {
				descriptors[innerFieldName] = innerDescriptor
			}
			break
		}

		tpe := field.Type
		if tpe.Kind() == reflect.Pointer && tpe != bigIntType {
			tpe = tpe.Elem()
		}

		var err error
		switch tag {
		case "median":
			if !isNumeric(tpe) {
				return nil, fmt.Errorf("field %s marked as median but is not a numeric type", field.Name)
			}
			descriptors[serializedName] = &pb.ConsensusDescriptor{Descriptor_: &pb.ConsensusDescriptor_Aggregation{Aggregation: pb.AggregationType_AGGREGATION_TYPE_MEDIAN}}
		case "identical":
			if !isIdenticalType(tpe) {
				return nil, fmt.Errorf("field %s marked as identical but is not a valid type", field.Name)
			}
			descriptors[serializedName] = &pb.ConsensusDescriptor{Descriptor_: &pb.ConsensusDescriptor_Aggregation{Aggregation: pb.AggregationType_AGGREGATION_TYPE_IDENTICAL}}
		case "common_prefix":
			if !isIdenticalSliceOrArray(tpe) {
				return nil, fmt.Errorf("field %s marked as common_prefix but is not slice/array", field.Name)
			}
			descriptors[serializedName] = &pb.ConsensusDescriptor{Descriptor_: &pb.ConsensusDescriptor_Aggregation{Aggregation: pb.AggregationType_AGGREGATION_TYPE_COMMON_PREFIX}}
		case "common_suffix":
			if !isIdenticalSliceOrArray(field.Type) {
				return nil, fmt.Errorf("field %s marked as common_suffix but is not slice/array", field.Name)
			}
			descriptors[serializedName] = &pb.ConsensusDescriptor{Descriptor_: &pb.ConsensusDescriptor_Aggregation{Aggregation: pb.AggregationType_AGGREGATION_TYPE_COMMON_SUFFIX}}
		case "nested":
			descriptors[serializedName], err = parseConsensusTag(field.Type, path+field.Name+".")
			if err != nil {
				return nil, fmt.Errorf("nested field %s: %w", field.Name, err)
			}
		default:
			return nil, fmt.Errorf("unknown consensus tag: %s on field %s", tag, field.Name)
		}
	}

	return &pb.ConsensusDescriptor{
		Descriptor_: &pb.ConsensusDescriptor_FieldsMap{
			FieldsMap: &pb.FieldsMap{Fields: descriptors},
		},
	}, nil
}

func isIdenticalSliceOrArray(t reflect.Type) bool {
	return (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) && isIdenticalType(t.Elem())
}

func isNumeric(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return t == bigIntType || t == decimalType || t == timeType
	}
}

func isIdenticalType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.Map:
		return t.Key().Kind() == reflect.String && isIdenticalType(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if !isIdenticalType(t.Field(i).Type) {
				return false
			}
		}
		return true
	case reflect.Slice, reflect.Array:
		return isIdenticalType(t.Elem())
	case reflect.Pointer:
		if t == bigIntType {
			return true
		}
		return t.Elem().Kind() != reflect.Pointer && isIdenticalType(t.Elem())
	default:
		return false
	}
}
