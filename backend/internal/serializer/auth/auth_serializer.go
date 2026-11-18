package authserializer

import (
	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type AuthSerializer interface {
	ToSession(snapshot entity.SessionSnapshot) dto.Session
}
