package handlers

import "testing"

func TestSanitizeProviderError(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "status 401",
			in:   "upstream failed with 401 unauthorized",
			want: "上游服务认证失败",
		},
		{
			name: "status 500",
			in:   "request failed: 500",
			want: "上游服务暂时不可用，请稍后重试",
		},
		{
			name: "timeout text",
			in:   "deadline exceeded while waiting",
			want: "上游服务超时",
		},
		{
			name: "connection refused",
			in:   "dial tcp: connection refused",
			want: "上游服务连接失败",
		},
		{
			name: "json payload",
			in:   `{"error":"bad request","code":400}`,
			want: "上游服务拒绝了请求",
		},
		{
			name: "generic message",
			in:   "something bad happened",
			want: "请求处理失败",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeProviderError(testError(tc.in))
			if got != tc.want {
				t.Fatalf("sanitizeProviderError() = %q, want %q", got, tc.want)
			}
		})
	}
}

type testError string

func (e testError) Error() string {
	return string(e)
}
