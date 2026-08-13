package domain

func ModelVersionFromLegacy(model Model) ModelVersion {
	return ModelVersion{
		ID:                model.ID,
		Name:              model.Name,
		Version:           model.Version,
		Lifecycle:         model.Lifecycle,
		Capabilities:      append([]string(nil), model.Capabilities...),
		EvaluationVersion: model.EvaluationVersion,
		License:           model.License,
		LicenseEvidence:   model.LicenseEvidence,
		Provenance:        model.Provenance,
		ArtifactDigest:    model.ArtifactDigest,
		ApprovalStatus:    model.ApprovalStatus,
		RiskLevel:         model.RiskLevel,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
	}
}

func (v ModelVersion) LegacyModel(mode DeploymentMode) Model {
	return Model{
		ID:                v.ID,
		Name:              v.Name,
		Version:           v.Version,
		DeploymentMode:    mode,
		Lifecycle:         v.Lifecycle,
		Capabilities:      append([]string(nil), v.Capabilities...),
		EvaluationVersion: v.EvaluationVersion,
		License:           v.License,
		LicenseEvidence:   v.LicenseEvidence,
		Provenance:        v.Provenance,
		ArtifactDigest:    v.ArtifactDigest,
		ApprovalStatus:    v.ApprovalStatus,
		RiskLevel:         v.RiskLevel,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
	}
}
