package identity

// NewProjectSystemPrincipal creates an internal System Principal that is
// explicitly constrained to one Tenant/Project. Broad administrative database
// access uses a separate admin repository and database credential.
func NewProjectSystemPrincipal(subjectID, purpose, tenantID, projectID string, capabilities ...string) (Principal, error) {
	principal := Principal{
		Type:         PrincipalSystem,
		SubjectID:    subjectID,
		TenantID:     tenantID,
		ProjectID:    projectID,
		Capabilities: append([]string(nil), capabilities...),
		AuthnMethod:  "internal_system",
		Issuer:       "aicloud",
		Purpose:      purpose,
	}
	if err := Validate(principal); err != nil {
		return Principal{}, err
	}
	return normalize(principal), nil
}
