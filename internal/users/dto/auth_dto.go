package dto

type RegisterReq struct {
	Fullname string    `json:"fullname" binding:"required,min=2,max=50"`
	Username string    `json:"username" binding:"required,min=2,max=50"`
	Email    string    `json:"email" binding:"required,max=100"`
	Password string    `json:"password" binding:"required,min=6,max=20"`
	Device   DeviceReq `json:"device"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Password string `json:"password" binding:"required,min=6,max=20"`
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
