package dto

import "time"

type RegisterRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type RegisterResponse struct {
	Registered bool   `json:"registered"`
	Email      string `json:"email"`
}

type RequestEmailVerificationCodeRequest struct {
	Email string `json:"email"`
}

type RequestEmailVerificationCodeResponse struct {
	Sent               bool `json:"sent"`
	ExpiresInSeconds   int  `json:"expires_in_seconds"`
	ResendAfterSeconds int  `json:"resend_after_seconds"`
}

type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type VerifyEmailResponse struct {
	Verified   bool      `json:"verified"`
	VerifiedAt time.Time `json:"verified_at"`
}
