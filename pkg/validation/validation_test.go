package validation

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/go-playground/validator/v10"
)

type deviceReq struct {
	OS string `json:"os" binding:"required,oneof=IOS ANDROID"`
}

type loginReq struct {
	Username string    `json:"username" binding:"required,min=2,max=5"`
	Email    string    `json:"email" binding:"omitempty,email"`
	Code     string    `json:"code" binding:"omitempty,len=4"`
	Age      int       `json:"age" binding:"omitempty,min=18,max=99"`
	Page     int       `form:"page" binding:"omitempty,gte=1"`
	Device   deviceReq `json:"device"`
}

func TestFromError(t *testing.T) {
	v := validator.New()
	v.SetTagName("binding")
	registerTagName(v)

	valid := loginReq{Username: "abc", Device: deviceReq{OS: "IOS"}}
	tests := []struct {
		name   string
		modify func(r *loginReq)
		want   FieldError
	}{
		{"required", func(r *loginReq) { r.Username = "" },
			FieldError{"username", i18n.FieldRequired, i18n.Params{"field": "username"}}},
		{"min length", func(r *loginReq) { r.Username = "a" },
			FieldError{"username", i18n.FieldMinLength, i18n.Params{"field": "username", "min": "2"}}},
		{"max length", func(r *loginReq) { r.Username = "abcdef" },
			FieldError{"username", i18n.FieldMaxLength, i18n.Params{"field": "username", "max": "5"}}},
		{"email", func(r *loginReq) { r.Email = "nope" },
			FieldError{"email", i18n.FieldInvalidEmail, i18n.Params{"field": "email"}}},
		{"exact length", func(r *loginReq) { r.Code = "12" },
			FieldError{"code", i18n.FieldExactLength, i18n.Params{"field": "code", "len": "4"}}},
		{"min value", func(r *loginReq) { r.Age = 10 },
			FieldError{"age", i18n.FieldMinValue, i18n.Params{"field": "age", "min": "18"}}},
		{"max value", func(r *loginReq) { r.Age = 100 },
			FieldError{"age", i18n.FieldMaxValue, i18n.Params{"field": "age", "max": "99"}}},
		{"form tag", func(r *loginReq) { r.Page = -1 },
			FieldError{"page", i18n.FieldMinValue, i18n.Params{"field": "page", "min": "1"}}},
		{"nested oneof", func(r *loginReq) { r.Device.OS = "BEOS" },
			FieldError{"device.os", i18n.FieldOneOf, i18n.Params{"field": "device.os", "values": "IOS, ANDROID"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			tt.modify(&req)
			got, ok := FromError(v.Struct(req))
			if !ok || len(got) != 1 {
				t.Fatalf("FromError() = %+v, %v; want one field error", got, ok)
			}
			if !reflect.DeepEqual(got[0], tt.want) {
				t.Errorf("FromError() = %+v, want %+v", got[0], tt.want)
			}
		})
	}
}

func TestFromErrorJSON(t *testing.T) {
	var req loginReq
	err := json.Unmarshal([]byte(`{"age":"old"}`), &req)
	got, ok := FromError(err)
	want := []FieldError{{"age", i18n.FieldInvalidType, i18n.Params{"field": "age", "type": "number"}}}
	if !ok || !reflect.DeepEqual(got, want) {
		t.Errorf("FromError(type error) = %+v, %v; want %+v", got, ok, want)
	}

	if _, ok := FromError(json.Unmarshal([]byte(`{`), &req)); ok {
		t.Error("FromError(syntax error) ok = true, want false")
	}
}
