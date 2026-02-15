package sdk

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

var token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMjQyMTA2ODQ0OTM5NTIsImRldmljZV9pZCI6MCwiaXNzIjoid2VhdmVyX2FjY291bnQiLCJleHAiOjE3NzE2OTM3NzEsIm5iZiI6MTc3MTA4ODk3MH0.XYSTS601JbsXx0IIQCSOIC5Mi4xEqcsLsh1ypx65Y0A"
var userTestCli = MustNewClient("http://54.196.109.226/8080", WithToken(token))
var ctx = context.TODO()

func toString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestClient_GetUserInfoById(t *testing.T) {
	ret, err := userTestCli.GetUserInfoById(ctx, "u___124210684493952")
	require.NoError(t, err)
	t.Log(toString(ret))
}
