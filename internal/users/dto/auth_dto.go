package dto

type RegisterReq struct {
	Fullname string    `json:"fullname" binding:"required,min=2,max=50"`
	Username string    `json:"username" binding:"required,min=2,max=50"`
	Email    string    `json:"email" binding:"omitempty,max=80"`
	Password string    `json:"password" binding:"required,min=6,max=20"`
	Device   DeviceReq `json:"device" binding:"required"`
}

type LoginReq struct {
	Username string    `json:"username" binding:"required,min=2,max=50"`
	Password string    `json:"password" binding:"required,min=6,max=20"`
	Device   DeviceReq `json:"device" binding:"required"`
	IP       string    `json:"-"`
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" binding:"required,max=1000"`
}

type TokenRes struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresAt int64  `json:"-"`
	SessionID        string `json:"-"`
}

type MailOTPRequestReq struct {
	Email string `json:"email" binding:"required,email,max=100" example:"user@example.com"`
}

type MailOTPRequestRes struct {
	// Seconds until the code expires.
	ExpiresIn int `json:"expires_in" example:"300"`
}

type MailOTPVerifyReq struct {
	Email string `json:"email" binding:"required,email,max=100" example:"user@example.com"`
	OTP   string `json:"otp" binding:"required,len=6" example:"123456"`
}

type MailOTPVerifyRes struct {
	// Single-use key for POST /auth/reset-password.
	Key string `json:"key"`
	// Seconds until the key expires.
	ExpiresIn int `json:"expires_in" example:"600"`
}

type ResetPasswordReq struct {
	Key         string `json:"key" binding:"required,max=500"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=20"`
}
