package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	iamEvent "github.com/masterfabric-go/masterfabric/internal/domain/iam/event"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"github.com/masterfabric-go/masterfabric/internal/shared/events"
)

const (
	inviteRoleEmployee = "employee"
	inviteRoleOrgAdmin = "org-admin"

	organisationRoleEmployee = "trainee"
	organisationRoleAdmin    = "admin"
)

// InviteUserUseCase provisions an account and an invited organisation
// membership. Mail delivery is intentionally absent: Desktop requests the one
// login code when the recipient is ready to verify their mailbox.
type InviteUserUseCase struct {
	users    repository.UserRepository
	orgUsers repository.OrgUserRepository
	roles    repository.RoleRepository
	events   events.EventBus
	now      func() time.Time
}

func NewInviteUserUseCase(
	users repository.UserRepository,
	orgUsers repository.OrgUserRepository,
	roles repository.RoleRepository,
	eventBus events.EventBus,
) *InviteUserUseCase {
	return &InviteUserUseCase{users: users, orgUsers: orgUsers, roles: roles, events: eventBus, now: time.Now}
}

func (uc *InviteUserUseCase) Execute(ctx context.Context, req dto.InviteUserRequest) (*dto.InviteUserResponse, error) {
	email := model.NormalizeEmail(req.Email)
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	if !validEmail(email) {
		return nil, domainErr.New(domainErr.ErrValidation, "valid email is required", nil)
	}
	if firstName == "" || len(firstName) > 255 || len(lastName) > 255 {
		return nil, domainErr.New(domainErr.ErrValidation, "first name is required", nil)
	}
	if req.OrganizationID == uuid.Nil || req.InvitedBy == uuid.Nil {
		return nil, domainErr.New(domainErr.ErrUnauthorized, "administrator session is required", nil)
	}

	role, err := uc.roleForInvite(ctx, req.OrganizationID, req.Role)
	if err != nil {
		return nil, err
	}

	user, err := uc.users.GetByEmail(ctx, email)
	created := false
	switch {
	case err == nil:
		if user.IsDeleted() {
			return nil, domainErr.New(domainErr.ErrAlreadyExists, "an account already exists for this email", nil)
		}
	case errors.Is(err, domainErr.ErrNotFound):
		user = &model.User{
			Email: email, FirstName: firstName, LastName: lastName,
			Status: model.UserStatusInactive,
		}
		if err := uc.users.Create(ctx, user); err != nil {
			return nil, err
		}
		created = true
		if uc.events != nil {
			_ = uc.events.Publish(ctx, events.TopicIAM, iamEvent.UserRegistered{
				UserID: user.ID, Email: user.Email, Timestamp: uc.now().UTC(),
			})
		}
	default:
		return nil, err
	}

	// Existing accounts can be members of several organisations. A duplicate
	// invite never downgrades an active membership back to pending.
	membership, err := uc.orgUsers.GetByOrgAndUser(ctx, req.OrganizationID, user.ID)
	if err != nil && !errors.Is(err, domainErr.ErrNotFound) {
		return nil, err
	}
	if errors.Is(err, domainErr.ErrNotFound) {
		if err := uc.orgUsers.Add(ctx, &model.OrganizationUser{
			OrganizationID: req.OrganizationID,
			UserID:         user.ID,
			Status:         model.OrgUserStatusInvited,
			InvitedBy:      &req.InvitedBy,
		}); err != nil {
			if created {
				_ = uc.users.Delete(ctx, user.ID)
			}
			return nil, err
		}
	} else if membership == nil {
		return nil, domainErr.New(domainErr.ErrInternal, "organization membership is unavailable", nil)
	}

	if err := uc.roles.AssignRoleToUser(ctx, &model.UserRole{
		UserID: user.ID, RoleID: role.ID, OrganizationID: req.OrganizationID,
	}); err != nil {
		return nil, err
	}

	return &dto.InviteUserResponse{Invited: true, Email: user.Email}, nil
}

func (uc *InviteUserUseCase) roleForInvite(ctx context.Context, orgID uuid.UUID, input string) (*model.Role, error) {
	var wanted string
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "", inviteRoleEmployee:
		wanted = organisationRoleEmployee
	case inviteRoleOrgAdmin, "org_admin":
		wanted = organisationRoleAdmin
	default:
		return nil, domainErr.New(domainErr.ErrValidation, "invalid organization role", nil)
	}

	roles, err := uc.roles.ListByScope(ctx, model.ScopeTypeOrganization, orgID)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.Name == wanted {
			return role, nil
		}
	}
	return nil, domainErr.New(domainErr.ErrValidation, "the organization does not have the required role", nil)
}

// OrganizationUserRecord is the manager-facing view of a person and the
// machines they have paired after a successful Desktop sign-in.
type OrganizationUserRecord struct {
	User       *model.User
	Membership *model.OrganizationUser
	Devices    []*model.Device
}

type ListOrganizationUsersUseCase struct {
	users    repository.UserRepository
	orgUsers repository.OrgUserRepository
	devices  repository.DeviceRepository
}

func NewListOrganizationUsersUseCase(
	users repository.UserRepository,
	orgUsers repository.OrgUserRepository,
	devices repository.DeviceRepository,
) *ListOrganizationUsersUseCase {
	return &ListOrganizationUsersUseCase{users: users, orgUsers: orgUsers, devices: devices}
}

func (uc *ListOrganizationUsersUseCase) Execute(ctx context.Context, orgID uuid.UUID) ([]*OrganizationUserRecord, error) {
	memberships, _, err := uc.orgUsers.ListByOrg(ctx, orgID, 0, 500)
	if err != nil {
		return nil, err
	}

	records := make([]*OrganizationUserRecord, 0, len(memberships))
	for _, membership := range memberships {
		if membership == nil || membership.Status == model.OrgUserStatusRemoved {
			continue
		}
		user, err := uc.users.GetByID(ctx, membership.UserID)
		if err != nil {
			if errors.Is(err, domainErr.ErrNotFound) {
				continue
			}
			return nil, err
		}
		devices, err := uc.devices.ListByUser(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		records = append(records, &OrganizationUserRecord{User: user, Membership: membership, Devices: devices})
	}
	return records, nil
}
