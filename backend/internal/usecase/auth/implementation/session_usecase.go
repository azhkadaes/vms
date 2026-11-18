package authusecaseimplementation

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxRevision int64 = 9007199254740991

type sessionUsecase struct {
	repo                 authrepository.SessionRepository
	reader               authusecase.AuthUsecase
	verifier             authusecase.CredentialVerifier
	tokens               security.Tokens
	rateKey, recoveryKey []byte
}

func NewSessionUsecase(repo authrepository.SessionRepository, reader authusecase.AuthUsecase, verifier authusecase.CredentialVerifier, tokens security.Tokens, rateKey, recoveryKey []byte) authusecase.SessionUsecase {
	return &sessionUsecase{repo: repo, reader: reader, verifier: verifier, tokens: tokens, rateKey: rateKey, recoveryKey: recoveryKey}
}
func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if len(email) > 254 || len(email) == 0 {
		return "", invalidEmail()
	}
	for _, c := range email {
		if c > 127 || c < 33 {
			return "", invalidEmail()
		}
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", invalidEmail()
	}
	return email, nil
}
func invalidEmail() error {
	return failure.Validation().Add("email", "Masukkan alamat email yang valid.").Err()
}

// missing reports a repository not-found, from either the legacy GORM sentinel or failure.ErrNotFound.
func missing(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, failure.ErrNotFound)
}

// signedOutIfMissing treats a vanished session, credential or employee row as signed out, not a 500.
func signedOutIfMissing(err error) error {
	if missing(err) {
		return authusecase.ErrUnauthenticated
	}
	return err
}
func validSession(s entity.StaffSession, credential entity.EmployeeCredential, now time.Time) bool {
	return credential.Active && credential.EmployeeID == s.EmployeeID && s.RevokedAt == nil && now.Before(s.IdleExpiresAt) && now.Before(s.AbsoluteExpiresAt) && !now.Before(s.CreatedAt)
}
func proofFor(s entity.StaffSession) entity.SessionProof {
	return entity.SessionProof{SessionID: s.ID, EmployeeID: s.EmployeeID, CredentialGeneration: s.CredentialGeneration, RoleRevision: s.RoleRevision}
}
func (u *sessionUsecase) result(ctx context.Context, s entity.StaffSession, secret string) (authusecase.CredentialResult, error) {
	session, err := u.reader.GetSession(ctx, proofFor(s))
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	// API timestamps are UTC with Z regardless of the host's local zone.
	now := time.UnixMilli(session.Timing.ServerTime).UTC()
	expiry := minTime(now.Add(5*time.Minute), session.Timing.IdleExpiresAt, session.Timing.AbsoluteExpiresAt).UTC()
	token, err := u.tokens.Sign(proofFor(s), now, expiry)
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	return authusecase.CredentialResult{Result: dto.TokenResult{AccessToken: token, TokenType: "Bearer", AccessExpiresAt: expiry, Session: session}, Refresh: secret, SessionID: s.ID, AbsoluteExpiresAt: s.AbsoluteExpiresAt}, nil
}
func minTime(values ...time.Time) time.Time {
	out := values[0]
	for _, v := range values[1:] {
		if v.Before(out) {
			out = v
		}
	}
	return out
}
func (u *sessionUsecase) Login(ctx context.Context, req dto.Login, ip string) (authusecase.CredentialResult, error) {
	email, err := NormalizeEmail(req.Email)
	if err != nil {
		return authusecase.CredentialResult{}, failure.New(401, "invalid_credentials")
	}
	reservation, retry, err := u.repo.ReserveLogin(ctx, security.Digest(u.rateKey, "login-ip", ip), security.Digest(u.rateKey, "login-account", email))
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	if retry > 0 {
		return authusecase.CredentialResult{}, &failure.Error{Status: 429, Code: "rate_limited", RetryAfter: retry}
	}
	credential, err := u.repo.GetCredential(ctx, email)
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	matched, err := u.verifier.Verify(ctx, credential.EncodedHash, req.Password)
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	if !matched || !credential.Active || credential.EmployeeID <= 0 {
		return authusecase.CredentialResult{}, failure.New(401, "invalid_credentials")
	}
	secret, err := security.Random()
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	var session entity.StaffSession
	err = u.repo.Transaction(ctx, credential.EmployeeID, func(tx authrepository.SessionTransaction) error {
		current, err := tx.Credential(credential.EmployeeID)
		if err != nil {
			return err
		}
		if !current.Active || current.EncodedHash != credential.EncodedHash {
			return failure.New(401, "invalid_credentials")
		}
		roles, err := tx.Roles(credential.EmployeeID)
		if err != nil {
			return err
		}
		if len(roles) == 0 {
			return failure.New(401, "invalid_credentials")
		}
		now := tx.Now()
		session = entity.StaffSession{ID: id.String(), EmployeeID: credential.EmployeeID, CredentialGeneration: 1, RoleRevision: 1, RefreshHash: security.Hash(secret), CreatedAt: now, LastActivityAt: now, IdleExpiresAt: now.Add(8 * time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour)}
		if len(roles) == 1 {
			for _, id := range roles {
				selected := id
				session.ActiveRoleID = &selected
			}
		}
		if err := tx.Create(session); err != nil {
			return err
		}
		return tx.Audit(credential.EmployeeID, "session.login")
	})
	if missing(err) {
		return authusecase.CredentialResult{}, failure.New(401, "invalid_credentials")
	}
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	if err = u.repo.FinishLogin(ctx, reservation, true); err != nil {
		return authusecase.CredentialResult{}, err
	}
	return u.result(ctx, session, secret)
}
func (u *sessionUsecase) sessionForSecret(ctx context.Context, secret string) (string, entity.SessionSnapshot, error) {
	if len(secret) != 43 {
		return "", entity.SessionSnapshot{}, authusecase.ErrUnauthenticated
	}
	id, err := u.repo.FindRefresh(ctx, security.Hash(secret))
	if err != nil {
		return "", entity.SessionSnapshot{}, signedOutIfMissing(err)
	}
	snapshot, err := u.repo.GetSessionSnapshot(ctx, id)
	return id, snapshot, signedOutIfMissing(err)
}
func (u *sessionUsecase) Binding(ctx context.Context, secret string, allowConsumed bool) (string, time.Time, error) {
	id, snapshot, err := u.sessionForSecret(ctx, secret)
	if err != nil {
		return "", time.Time{}, err
	}
	err = u.repo.Transaction(ctx, snapshot.EmployeeID, func(tx authrepository.SessionTransaction) error {
		s, err := tx.Session(id)
		if err != nil {
			return err
		}
		c, err := tx.Credential(s.EmployeeID)
		if err != nil {
			return err
		}
		if !validSession(s, c, tx.Now()) || (!allowConsumed && s.RefreshHash != security.Hash(secret)) {
			return authusecase.ErrUnauthenticated
		}
		return nil
	})
	return id, snapshot.AbsoluteExpiresAt, signedOutIfMissing(err)
}
func (u *sessionUsecase) Refresh(ctx context.Context, secret, requestID string) (authusecase.CredentialResult, error) {
	parsed, err := uuid.Parse(requestID)
	if err != nil || parsed == uuid.Nil || parsed.String() != requestID {
		return authusecase.CredentialResult{}, failure.Validation().Add("refreshRequestId", "ID permintaan pembaruan tidak valid.").Err()
	}
	id, snapshot, err := u.sessionForSecret(ctx, secret)
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	var session entity.StaffSession
	var successor string
	var reuse bool
	err = u.repo.Transaction(ctx, snapshot.EmployeeID, func(tx authrepository.SessionTransaction) error {
		var err error
		session, err = tx.Session(id)
		if err != nil {
			return err
		}
		credential, err := tx.Credential(session.EmployeeID)
		if err != nil {
			return err
		}
		if !validSession(session, credential, tx.Now()) {
			return authusecase.ErrUnauthenticated
		}
		hash := security.Hash(secret)
		recovery, err := tx.Recovery(id, requestID)
		if err == nil && tx.Now().Before(recovery.ExpiresAt) && session.RefreshHash == recovery.SuccessorHash && (hash == recovery.PredecessorHash || hash == recovery.SuccessorHash) {
			successor, err = security.Open(u.recoveryKey, recovery.SuccessorCipher, id+":"+requestID)
			return err
		}
		if err != nil && !missing(err) {
			return err
		}
		if hash != session.RefreshHash {
			now := tx.Now()
			session.RevokedAt = &now
			reuse = true
			if err := tx.ClearRecoveries(id); err != nil {
				return err
			}
			if err := tx.Save(session); err != nil {
				return err
			}
			return tx.Audit(session.EmployeeID, "session.refresh_reuse")
		}
		if err == nil {
			return failure.New(409, "idempotency_conflict")
		}
		successor, err = security.Random()
		if err != nil {
			return err
		}
		cipher, err := security.Seal(u.recoveryKey, successor, id+":"+requestID)
		if err != nil {
			return err
		}
		if err := tx.RememberRefresh(id, session.RefreshHash); err != nil {
			return err
		}
		session.RefreshHash = security.Hash(successor)
		if err := tx.Save(session); err != nil {
			return err
		}
		return tx.SaveRecovery(entity.SessionRecovery{SessionID: id, RequestID: requestID, PredecessorHash: hash, SuccessorHash: session.RefreshHash, SuccessorCipher: cipher, ExpiresAt: tx.Now().Add(10 * time.Second)})
	})
	if err != nil {
		return authusecase.CredentialResult{}, signedOutIfMissing(err)
	}
	if reuse {
		return authusecase.CredentialResult{}, authusecase.ErrUnauthenticated
	}
	return u.result(ctx, session, successor)
}
func (u *sessionUsecase) Logout(ctx context.Context, secret string) error {
	id, snapshot, err := u.sessionForSecret(ctx, secret)
	if errors.Is(err, authusecase.ErrUnauthenticated) {
		return nil
	}
	if err != nil {
		return err
	}
	err = u.repo.Transaction(ctx, snapshot.EmployeeID, func(tx authrepository.SessionTransaction) error {
		s, err := tx.Session(id)
		if err != nil {
			return err
		}
		if s.RevokedAt != nil {
			return nil
		}
		now := tx.Now()
		s.RevokedAt = &now
		if err := tx.Save(s); err != nil {
			return err
		}
		if err := tx.ClearRecoveries(id); err != nil {
			return err
		}
		return tx.Audit(s.EmployeeID, "session.logout")
	})
	if missing(err) {
		return nil
	}
	return err
}
func (u *sessionUsecase) mutate(ctx context.Context, proof entity.SessionProof, operation func(authrepository.SessionTransaction, *entity.StaffSession) error) (entity.StaffSession, error) {
	var session entity.StaffSession
	err := u.repo.Transaction(ctx, proof.EmployeeID, func(tx authrepository.SessionTransaction) error {
		var err error
		session, err = tx.Session(proof.SessionID)
		if err != nil {
			return err
		}
		c, err := tx.Credential(proof.EmployeeID)
		if err != nil {
			return err
		}
		if !validSession(session, c, tx.Now()) || session.EmployeeID != proof.EmployeeID || session.CredentialGeneration != proof.CredentialGeneration {
			return authusecase.ErrUnauthenticated
		}
		if session.RoleRevision != proof.RoleRevision {
			return authusecase.ErrRoleChanged
		}
		if err := operation(tx, &session); err != nil {
			return err
		}
		return tx.Save(session)
	})
	return session, signedOutIfMissing(err)
}
func (u *sessionUsecase) Renew(ctx context.Context, proof entity.SessionProof) (dto.Session, error) {
	_, err := u.mutate(ctx, proof, func(tx authrepository.SessionTransaction, s *entity.StaffSession) error {
		if tx.Now().Sub(s.LastActivityAt) >= time.Minute {
			s.LastActivityAt = tx.Now()
			s.IdleExpiresAt = minTime(tx.Now().Add(8*time.Hour), s.AbsoluteExpiresAt)
		}
		return nil
	})
	if err != nil {
		return dto.Session{}, err
	}
	return u.reader.GetSession(ctx, proof)
}
func (u *sessionUsecase) SwitchRole(ctx context.Context, proof entity.SessionProof, role string) (authusecase.CredentialResult, error) {
	session, err := u.mutate(ctx, proof, func(tx authrepository.SessionTransaction, s *entity.StaffSession) error {
		roles, err := tx.Roles(proof.EmployeeID)
		if err != nil {
			return err
		}
		id, ok := roles[role]
		if !ok {
			return failure.New(403, "forbidden")
		}
		if s.RoleRevision >= maxRevision {
			return failure.New(409, "role_changed")
		}
		s.ActiveRoleID = &id
		s.RoleRevision++
		return tx.Audit(s.EmployeeID, "session.role_changed")
	})
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	return u.result(ctx, session, "")
}
func (u *sessionUsecase) ChangePassword(ctx context.Context, proof entity.SessionProof, req dto.PasswordChange) (authusecase.CredentialResult, error) {
	snapshot, err := u.repo.GetSessionSnapshot(ctx, proof.SessionID)
	if err != nil {
		return authusecase.CredentialResult{}, signedOutIfMissing(err)
	}
	if snapshot.EmployeeID != proof.EmployeeID {
		return authusecase.CredentialResult{}, authusecase.ErrUnauthenticated
	}
	var credential entity.EmployeeCredential
	err = u.repo.Transaction(ctx, proof.EmployeeID, func(tx authrepository.SessionTransaction) error {
		var err error
		credential, err = tx.Credential(proof.EmployeeID)
		return err
	})
	if err != nil {
		return authusecase.CredentialResult{}, signedOutIfMissing(err)
	}
	ok, err := u.verifier.Verify(ctx, credential.EncodedHash, req.CurrentPassword)
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	if !ok {
		// A field error keeps the session: an authenticated 401 would sign the user out (V-10).
		return authusecase.CredentialResult{}, failure.Validation().Add("currentPassword", "Kata sandi saat ini tidak sesuai.").Err()
	}
	if req.NewPassword == req.CurrentPassword {
		return authusecase.CredentialResult{}, failure.Validation().Add("newPassword", "Gunakan kata sandi baru yang berbeda.").Err()
	}
	encoded, err := u.verifier.Hash(ctx, req.NewPassword)
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	secret, err := security.Random()
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	session, err := u.mutate(ctx, proof, func(tx authrepository.SessionTransaction, s *entity.StaffSession) error {
		current, err := tx.Credential(proof.EmployeeID)
		if err != nil {
			return err
		}
		if current.EncodedHash != credential.EncodedHash {
			return authusecase.ErrUnauthenticated
		}
		if s.CredentialGeneration >= maxRevision {
			return failure.New(409, "version_conflict")
		}
		if err := tx.SetPassword(proof.EmployeeID, encoded, s.ID); err != nil {
			return err
		}
		if err := tx.RememberRefresh(s.ID, s.RefreshHash); err != nil {
			return err
		}
		s.CredentialGeneration++
		s.RefreshHash = security.Hash(secret)
		return tx.Audit(s.EmployeeID, "session.password_changed")
	})
	if err != nil {
		return authusecase.CredentialResult{}, err
	}
	return u.result(ctx, session, secret)
}
