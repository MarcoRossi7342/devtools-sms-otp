package otpflow

import (
	"context"
	"errors"
)

type OTPGateway interface {
	RequestCode(context.Context, string, string) error
	VerifyCode(context.Context, string, string, string) (bool, error)
}

type LoginPipeline struct {
	sms OTPGateway
}

type LoginInput struct {
	Phone     string `json:"phone"`
	Code      string `json:"code,omitempty"`
	RequestID string `json:"request_id"`
	BuildID   string `json:"build_id"`
	ReleaseID string `json:"release_id"`
}

type LoginResult struct {
	Allowed          bool   `json:"allowed"`
	BuildEvent       string `json:"build_event"`
	ReleaseOperation string `json:"release_operation"`
	Diagnostic       string `json:"diagnostic"`
}

func NewLoginPipeline(sms OTPGateway) *LoginPipeline {
	return &LoginPipeline{sms: sms}
}

func (p *LoginPipeline) Send(ctx context.Context, in LoginInput) (LoginResult, error) {
	if in.Phone == "" || in.RequestID == "" || in.BuildID == "" || in.ReleaseID == "" {
		return LoginResult{}, errors.New("phone, request_id, build_id, and release_id are required")
	}
	if err := p.sms.RequestCode(ctx, in.Phone, in.RequestID); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{BuildEvent: "otp_requested", ReleaseOperation: "login_pending", Diagnostic: "code dispatched"}, nil
}

func (p *LoginPipeline) Verify(ctx context.Context, in LoginInput) (LoginResult, error) {
	if in.Phone == "" || in.Code == "" || in.RequestID == "" || in.BuildID == "" || in.ReleaseID == "" {
		return LoginResult{}, errors.New("phone, code, request_id, build_id, and release_id are required")
	}
	verified, err := p.sms.VerifyCode(ctx, in.Phone, in.Code, in.RequestID)
	if err != nil {
		return LoginResult{}, err
	}
	if !verified {
		return LoginResult{BuildEvent: "otp_rejected", ReleaseOperation: "login_blocked", Diagnostic: "code did not verify"}, nil
	}
	return LoginResult{Allowed: true, BuildEvent: "otp_verified", ReleaseOperation: "login_granted", Diagnostic: "developer identity verified"}, nil
}
