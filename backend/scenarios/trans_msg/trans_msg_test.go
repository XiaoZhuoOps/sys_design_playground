package transmsg

import (
	"SYS_DESIGN_PLAYGROUND/pkg/repo"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	repo.Init()
}

func TestValidateWebProduct(t *testing.T) {
	var (
		ctx = context.TODO()

		// TODO: 将 uid name 改成随机数 便于重复测试
		uid1  int64 = 123
		name1       = "name1"
		name2       = "name2"
		name3       = "name3"
	)

	code1, err := CreateWebProduct(ctx, name1, uid1)
	assert.Nil(t, err)

	code2, err = CreateWebProduct(ctx, name2, uid1)
	assert.Nil(t, err)

	assert.True(t, ValidateWebProduct(ctx, uid1, code1, name1))
	assert.False(t, ValidateWebProduct(ctx, uid1, code1, name2))
	assert.True(t, ValidateWebProduct(ctx, uid1, code1, name3))
}
