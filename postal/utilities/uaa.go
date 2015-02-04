package utilities

import "github.com/pivotal-cf/uaa-sso-golang/uaa"

type UAAInterface interface {
	uaa.UsersGUIDsByScopeInterface
	uaa.AllUsersInterface
}
