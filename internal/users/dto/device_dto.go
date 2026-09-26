package dto

type DeviceReq struct {
	OS        string `json:"os" binding:"required,oneof=IOS ANDROID WEB WINDOW MAC LINUX" example:"IOS"`
	Name      string `json:"name" binding:"required,max=100" example:"iPhone 15"`
	UID       string `json:"uid" binding:"required,max=255" example:"3f2b8c1e-9a7d-4e21-b6c5-0d1e2f3a4b5c"`
	Version   string `json:"version" binding:"omitempty,max=50" example:"17.4"`
	UserAgent string `json:"-"`
}
