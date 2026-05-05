package sdk

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/stretchr/testify/require"
)

func TestRequestReturnsHTTPStatusErrorBeforeJSONDecode(t *testing.T) {
	resp := &protocol.Response{}
	resp.SetStatusCode(404)
	resp.SetBodyString("Not Found\n")

	var result UserInfo
	err := decodeAPIResponse(resp, &result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected status 404")
	require.Contains(t, err.Error(), "Not Found")
}
