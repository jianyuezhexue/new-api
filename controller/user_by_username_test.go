package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetUserByUsernameReturnsSameUserDetailsAsIDLookup(t *testing.T) {
	db := openTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}))

	target := &model.User{
		Username:    "username-lookup-target",
		Password:    "hashed-password",
		DisplayName: "Lookup Target",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Quota:       123456,
	}
	require.NoError(t, db.Create(target).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "user_id", Value: target.Username}}
	ctx.Set("role", common.RoleAdminUser)

	GetUserByUsername(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool       `json:"success"`
		Data    model.User `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, target.Id, response.Data.Id)
	require.Equal(t, target.Username, response.Data.Username)
	require.Equal(t, target.Quota, response.Data.Quota)
	require.Empty(t, response.Data.Password)
}
