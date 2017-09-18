package install

func (ae *ansibleExecutor) SeedRegistry(plan Plan) error {
	cc, err := ae.buildClusterCatalog(&plan)
	if err != nil {
		return err
	}
	t := task{
		name:           "seed-registry",
		playbook:       "seed-registry.yaml",
		plan:           plan,
		inventory:      buildInventoryFromPlan(&plan),
		clusterCatalog: *cc,
		explainer:      ae.defaultExplainer(),
	}
	return ae.execute(t)
}
