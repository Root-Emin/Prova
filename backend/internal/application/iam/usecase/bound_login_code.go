package usecase

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
)

var errMissingBoundAccount = errors.New("account-bound login code requires an account")

// generateLoginCode selects the account-bound contract when the configured
// service supports it. The legacy fallback exists only for old test doubles
// and deployments during a rolling upgrade; the production implementation is
// always AccountBoundLoginCodeService.
func generateLoginCode(codeSvc service.LoginCodeService, user *model.User, email string) (string, string, error) {
	if bound, ok := codeSvc.(service.AccountBoundLoginCodeService); ok {
		if user == nil || user.ID == uuid.Nil {
			return "", "", errMissingBoundAccount
		}
		return bound.GenerateFor(user.ID, email)
	}
	return codeSvc.Generate()
}

func matchesLoginCode(codeSvc service.LoginCodeService, user *model.User, email, digest, code string) bool {
	if bound, ok := codeSvc.(service.AccountBoundLoginCodeService); ok {
		return user != nil && bound.MatchesFor(user.ID, email, digest, code)
	}
	return codeSvc.Matches(digest, code)
}

func consumeLoginCode(ctx context.Context, codes repository.LoginCodeRepository, id uuid.UUID, at time.Time) (bool, error) {
	if atomic, ok := codes.(repository.AtomicLoginCodeRepository); ok {
		return atomic.MarkConsumedIfActive(ctx, id, at)
	}
	if err := codes.MarkConsumed(ctx, id, at); err != nil {
		return false, err
	}
	return true, nil
}
