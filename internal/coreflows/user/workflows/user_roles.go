package workflows

import (
	"context"
	"errors"

	linkdomain "github.com/example/medusa-go-clone/internal/modules/links/domain"
	rbacmodels "github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
	usermodels "github.com/example/medusa-go-clone/internal/modules/user/domain/models"
)

var ErrUserRolePermission = errors.New("You do not have permission to assign these roles")

const AssignUserRolesWorkflowID = "assign-user-roles"
const RemoveUserRolesWorkflowID = "remove-user-roles"
const ValidateUserRolePermissionsStepID = "validate-user-role-permissions"
const GetUsersForRoleStepID = "get-users-for-role"

type UserRoleCatalog interface {
	ListRoles(context.Context, int, int) ([]rbacmodels.Role, int, error)
	ListPolicies(context.Context, int, int) ([]rbacmodels.Policy, int, error)
	GetRole(context.Context, string) (rbacmodels.Role, error)
	GetPolicy(context.Context, string) (rbacmodels.Policy, error)
	ListPoliciesForRoles(context.Context, []string) (map[string][]rbacmodels.Policy, error)
}

func ValidateUserPermissions(ctx context.Context, catalog UserRoleCatalog, links linkdomain.Repository, actorID, actorType string, policyIDs []string) error {
	if len(policyIDs) == 0 {
		return nil
	}
	if actorID == "" || (actorType != "" && actorType != "user") {
		return ErrUserRolePermission
	}
	roles, err := UserRoles(ctx, catalog, links, actorID)
	if err != nil {
		return err
	}
	grants := []rbacmodels.Policy{}
	for _, role := range roles {
		grants = append(grants, role.Policies...)
	}
	for _, policyID := range policyIDs {
		policy, err := catalog.GetPolicy(ctx, policyID)
		if err != nil {
			return err
		}
		allowed := false
		for _, grant := range grants {
			if (grant.Resource == policy.Resource || grant.Resource == "*") && (grant.Operation == policy.Operation || grant.Operation == "*") {
				allowed = true
				break
			}
		}
		if !allowed {
			return ErrUserRolePermission
		}
	}
	return nil
}

func HasUserPermissions(ctx context.Context, catalog UserRoleCatalog, links linkdomain.Repository, actorID, actorType string, actions []rbacmodels.PermissionAction) (bool, error) {
	if len(actions) == 0 {
		return true, nil
	}
	if actorID == "" || (actorType != "" && actorType != "user") {
		return false, nil
	}
	roles, err := UserRoles(ctx, catalog, links, actorID)
	if err != nil {
		return false, err
	}
	if len(roles) == 0 {
		return false, nil
	}
	grants := []rbacmodels.Policy{}
	for _, role := range roles {
		grants = append(grants, role.Policies...)
	}
	for _, action := range actions {
		allowed := false
		for _, grant := range grants {
			if (grant.Resource == action.Resource || grant.Resource == "*") && (grant.Operation == action.Operation || grant.Operation == "*") {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, nil
		}
	}
	return true, nil
}

func AssignUserRoles(ctx context.Context, catalog UserRoleCatalog, links linkdomain.Repository, actorID, actorType, userID string, roleIDs []string) error {
	if err := validateAssignableRoles(ctx, catalog, links, actorID, actorType, roleIDs); err != nil {
		return err
	}
	def, ok := userRoleDefinition()
	if !ok {
		return errors.New("UserRBACRole link definition is missing")
	}
	rows := make([]map[string]any, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		rows = append(rows, map[string]any{"user_id": userID, "rbac_role_id": roleID})
	}
	_, err := links.Create(ctx, def, rows)
	return err
}

func RemoveUserRoles(ctx context.Context, catalog UserRoleCatalog, links linkdomain.Repository, actorID, actorType, userID string, roleIDs []string) error {
	if err := validateAssignableRoles(ctx, catalog, links, actorID, actorType, roleIDs); err != nil {
		return err
	}
	def, ok := userRoleDefinition()
	if !ok {
		return errors.New("UserRBACRole link definition is missing")
	}
	rows := make([]map[string]any, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		rows = append(rows, map[string]any{"user_id": userID, "rbac_role_id": roleID})
	}
	_, err := links.Dismiss(ctx, def, rows)
	return err
}

type RoleUserReader interface {
	ListUsers(context.Context, []string) ([]usermodels.User, error)
}

func UsersForRole(ctx context.Context, users RoleUserReader, links linkdomain.Repository, roleID string, userIDs []string, limit, offset int) ([]usermodels.User, int, error) {
	def, ok := userRoleDefinition()
	if !ok {
		return nil, 0, errors.New("UserRBACRole link definition is missing")
	}
	filters := map[string]any{"rbac_role_id": roleID}
	if len(userIDs) > 0 {
		filters["user_id"] = userIDs
	}
	linkRows, err := links.List(ctx, def, filters)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]string, 0, len(linkRows))
	for _, row := range linkRows {
		if id, ok := row.Fields["user_id"].(string); ok {
			ids = append(ids, id)
		}
	}
	count := len(ids)
	if offset >= count {
		return []usermodels.User{}, count, nil
	}
	end := offset + limit
	if limit <= 0 || end > count {
		end = count
	}
	ids = ids[offset:end]
	rows, err := users.ListUsers(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	byID := make(map[string]usermodels.User, len(rows))
	for _, user := range rows {
		byID[user.ID] = user
	}
	ordered := make([]usermodels.User, 0, len(ids))
	for _, id := range ids {
		if user, exists := byID[id]; exists {
			ordered = append(ordered, user)
		}
	}
	return ordered, count, nil
}

func AssignUsersToRole(ctx context.Context, catalog UserRoleCatalog, links linkdomain.Repository, actorID, actorType, roleID string, userIDs []string) error {
	if err := validateAssignableRoles(ctx, catalog, links, actorID, actorType, []string{roleID}); err != nil {
		return err
	}
	def, ok := userRoleDefinition()
	if !ok {
		return errors.New("UserRBACRole link definition is missing")
	}
	rows := make([]map[string]any, 0, len(userIDs))
	for _, userID := range userIDs {
		rows = append(rows, map[string]any{"user_id": userID, "rbac_role_id": roleID})
	}
	_, err := links.Create(ctx, def, rows)
	return err
}

func RemoveUsersFromRole(ctx context.Context, catalog UserRoleCatalog, links linkdomain.Repository, actorID, actorType, roleID string, userIDs []string) error {
	if err := validateAssignableRoles(ctx, catalog, links, actorID, actorType, []string{roleID}); err != nil {
		return err
	}
	def, ok := userRoleDefinition()
	if !ok {
		return errors.New("UserRBACRole link definition is missing")
	}
	rows := make([]map[string]any, 0, len(userIDs))
	for _, userID := range userIDs {
		rows = append(rows, map[string]any{"user_id": userID, "rbac_role_id": roleID})
	}
	_, err := links.Dismiss(ctx, def, rows)
	return err
}

func validateAssignableRoles(ctx context.Context, catalog UserRoleCatalog, links linkdomain.Repository, actorID, actorType string, roleIDs []string) error {
	unique := map[string]bool{}
	ids := make([]string, 0, len(roleIDs))
	for _, id := range roleIDs {
		if !unique[id] {
			unique[id] = true
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		if _, err := catalog.GetRole(ctx, id); err != nil {
			return err
		}
	}
	targetPolicies, err := catalog.ListPoliciesForRoles(ctx, ids)
	if err != nil {
		return err
	}
	wanted := []rbacmodels.Policy{}
	for _, policies := range targetPolicies {
		for _, p := range policies {
			wanted = append(wanted, p)
		}
	}
	if len(wanted) == 0 {
		return nil
	}
	if actorID == "" {
		return ErrUserRolePermission
	}
	if actorType != "" && actorType != "user" {
		return ErrUserRolePermission
	}
	def, ok := userRoleDefinition()
	if !ok {
		return errors.New("UserRBACRole link definition is missing")
	}
	actorLinks, err := links.List(ctx, def, map[string]any{"user_id": actorID})
	if err != nil {
		return err
	}
	roleIDsForActor := make([]string, 0, len(actorLinks))
	for _, row := range actorLinks {
		if value, ok := row.Fields["rbac_role_id"].(string); ok {
			roleIDsForActor = append(roleIDsForActor, value)
		}
	}
	actorPolicies, err := catalog.ListPoliciesForRoles(ctx, roleIDsForActor)
	if err != nil {
		return err
	}
	allowed := []rbacmodels.Policy{}
	for _, policies := range actorPolicies {
		for _, p := range policies {
			allowed = append(allowed, p)
		}
	}
	for _, action := range wanted {
		covered := false
		for _, grant := range allowed {
			if (grant.Resource == action.Resource || grant.Resource == "*") && (grant.Operation == action.Operation || grant.Operation == "*") {
				covered = true
				break
			}
		}
		if !covered {
			return ErrUserRolePermission
		}
	}
	return nil
}

func UserRoles(ctx context.Context, catalog UserRoleCatalog, links linkdomain.Repository, userID string) ([]rbacmodels.Role, error) {
	def, ok := userRoleDefinition()
	if !ok {
		return nil, errors.New("UserRBACRole link definition is missing")
	}
	rows, err := links.List(ctx, def, map[string]any{"user_id": userID})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if id, ok := row.Fields["rbac_role_id"].(string); ok {
			ids = append(ids, id)
		}
	}
	roles := make([]rbacmodels.Role, 0, len(ids))
	policies, err := catalog.ListPoliciesForRoles(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		role, e := catalog.GetRole(ctx, id)
		if e != nil {
			return nil, e
		}
		role.Policies = policies[id]
		roles = append(roles, role)
	}
	return roles, nil
}
