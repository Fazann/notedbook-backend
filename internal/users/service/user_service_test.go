package service

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/Fazann/notedbook-backend/internal/users/dto"
	"github.com/Fazann/notedbook-backend/internal/users/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/utils"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// fakeUserRepo keeps users in memory; methods the service does not call panic via the nil embed.
type fakeUserRepo struct {
	repo.UserRepoImpl
	users   map[uint]*model.User
	updates map[string]any
}

func (r *fakeUserRepo) FindOne(_ context.Context, cond ...any) (*model.User, error) {
	if u, ok := r.users[cond[0].(uint)]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepo) FindByIDWithAvatar(ctx context.Context, id uint) (*model.User, error) {
	return r.FindOne(ctx, id)
}

func (r *fakeUserRepo) find(match func(*model.User) bool) (*model.User, error) {
	for _, u := range r.users {
		if match(u) {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepo) FindByUsername(_ context.Context, username string) (*model.User, error) {
	return r.find(func(u *model.User) bool { return u.Username == username })
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (*model.User, error) {
	return r.find(func(u *model.User) bool { return u.Email == email })
}

func (r *fakeUserRepo) Update(_ context.Context, _ any, updates map[string]any) error {
	r.updates = updates
	return nil
}

type fakeAttachments struct {
	err     error
	called  []uint
	deleted []uint
}

func (f *fakeAttachments) MarkUsed(_ context.Context, id uint, _ ...enums.AttachmentTypeEnum) error {
	f.called = append(f.called, id)
	return f.err
}

func (f *fakeAttachments) Delete(_ context.Context, id uint, _ uint) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func ptr[T any](v T) *T { return &v }

func TestUpdateProfile(t *testing.T) {
	tests := []struct {
		name          string
		req           dto.UpdateProfileReq
		attachmentErr error
		wantStatus    int
		wantErr       error
		wantUpdates   map[string]any
		wantMarked    []uint
		wantDeleted   []uint
		noAvatar      bool
	}{
		{name: "nothing sent", wantStatus: http.StatusOK},
		{name: "all fields", wantStatus: http.StatusOK,
			req:         dto.UpdateProfileReq{Fullname: ptr(" New Name "), Username: ptr("newname"), Email: ptr("New@Example.com"), AvatarID: ptr(uint(7))},
			wantUpdates: map[string]any{"fullname": "New Name", "username": "newname", "email": "new@example.com", "avatar_id": uint(7), "updated_by": uint(1)},
			wantMarked:  []uint{7}, wantDeleted: []uint{5}},
		{name: "first avatar deletes nothing", wantStatus: http.StatusOK, noAvatar: true,
			req:         dto.UpdateProfileReq{AvatarID: ptr(uint(7))},
			wantUpdates: map[string]any{"avatar_id": uint(7), "updated_by": uint(1)},
			wantMarked:  []uint{7}},
		{name: "same username and email are not re-checked", wantStatus: http.StatusOK,
			req: dto.UpdateProfileReq{Username: ptr("me"), Email: ptr("me@example.com")}},
		{name: "same avatar is not re-marked", wantStatus: http.StatusOK,
			req: dto.UpdateProfileReq{AvatarID: ptr(uint(5))}},
		{name: "username taken", req: dto.UpdateProfileReq{Username: ptr("other")},
			wantStatus: http.StatusConflict, wantErr: i18n.UsernameTaken},
		{name: "email taken", req: dto.UpdateProfileReq{Email: ptr("other@example.com")},
			wantStatus: http.StatusConflict, wantErr: i18n.EmailTaken},
		{name: "invalid username", req: dto.UpdateProfileReq{Username: ptr("bad!")},
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidUsername},
		{name: "invalid email", req: dto.UpdateProfileReq{Email: ptr("nope")},
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidEmail},
		{name: "blank fullname", req: dto.UpdateProfileReq{Fullname: ptr("   ")},
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidFullname},
		{name: "avatar not claimed when username is taken", req: dto.UpdateProfileReq{Username: ptr("other"), AvatarID: ptr(uint(7))},
			wantStatus: http.StatusConflict, wantErr: i18n.UsernameTaken},
		{name: "avatar in use", req: dto.UpdateProfileReq{AvatarID: ptr(uint(7))}, attachmentErr: i18n.AttachmentInUse,
			wantStatus: http.StatusConflict, wantErr: i18n.AttachmentInUse, wantMarked: []uint{7}},
		{name: "avatar not found", req: dto.UpdateProfileReq{AvatarID: ptr(uint(7))}, attachmentErr: i18n.RecordNotFound,
			wantStatus: http.StatusNotFound, wantErr: i18n.RecordNotFound, wantMarked: []uint{7}},
		{name: "avatar not an image", req: dto.UpdateProfileReq{AvatarID: ptr(uint(7))},
			attachmentErr: i18n.Err(i18n.InvalidAttachmentType, i18n.Params{"types": "image"}),
			wantStatus:    http.StatusBadRequest, wantErr: i18n.InvalidAttachmentType, wantMarked: []uint{7}},
		{name: "attachment db error", req: dto.UpdateProfileReq{AvatarID: ptr(uint(7))}, attachmentErr: errors.New("db down"),
			wantStatus: http.StatusInternalServerError, wantMarked: []uint{7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := &fakeUserRepo{users: map[uint]*model.User{
				1: {BaseModelID: model.BaseModelID{ID: 1}, Username: "me", Email: "me@example.com", AvatarID: ptr(uint(5))},
				2: {BaseModelID: model.BaseModelID{ID: 2}, Username: "other", Email: "other@example.com"},
			}}
			if tt.noAvatar {
				users.users[1].AvatarID = nil
			}
			attachments := &fakeAttachments{err: tt.attachmentErr}
			svc := &userSvc{userRepo: users, attachments: attachments}

			res := svc.UpdateProfile(context.Background(), 1, &tt.req)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if tt.wantErr != nil {
				key, _, _ := i18n.FromError(res.Error)
				if key != tt.wantErr {
					t.Errorf("error = %v, want %v", res.Error, tt.wantErr)
				}
			}
			if !reflect.DeepEqual(users.updates, tt.wantUpdates) {
				t.Errorf("updates = %v, want %v", users.updates, tt.wantUpdates)
			}
			if !reflect.DeepEqual(attachments.called, tt.wantMarked) {
				t.Errorf("MarkUsed calls = %v, want %v", attachments.called, tt.wantMarked)
			}
			if !reflect.DeepEqual(attachments.deleted, tt.wantDeleted) {
				t.Errorf("Delete calls = %v, want %v", attachments.deleted, tt.wantDeleted)
			}
		})
	}
}

func TestChangePassword(t *testing.T) {
	hash, err := utils.GenerateHash("oldpass")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		current    string
		disabled   bool
		wantStatus int
		wantErr    error
	}{
		{name: "changed", current: "oldpass", wantStatus: http.StatusOK},
		{name: "wrong current password", current: "wrong", wantStatus: http.StatusBadRequest, wantErr: i18n.IncorrectPassword},
		{name: "disabled account", current: "oldpass", disabled: true, wantStatus: http.StatusForbidden, wantErr: i18n.AccountDisabled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := &fakeUserRepo{users: map[uint]*model.User{
				1: {BaseModelID: model.BaseModelID{ID: 1}, Username: "me", Password: hash, IsDisabled: tt.disabled},
			}}
			svc := &userSvc{userRepo: users}

			res := svc.ChangePassword(context.Background(), &dto.ChangePasswordReq{UserID: 1, CurrentPassword: tt.current, NewPassword: "newpass"})

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if tt.wantErr != nil {
				if key, _, _ := i18n.FromError(res.Error); key != tt.wantErr {
					t.Errorf("error = %v, want %v", res.Error, tt.wantErr)
				}
				if users.updates != nil {
					t.Errorf("updates = %v, want none", users.updates)
				}
				return
			}
			if _, ok := res.Data.(*dto.ProfileRes); !ok {
				t.Errorf("data = %T, want *dto.ProfileRes", res.Data)
			}
			newHash, _ := users.updates["password"].(string)
			if utils.CompareHash(newHash, "newpass") != nil {
				t.Errorf("stored password does not match the new one")
			}
		})
	}
}
