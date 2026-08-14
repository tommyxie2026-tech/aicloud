package repository

func (r *PostgresRepositories) DeploymentRepository() *PostgresDeployments {
	if r == nil || r.DB == nil {
		return nil
	}
	return &PostgresDeployments{db: r.DB}
}

func (r *PostgresRepositories) DeploymentLifecycleEvents() *PostgresDeploymentLifecycleEvents {
	if r == nil || r.DB == nil {
		return nil
	}
	return NewPostgresDeploymentLifecycleEvents(r.DB)
}
