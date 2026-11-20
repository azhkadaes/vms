package roleserializerimplementation_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	implementation "github.com/azhkadaes/vms/backend/internal/serializer/role/implementation"
)

// The frontend's roleDefinitionSchema requires every key and rejects null lists.
func TestARoleDefinitionCarriesEveryKeyWithListsNeverNull(t *testing.T) {
	out := implementation.NewRoleSerializer().ToRoleDefinitions([]entity.Role{{ID: 3, Name: policy.RoleReceptionist, Version: 2, PolicyID: 0}})
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"id": "3", "name": "Receptionist", "description": policy.RoleDescription(policy.RoleReceptionist), "permissions": []any{},
		"assignablePermissions": []any{"bookings", "checkin", "checkout"}, "editable": false, "version": float64(2)}
	if len(decoded) != 1 || len(decoded[0]) != len(want) {
		t.Fatalf("got %s", raw)
	}
	for key, value := range want {
		got, _ := json.Marshal(decoded[0][key])
		expected, _ := json.Marshal(value)
		if string(got) != string(expected) {
			t.Errorf("%s = %s, want %s", key, got, expected)
		}
	}
	empty, _ := json.Marshal(implementation.NewRoleSerializer().ToRoleDefinitions(nil))
	if string(empty) != "[]" {
		t.Fatalf("no roles = %s, want []", empty)
	}
}

func TestARoleDefinitionDoesNotShareTheStoredPermissionList(t *testing.T) {
	stored := []string{"bookings", "checkin"}
	out := implementation.NewRoleSerializer().ToRoleDefinition(entity.Role{ID: 3, Name: policy.RoleReceptionist, Version: 1, PolicyID: 3, Permissions: stored})
	out.Permissions[0] = "changed"
	if !slices.Equal(stored, []string{"bookings", "checkin"}) || !out.Editable {
		t.Fatalf("stored = %v, editable = %v", stored, out.Editable)
	}
}
