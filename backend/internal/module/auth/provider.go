package auth

import (
	"database/sql"

	"backend/internal/module/auth/delivery"
	"backend/internal/module/auth/port"
	authrepo "backend/internal/module/auth/repository"
	"backend/internal/module/auth/teams"
	"backend/internal/module/auth/usecase"
	characterrepo "backend/internal/module/character/repository"
	userrepo "backend/internal/module/user/repository"
)

type Provider struct {
	db             *sql.DB
	jwtSecret      string
	teamsClientID  string
	teamsTenantID  string
	defaultMapCode string

	authRepo             port.AuthRepository
	userReader           port.UserReader
	teamsTokenVerifier   port.TeamsTokenVerifier
	characterProvisioner port.CharacterProvisioner
	usecase              *usecase.AuthUsecase
	handler              *delivery.AuthHandler
}

func NewProvider(db *sql.DB, jwtSecret string, teamsClientID string, teamsTenantID string, defaultMapCode string) *Provider {
	return &Provider{db: db, jwtSecret: jwtSecret, teamsClientID: teamsClientID, teamsTenantID: teamsTenantID, defaultMapCode: defaultMapCode}
}

func (p *Provider) AuthRepository() port.AuthRepository {
	if p.authRepo == nil {
		p.authRepo = authrepo.NewAuthRepository(p.db)
	}
	return p.authRepo
}

// UserReader() là nơi cross-module wiring thật sự xảy ra: bind interface UserReader (định nghĩa
// trong chính package port của auth) bằng implementation cụ thể của module user. auth/usecase không
// bao giờ nhìn thấy user/repository.UserRepository — chỉ Provider mới được phép biết.
func (p *Provider) UserReader() port.UserReader {
	if p.userReader == nil {
		p.userReader = userrepo.NewUserRepository(p.db)
	}
	return p.userReader
}

func (p *Provider) TeamsTokenVerifier() port.TeamsTokenVerifier {
	if p.teamsTokenVerifier == nil {
		p.teamsTokenVerifier = teams.NewMicrosoftTokenVerifier(p.teamsClientID, p.teamsTenantID)
	}
	return p.teamsTokenVerifier
}

// CharacterProvisioner() bind trực tiếp characterrepo.CharacterRepository, cùng cross-module
// pattern với UserReader() ở trên: auth chỉ cần đúng 1 method CreateDefaultWithTx, không phụ
// thuộc vào character/port hay character/usecase.
func (p *Provider) CharacterProvisioner() port.CharacterProvisioner {
	if p.characterProvisioner == nil {
		p.characterProvisioner = characterrepo.NewCharacterRepository(p.db, p.defaultMapCode)
	}
	return p.characterProvisioner
}

func (p *Provider) Usecase() *usecase.AuthUsecase {
	if p.usecase == nil {
		p.usecase = usecase.NewAuthUsecase(p.db, p.AuthRepository(), p.UserReader(), p.TeamsTokenVerifier(), p.CharacterProvisioner(), p.jwtSecret)
	}
	return p.usecase
}

func (p *Provider) Handler() *delivery.AuthHandler {
	if p.handler == nil {
		p.handler = delivery.NewAuthHandler(p.Usecase())
	}
	return p.handler
}
