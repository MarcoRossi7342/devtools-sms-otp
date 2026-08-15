package otpflow

import (
	"context"
	"testing"
)

type gatewayStub struct {
	verified bool
}

func (gatewayStub) RequestCode(context.Context, string, string) error { return nil }
func (g gatewayStub) VerifyCode(context.Context, string, string, string) (bool, error) {
	return g.verified, nil
}

func TestVerifyLoginDecision(t *testing.T) {
	tests := []struct {
		name      string
		verified  bool
		allowed   bool
		operation string
	}{
		{name: "verified code grants login", verified: true, allowed: true, operation: "login_granted"},
		{name: "unverified code blocks login", verified: false, allowed: false, operation: "login_blocked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewLoginPipeline(gatewayStub{verified: tt.verified})
			got, err := pipeline.Verify(context.Background(), LoginInput{
				Phone: "+15551234567", Code: "123456", RequestID: "login-42-verify",
				BuildID: "build-42", ReleaseID: "release-7",
			})
			if err != nil {
				t.Fatal(err)
			}
			if got.Allowed != tt.allowed || got.ReleaseOperation != tt.operation {
				t.Fatalf("got allowed=%v operation=%q", got.Allowed, got.ReleaseOperation)
			}
		})
	}
}
