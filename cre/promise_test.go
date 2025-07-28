package cre_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/cre-sdk-go/cre"
)

func TestPromiseFromResult(t *testing.T) {
	p := cre.PromiseFromResult("hello", nil)

	val, err := p.Await()
	assert.NoError(t, err)
	assert.Equal(t, "hello", val)
}

func TestPromiseFromResultError(t *testing.T) {
	expectedErr := errors.New("failure")
	p := cre.PromiseFromResult[string]("", expectedErr)

	val, err := p.Await()
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, "", val)
}

func TestBasicPromiseResolvesOnlyOnce(t *testing.T) {
	counter := 0
	p := cre.NewBasicPromise(func() (int, error) {
		counter++
		return 42, nil
	})

	val1, err1 := p.Await()
	val2, err2 := p.Await()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 42, val1)
	assert.Equal(t, 42, val2)
	assert.Equal(t, 1, counter, "await should be called only once")
}

func TestBasicPromiseError(t *testing.T) {
	p := cre.NewBasicPromise(func() (int, error) {
		return 0, errors.New("something went wrong")
	})

	_, err := p.Await()
	assert.Error(t, err)
}

func TestThenChainsCorrectly(t *testing.T) {
	p := cre.PromiseFromResult(3, nil)
	chained := cre.Then(p, func(i int) (string, error) {
		return string(rune('A' + i)), nil
	})

	result, err := chained.Await()
	assert.NoError(t, err)
	assert.Equal(t, "D", result)
}

func TestThenPropagatesError(t *testing.T) {
	expectedErr := errors.New("boom")
	p := cre.PromiseFromResult[int](0, expectedErr)

	chained := cre.Then(p, func(i int) (string, error) {
		return "should not happen", nil
	})

	_, err := chained.Await()
	assert.ErrorIs(t, err, expectedErr)
}

func TestThenHandlesFnError(t *testing.T) {
	p := cre.PromiseFromResult(123, nil)
	fnErr := errors.New("failed")
	chained := cre.Then(p, func(i int) (string, error) {
		return "", fnErr
	})

	_, err := chained.Await()
	assert.ErrorIs(t, err, fnErr)
}
