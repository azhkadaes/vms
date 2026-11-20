package policy

import (
	"testing"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func TestEachUploadPurposeAcceptsOnlyItsOwnContentTypes(t *testing.T) {
	for _, tc := range []struct {
		purpose, mime string
		want          bool
	}{
		{PurposeDocument, "application/pdf", true},
		{PurposeDocument, "image/jpeg", false},
		{PurposeDocument, "image/png", false},
		{PurposeVisitorPhoto, "image/jpeg", true},
		{PurposeVisitorPhoto, "image/png", true},
		{PurposeVisitorPhoto, "application/pdf", false},
		{PurposeVisitorPhoto, "image/gif", false},
		{"photo", "image/jpeg", false},
		{"", "application/pdf", false},
	} {
		if got := AcceptsUpload(tc.purpose, tc.mime); got != tc.want {
			t.Errorf("AcceptsUpload(%q, %q) = %v, want %v", tc.purpose, tc.mime, got, tc.want)
		}
	}
}

func TestOnlyKnownPurposesAreUploadPurposes(t *testing.T) {
	for purpose, want := range map[string]bool{"document": true, "visitorPhoto": true, "visitorphoto": false, "photo": false, "": false} {
		if got := UploadPurpose(purpose); got != want {
			t.Errorf("UploadPurpose(%q) = %v, want %v", purpose, got, want)
		}
	}
}

// Staff uploads feed staff visit creation, so they need a ready workspace and the create permission.
func TestStaffMayUploadOnlyFromAReadyWorkspaceWithTheCreatePermission(t *testing.T) {
	for _, tc := range []struct {
		name    string
		session dto.Session
		want    bool
	}{
		{"PA with create", dto.Session{Active: true, Role: RolePA, Permissions: []string{PermBookings, PermCreate}}, true},
		{"Protocol Officer with create", dto.Session{Active: true, Role: RoleProtocol, Permissions: []string{PermCreate}}, true},
		{"Receptionist without create", dto.Session{Active: true, Role: RoleReceptionist, Permissions: []string{PermBookings, PermCheckin, PermCheckout}}, false},
		{"Super Admin without create", dto.Session{Active: true, Role: RoleSuperAdmin, Permissions: []string{PermBookings, PermPeople}}, false},
		{"role selection pending", dto.Session{Active: true, RoleSelectionRequired: true, Permissions: []string{PermCreate}}, false},
		{"password change pending", dto.Session{Active: true, Role: RolePA, MustChangePassword: true, Permissions: []string{PermCreate}}, false},
		{"signed out", dto.Session{Role: RolePA, Permissions: []string{PermCreate}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanUpload(tc.session); got != tc.want {
				t.Fatalf("CanUpload = %v, want %v", got, tc.want)
			}
		})
	}
}
