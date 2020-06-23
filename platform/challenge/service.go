package challenge

import (
	"context"
	"errors"
	"github.com/micromdm/scep/challenge"
)

type Service interface {
	SCEPChallenge(ctx context.Context) (string, error)
}

type ChallengeService struct {
	scepChallengeStore challenge.Store
}

func (c *ChallengeService) SCEPChallenge(ctx context.Context) (string, error) {
	if c.scepChallengeStore == nil {
		return "", errors.New("SCEP challenge store missing")
	}
	return c.scepChallengeStore.SCEPChallenge()
}

func NewService(cs challenge.Store) *ChallengeService {
	return &ChallengeService{
		scepChallengeStore: cs,
	}
}
