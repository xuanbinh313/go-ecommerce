package workflows

import (
	"context"
	"strings"

	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	rbacmodels "github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
)

const GetAssignableRolesWorkflowID = "get-assignable-roles-workflow"
const GetAssignableRolesStepID = "get-assignable-roles"
const GetAssignablePoliciesWorkflowID = "get-assignable-policies-workflow"
const GetAssignablePoliciesStepID = "get-assignable-policies"

// ListAssignableRoles resolves the actor's role grants, pages the filtered
// candidates, then keeps only roles whose policies are covered by those grants.
func ListAssignableRoles(ctx context.Context, catalog UserRoleCatalog, links linkdomain.Repository, actorID, actorType, query string, ids, names []string, limit, offset int) ([]rbacmodels.Role, int, error) {
	if actorID == "" {
		return []rbacmodels.Role{}, 0, nil
	}
	actorRoles, err := UserRoles(ctx, catalog, links, actorID)
	if err != nil {
		return nil, 0, err
	}
	if len(actorRoles) == 0 {
		return []rbacmodels.Role{}, 0, nil
	}
	if actorType != "" && actorType != "user" {
		return []rbacmodels.Role{}, 0, nil
	}
	actorPolicies := []rbacmodels.Policy{}
	for _, role := range actorRoles {
		actorPolicies = append(actorPolicies, role.Policies...)
	}
	_, total, err := catalog.ListRoles(ctx, 10000, 0)
	if err != nil {
		return nil, 0, err
	}
	candidates := make([]rbacmodels.Role, 0, total)
	for offset := 0; offset < total; offset += 10000 {
		batch, _, err := catalog.ListRoles(ctx, 10000, offset)
		if err != nil {
			return nil, 0, err
		}
		candidates = append(candidates, batch...)
		if len(batch) == 0 {
			break
		}
	}
	idSet := toSet(ids)
	nameSet := toSet(names)
	filtered := make([]rbacmodels.Role, 0, len(candidates))
	query = strings.ToLower(strings.TrimSpace(query))
	for _, candidate := range candidates {
		if len(idSet) > 0 && !idSet[candidate.ID] {
			continue
		}
		if len(nameSet) > 0 && !nameSet[candidate.Name] {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(candidate.Name+" "+value(candidate.Description)), query) {
			continue
		}
		filtered = append(filtered, candidate)
	}
	if offset >= len(filtered) {
		return []rbacmodels.Role{}, 0, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(filtered) {
		end = len(filtered)
	}
	page := filtered[offset:end]
	assignable := make([]rbacmodels.Role, 0, len(page))
	for _, candidate := range page {
		policyMap, err := catalog.ListPoliciesForRoles(ctx, []string{candidate.ID})
		if err != nil {
			return nil, 0, err
		}
		allowed := true
		for _, policy := range policyMap[candidate.ID] {
			if policy.Resource == "" || policy.Operation == "" {
				continue
			}
			covered := false
			for _, grant := range actorPolicies {
				if (grant.Resource == policy.Resource || grant.Resource == "*") && (grant.Operation == policy.Operation || grant.Operation == "*") {
					covered = true
					break
				}
			}
			if !covered {
				allowed = false
				break
			}
		}
		if allowed {
			assignable = append(assignable, rbacmodels.Role{ID: candidate.ID, Name: candidate.Name, Description: candidate.Description})
		}
	}
	return assignable, len(assignable), nil
}

func ListAssignablePolicies(ctx context.Context, catalog UserRoleCatalog, links linkdomain.Repository, actorID, actorType, query string, ids, keys, resources, operations []string, limit, offset int) ([]rbacmodels.Policy, int, error) {
	if actorID == "" || (actorType != "" && actorType != "user") {
		return []rbacmodels.Policy{}, 0, nil
	}
	roles, err := UserRoles(ctx, catalog, links, actorID)
	if err != nil {
		return nil, 0, err
	}
	if len(roles) == 0 {
		return []rbacmodels.Policy{}, 0, nil
	}
	grants := []rbacmodels.Policy{}
	for _, role := range roles {
		grants = append(grants, role.Policies...)
	}
	allPolicies := []rbacmodels.Policy{}
	_, total, err := catalog.ListPolicies(ctx, 10000, 0)
	if err != nil {
		return nil, 0, err
	}
	for pageOffset := 0; pageOffset < total; pageOffset += 10000 {
		page, _, err := catalog.ListPolicies(ctx, 10000, pageOffset)
		if err != nil {
			return nil, 0, err
		}
		allPolicies = append(allPolicies, page...)
		if len(page) == 0 {
			break
		}
	}
	idsSet, keysSet := toSet(ids), toSet(keys)
	resourcesSet, operationsSet := toSet(resources), toSet(operations)
	query = strings.ToLower(strings.TrimSpace(query))
	assignable := make([]rbacmodels.Policy, 0)
	for _, policy := range allPolicies {
		if policy.Resource == "" || policy.Operation == "" {
			continue
		}
		if len(idsSet) > 0 && !idsSet[policy.ID] || len(keysSet) > 0 && !keysSet[policy.Key] || len(resourcesSet) > 0 && !resourcesSet[policy.Resource] || len(operationsSet) > 0 && !operationsSet[policy.Operation] {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(policy.Key+" "+policy.Resource+" "+policy.Operation+" "+value(policy.Description)), query) {
			continue
		}
		for _, grant := range grants {
			if (grant.Resource == policy.Resource || grant.Resource == "*") && (grant.Operation == policy.Operation || grant.Operation == "*") {
				assignable = append(assignable, rbacmodels.Policy{ID: policy.ID, Key: policy.Key, Resource: policy.Resource, Operation: policy.Operation, Description: policy.Description})
				break
			}
		}
	}
	count := len(assignable)
	if offset >= len(assignable) {
		return []rbacmodels.Policy{}, count, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(assignable) {
		end = len(assignable)
	}
	return assignable[offset:end], count, nil
}

func toSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, v := range values {
		set[v] = true
	}
	return set
}

func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
