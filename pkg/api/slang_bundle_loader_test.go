package api

import (
	"testing"

	"github.com/Bitspark/slang/pkg/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSlangBundleLoader_List(t *testing.T) {
	assert := assert.New(t)

	id1 := uuid.New()
	id2 := uuid.New()

	blueprints := []core.Blueprint{
		{Id: id1},
		{Id: id2},
	}

	loader, err := NewBundleLoader(blueprints)
	assert.NoError(err)

	list, err := loader.List()
	assert.NoError(err)
	assert.Len(list, 2)
	assert.Contains(list, id1)
	assert.Contains(list, id2)
}
