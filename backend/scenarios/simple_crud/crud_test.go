package simplecrud

import (
	"SYS_DESIGN_PLAYGROUND/pkg/repo"
	"SYS_DESIGN_PLAYGROUND/pkg/repo/model/model"
	"SYS_DESIGN_PLAYGROUND/pkg/repo/model/query"
	"context"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func init() {
	repo.Init()
}

func TestValidateWebProduct(t *testing.T) {
	var (
		ctx = context.TODO()

		uid1  int64 = rand.Int63()
		name1       = uuid.New().String()
		name2       = uuid.New().String()
		name3       = uuid.New().String()
	)

	code1, err := CreateWebProduct(ctx, name1, uid1)
	assert.Nil(t, err)

	_, err = CreateWebProduct(ctx, name2, uid1)
	assert.Nil(t, err)

	isOK, err := CheckBeforeChangeProductName(ctx, uid1, code1, name1)
	assert.Nil(t, err)
	assert.True(t, isOK)

	isOK, err = CheckBeforeChangeProductName(ctx, uid1, code1, name2)
	assert.Nil(t, err)
	assert.False(t, isOK)

	isOK, err = CheckBeforeChangeProductName(ctx, uid1, code1, name3)
	assert.NotNil(t, err)
	t.Logf("err: %+v", err)
	assert.True(t, isOK)
}

func TestMGetWebProductCodes(t *testing.T) {
	var (
		ctx = context.TODO()

		uid1  int64 = rand.Int63()
		uid2  int64 = rand.Int63()
		name1       = uuid.New().String()
		name2       = uuid.New().String()
		name3       = uuid.New().String()
	)

	code1, err := CreateWebProduct(ctx, name1, uid1)
	assert.Nil(t, err)

	code2, err := CreateWebProduct(ctx, name2, uid1)
	assert.Nil(t, err)

	code3, err := CreateWebProduct(ctx, name3, uid2)
	assert.Nil(t, err)

	products1, err := MGetWebProductCodes(ctx, uid1)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(products1))
	assert.Contains(t, products1, code1)
	assert.Contains(t, products1, code2)

	products2, err := MGetWebProductCodes(ctx, uid2)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(products2))
	assert.Contains(t, products2, code3)

	products3, err := MGetWebProductCodes(ctx, rand.Int63())
	assert.Nil(t, err)
	assert.Equal(t, 0, len(products3))
}

func TestListProductCodesByUserRegion(t *testing.T) {
	// Clean up tables before test
	query.Q.User.WithContext(context.Background()).Delete(&model.User{})
	query.Q.WebProduct.WithContext(context.Background()).Delete(&model.WebProduct{})
	query.Q.WebProductUserRelation.WithContext(context.Background()).Delete(&model.WebProductUserRelation{})

	var (
		ctx = context.TODO()

		region1 = "asia"
		region2 = "europe"

		uid1  = rand.Int63()
		name1 = uuid.New().String()
		user1 = &model.User{
			ID:     uid1,
			Code:   uuid.New().String(),
			Name:   name1,
			Email:  uuid.New().String(),
			Region: region1,
			Status: func() *int32 {
				var s int32 = 1
				return &s
			}(),
			Extra: "{}",
		}

		uid2  = rand.Int63()
		name2 = uuid.New().String()
		user2 = &model.User{
			ID:     uid2,
			Code:   uuid.New().String(),
			Name:   name2,
			Email:  uuid.New().String(),
			Region: region2,
			Status: func() *int32 {
				var s int32 = 1
				return &s
			}(),
			Extra: "{}",
		}

		uid3  = rand.Int63()
		name3 = uuid.New().String()
		user3 = &model.User{
			ID:     uid3,
			Code:   uuid.New().String(),
			Name:   name3,
			Email:  uuid.New().String(),
			Region: region1,
			Status: func() *int32 {
				var s int32 = 0
				return &s
			}(),
			Extra: "{}",
		}
	)

	err := query.Q.Transaction(func(tx *query.Query) error {
		if err := tx.User.WithContext(ctx).Create(user1); err != nil {
			return err
		}
		if err := tx.User.WithContext(ctx).Create(user2); err != nil {
			return err
		}
		if err := tx.User.WithContext(ctx).Create(user3); err != nil {
			return err
		}
		return nil
	})
	assert.Nil(t, err)

	code1, err := CreateWebProduct(ctx, uuid.NewString(), uid1)
	assert.Nil(t, err)

	code2, err := CreateWebProduct(ctx, uuid.NewString(), uid1)
	assert.Nil(t, err)

	code3, err := CreateWebProduct(ctx, uuid.NewString(), uid2)
	assert.Nil(t, err)

	_, err = CreateWebProduct(ctx, uuid.NewString(), uid3)
	assert.Nil(t, err)

	products1, err := ListProductCodesByUserRegion(ctx, region1)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(products1))
	assert.Contains(t, products1, code1)
	assert.Contains(t, products1, code2)

	products2, err := ListProductCodesByUserRegion(ctx, region2)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(products2))
	assert.Contains(t, products2, code3)

	products3, err := ListProductCodesByUserRegion(ctx, "africa")
	assert.Nil(t, err)
	assert.Equal(t, 0, len(products3))
}

func TestUpsertUser(t *testing.T) {
	var (
		ctx   = context.TODO()
		email = "test@example.com"
		name1 = "initial_name"
		name2 = "updated_name"
	)

	// 1. Insert a new user
	ok, err := UpsertUser(ctx, email, name1)
	assert.Nil(t, err)
	assert.True(t, ok)

	// Verify the user was created
	user, err := query.Q.User.WithContext(ctx).Where(query.User.Email.Eq(email)).First()
	assert.Nil(t, err)
	assert.Equal(t, name1, user.Name)

	// 2. Update the user's name
	ok, err = UpsertUser(ctx, email, name2)
	assert.Nil(t, err)
	assert.True(t, ok)

	// Verify the user's name was updated
	updatedUser, err := query.Q.User.WithContext(ctx).Where(query.User.Email.Eq(email)).First()
	assert.Nil(t, err)
	assert.Equal(t, name2, updatedUser.Name)
	assert.Equal(t, user.ID, updatedUser.ID)
}
