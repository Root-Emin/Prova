package dto

import "github.com/google/uuid"

// InviteUserRequest is an administrator's request to provision a person in
// their current organisation. It creates no e-mail and never starts a session;
// the recipient proves mailbox ownership from Desktop with a login code.
type InviteUserRequest struct {
	Email          string
	FirstName      string
	LastName       string
	Role           string
	OrganizationID uuid.UUID
	InvitedBy      uuid.UUID
}

type InviteUserResponse struct {
	Invited bool
	Email   string
}
