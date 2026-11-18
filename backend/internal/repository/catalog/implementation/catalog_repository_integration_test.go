//go:build integration

package catalogrepositoryimplementation_test

import (
	"context"
	"testing"
	"time"

	implementation "github.com/azhkadaes/vms/backend/internal/repository/catalog/implementation"
	"github.com/azhkadaes/vms/backend/internal/repository/integrationtest"
)

func TestPublicCatalogReadsOnlyActiveDestinationsFromTheRebuiltSchema(t *testing.T) {
	db := integrationtest.Open(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	catalog, err := implementation.NewCatalogRepository(db).GetPublicCatalogs(ctx)
	if err != nil {
		t.Fatalf("catalog query failed against the rebuilt schema: %v", err)
	}
	for _, officer := range catalog.Officers {
		if !officer.Active || officer.ID <= 0 || officer.Name == "" || officer.JobTitle == "" {
			t.Fatalf("catalog returned an inactive or incomplete official: %+v", officer)
		}
	}
	for _, room := range catalog.Rooms {
		if !room.Active || room.ID <= 0 || room.Name == "" {
			t.Fatalf("catalog returned an inactive or incomplete room: %+v", room)
		}
	}
}
