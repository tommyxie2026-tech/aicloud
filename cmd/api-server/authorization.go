package main

import "github.com/tommyxie2026-tech/aicloud/internal/authorization"

func buildAPIAuthorizer() authorization.Authorizer {
	return authorization.NewDefault()
}
