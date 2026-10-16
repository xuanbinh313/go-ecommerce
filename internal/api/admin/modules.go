package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	apikeyflows "github.com/example/medusa-go-clone/internal/coreflows/api-key/workflows"
	categoryflows "github.com/example/medusa-go-clone/internal/coreflows/product-category/workflows"
	rbacflows "github.com/example/medusa-go-clone/internal/coreflows/rbac/workflows"
	settingsflows "github.com/example/medusa-go-clone/internal/coreflows/settings/workflows"
	settingspropertyflows "github.com/example/medusa-go-clone/internal/coreflows/settings/workflows"
	storeflows "github.com/example/medusa-go-clone/internal/coreflows/store/workflows"
	userroleflows "github.com/example/medusa-go-clone/internal/coreflows/user/workflows"
	apikeymodels "github.com/example/medusa-go-clone/internal/modules/api-key/domain/models"
	currencymodels "github.com/example/medusa-go-clone/internal/modules/currency/domain/models"
	customermodels "github.com/example/medusa-go-clone/internal/modules/customer/domain/models"
	fulfillmentdomain "github.com/example/medusa-go-clone/internal/modules/fulfillment/domain"
	notificationmodels "github.com/example/medusa-go-clone/internal/modules/notification/domain/models"
	productdomain "github.com/example/medusa-go-clone/internal/modules/product/domain"
	productmodels "github.com/example/medusa-go-clone/internal/modules/product/domain/models"
	rbacmodels "github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
	regionmodels "github.com/example/medusa-go-clone/internal/modules/region/domain/models"
	salesdomain "github.com/example/medusa-go-clone/internal/modules/sales-channel/domain"
	salesmodels "github.com/example/medusa-go-clone/internal/modules/sales-channel/domain/models"
	settingsmodels "github.com/example/medusa-go-clone/internal/modules/settings/domain/models"
	storemodels "github.com/example/medusa-go-clone/internal/modules/store/domain/models"
	usermodels "github.com/example/medusa-go-clone/internal/modules/user/domain/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type CustomerReader interface {
	Get(context.Context, string) (customermodels.Customer, error)
	List(context.Context, int, int) ([]customermodels.Customer, error)
	Count(context.Context) (int64, error)
}
type CustomerWorkflow interface {
	Create(context.Context, customermodels.Input) (customermodels.Customer, error)
	Update(context.Context, string, customermodels.Input) (customermodels.Customer, error)
	DeleteCustomer(context.Context, string) error
	CreateAddress(context.Context, string, customermodels.AddressInput) (customermodels.Address, error)
	ListAddresses(context.Context, string, int, int) ([]customermodels.Address, int64, error)
	GetAddress(context.Context, string, string) (customermodels.Address, error)
	UpdateAddress(context.Context, string, string, customermodels.AddressUpdate) (customermodels.Customer, error)
	DeleteAddress(context.Context, string, string) error
	LinkCustomerGroups(context.Context, string, []string, []string, *string) error
	ListCustomerGroups(context.Context) ([]customermodels.CustomerGroup, error)
	GetCustomerGroup(context.Context, string) (customermodels.CustomerGroup, error)
	CreateCustomerGroup(context.Context, string, json.RawMessage, *string) (customermodels.CustomerGroup, error)
	UpdateCustomerGroup(context.Context, string, customermodels.CustomerGroupUpdate) (customermodels.CustomerGroup, error)
	DeleteCustomerGroup(context.Context, string) error
	LinkGroupCustomers(context.Context, string, []string, []string, *string) error
}
type ProductReader interface {
	GetProduct(context.Context, string) (productmodels.Product, error)
	ListProductOptions(context.Context, productmodels.ProductOptionFilter, int, int) ([]productmodels.ProductOption, int, error)
	GetProductOption(context.Context, string) (productmodels.ProductOption, error)
	ListProductVariants(context.Context, string, productmodels.ProductVariantFilter, int, int) ([]productmodels.ProductVariant, int, error)
	ListProducts(context.Context, []string) ([]productmodels.Product, error)
	ListAdminProducts(context.Context, []string, *bool, int, int) ([]productmodels.Product, int, error)
	ListProductTypes(context.Context, []string, int, int) ([]productmodels.ProductType, int, error)
	GetProductType(context.Context, string) (productmodels.ProductType, error)
	ListProductTags(context.Context, []string, int, int) ([]productmodels.ProductTag, int, error)
	GetProductTag(context.Context, string) (productmodels.ProductTag, error)
	ListProductCategories(context.Context, []string, int, int) ([]productmodels.ProductCategory, int, error)
	GetProductCategory(context.Context, string) (productmodels.ProductCategory, error)
}
type ProductCreator interface {
	CreateProducts(context.Context, productmodels.CreateProductsInput) ([]productmodels.Product, error)
	UpdateProduct(context.Context, string, map[string]any) error
	DeleteProduct(context.Context, string) error
}
type ProductCatalogWorkflow interface {
	CreateProductTypes(context.Context, []productmodels.ProductTypeInput) ([]productmodels.ProductType, error)
	UpdateProductType(context.Context, string, map[string]any) (productmodels.ProductType, error)
	DeleteProductTypes(context.Context, []string) error
	CreateProductTags(context.Context, []productmodels.ProductTagInput) ([]productmodels.ProductTag, error)
	UpdateProductTag(context.Context, string, map[string]any) (productmodels.ProductTag, error)
	DeleteProductTags(context.Context, []string) error
}
type ProductOptionCreator interface {
	CreateProductOptions(context.Context, []productmodels.CreateProductOptionInput) ([]productmodels.ProductOption, error)
	UpdateProductOption(context.Context, string, map[string]any) (productmodels.ProductOption, error)
	DeleteProductOptions(context.Context, []string) error
}
type RegionReader interface {
	Get(context.Context, string) (regionmodels.Region, error)
	List(context.Context) ([]regionmodels.Region, error)
}
type RegionWorkflow interface {
	CreateRegion(context.Context, regionmodels.RegionInput) (regionmodels.Region, error)
	UpdateRegion(context.Context, string, regionmodels.RegionInput) (regionmodels.Region, error)
	DeleteRegion(context.Context, string) error
}
type SalesChannelReader interface {
	Get(context.Context, string) (salesmodels.SalesChannel, error)
	List(context.Context) ([]salesmodels.SalesChannel, error)
}
type SalesChannelWorkflow interface {
	CreateSalesChannel(context.Context, salesmodels.Input) (salesmodels.SalesChannel, error)
	UpdateSalesChannel(context.Context, string, salesmodels.Input) (salesmodels.SalesChannel, error)
	DeleteSalesChannel(context.Context, string) error
	LinkProducts(context.Context, string, []string, []string) error
}
type StoreReader interface {
	Get(context.Context, string) (storemodels.Store, error)
	List(context.Context) ([]storemodels.Store, error)
	Update(context.Context, string, storemodels.StoreInput) (storemodels.Store, error)
}
type CurrencyReader interface {
	RetrieveCurrency(context.Context, string) (currencymodels.Currency, error)
	ListCurrencies(context.Context, []string) ([]currencymodels.Currency, error)
}
type UserReader interface {
	GetUser(context.Context, string) (usermodels.User, error)
	ListUsers(context.Context, []string) ([]usermodels.User, error)
}
type UserWorkflow interface {
	UpdateUser(context.Context, string, usermodels.UserInput) (usermodels.User, error)
}
type UserRolesWorkflow interface {
	ListUserRoles(context.Context, string) ([]rbacmodels.Role, error)
	ListAssignableRoles(context.Context, string, string, string, []string, []string, int, int) ([]rbacmodels.Role, int, error)
	ListAssignablePolicies(context.Context, string, string, string, []string, []string, []string, []string, int, int) ([]rbacmodels.Policy, int, error)
	ValidateUserPermissions(context.Context, string, string, []string) error
	HasPermissions(context.Context, string, string, []rbacmodels.PermissionAction) (bool, error)
	ListRoleUsers(context.Context, string, []string, int, int) ([]usermodels.User, int, error)
	AssignUsersToRole(context.Context, string, string, string, []string) error
	RemoveUsersFromRole(context.Context, string, string, string, []string) error
	AssignUserRoles(context.Context, string, string, string, []string) error
	RemoveUserRoles(context.Context, string, string, string, []string) error
}
type UserAccountWorkflow interface {
	RemoveUserAccount(context.Context, string) error
}
type UserAuthProviderReader interface {
	ListProviderNamesByActor(context.Context, string, string) ([]string, error)
}
type UserResetTokenWorkflow interface {
	GenerateUserResetToken(context.Context, string) (string, error)
}
type LayoutReader interface {
	ListLayoutConfigurations(context.Context, string, settingsmodels.LayoutConfigurationFilter, int, int) ([]settingsmodels.LayoutConfiguration, int, error)
	GetLayoutConfiguration(context.Context, string, string) (settingsmodels.LayoutConfigurations, error)
	SetLayoutConfiguration(context.Context, string, string, bool, []byte) error
	ClearLayoutConfiguration(context.Context, string, string) error
}
type PropertyLabelWorkflow interface {
	settingspropertyflows.PropertyLabelService
}
type NotificationReader interface {
	List(context.Context, notificationmodels.NotificationFilter, int, int, string) ([]notificationmodels.Notification, int, error)
	Get(context.Context, string) (notificationmodels.Notification, error)
}
type APIKeyReader interface {
	List(context.Context, apikeymodels.Filter) ([]apikeymodels.APIKey, int, error)
	Get(context.Context, string) (apikeymodels.APIKey, error)
	Create(context.Context, apikeymodels.CreateInput) (apikeymodels.Created, error)
	Update(context.Context, string, apikeymodels.UpdateInput) (apikeymodels.APIKey, error)
	Revoke(context.Context, string, apikeymodels.RevokeInput) (apikeymodels.APIKey, error)
	Delete(context.Context, string) error
	LinkSalesChannels(context.Context, string, []string, []string) error
}
type PolicyWorkflow interface {
	ListRoles(context.Context, int, int) ([]rbacmodels.Role, int, error)
	ListPolicies(context.Context, int, int) ([]rbacmodels.Policy, int, error)
	GetRole(context.Context, string) (rbacmodels.Role, error)
	GetPolicy(context.Context, string) (rbacmodels.Policy, error)
	ListRolePolicies(context.Context, string, int, int) ([]rbacmodels.RolePolicy, int, error)
	UpdateRole(context.Context, string, rbacmodels.RoleUpdate) (rbacmodels.Role, error)
	UpdatePolicy(context.Context, string, rbacmodels.PolicyUpdate) (rbacmodels.Policy, error)
	CreateRoles(context.Context, []rbacmodels.RoleInput) ([]rbacmodels.Role, error)
	CreateRoleParents(context.Context, []rbacmodels.RoleParentInput) ([]rbacmodels.RoleParent, error)
	CreateRolePolicies(context.Context, []rbacmodels.RolePolicyInput) ([]rbacmodels.RolePolicy, error)
	DeleteRoles(context.Context, []string) error
	DeleteRoleParents(context.Context, []string) error
	DeleteRolePolicies(context.Context, []string) error
	SoftDeleteRoles(context.Context, []string) error
	CreatePolicies(context.Context, []rbacmodels.PolicyInput) ([]rbacmodels.Policy, error)
	DeletePolicies(context.Context, []string) error
	SoftDeletePolicies(context.Context, []string) error
}

type ModuleHTTP struct {
	Customers             CustomerReader
	CustomerWorkflow      CustomerWorkflow
	Products              ProductReader
	ProductCreator        ProductCreator
	ProductCatalog        ProductCatalogWorkflow
	ProductOptionCreator  ProductOptionCreator
	ProductCategories     categoryflows.ProductCategoryService
	Regions               RegionReader
	RegionWorkflow        RegionWorkflow
	SalesChannels         SalesChannelReader
	SalesChannelWorkflow  SalesChannelWorkflow
	Stores                StoreReader
	Currencies            CurrencyReader
	Users                 UserReader
	UserWorkflow          UserWorkflow
	UserRolesWorkflow     UserRolesWorkflow
	UserAccountWorkflow   UserAccountWorkflow
	UserAuthProviders     UserAuthProviderReader
	UserResetToken        UserResetTokenWorkflow
	Policies              PolicyWorkflow
	Layouts               LayoutReader
	PropertyLabels        PropertyLabelWorkflow
	PropertyLabelsEnabled bool
	Notifications         NotificationReader
	APIKeys               APIKeyReader
	RBACEnabled           bool
}

func (h *ModuleHTTP) Register(router *gin.RouterGroup, authenticate gin.HandlerFunc) {
	router.GET("/store/products", h.listStoreProducts)
	router.GET("/store/products/:id", h.getStoreProduct)
	router.GET("/store/regions", h.listStoreRegions)
	router.GET("/store/regions/:id", h.getStoreRegion)
	admin := router.Group("/admin", authenticate)
	if h.RBACEnabled {
		router.POST("/admin/rbac/policies", authenticate, h.rbacPermissionMiddleware("rbac_policy", "create"), h.createPolicy)
		admin.GET("/rbac/policies", h.rbacPermissionMiddleware("rbac_policy", "read"), h.listPolicies)
		admin.GET("/rbac/policies/assignable", h.rbacPermissionMiddleware("rbac_policy", "read"), h.listAssignablePolicies)
		admin.GET("/rbac/policies/:id", h.rbacPermissionMiddleware("rbac_policy", "read"), h.getPolicy)
		admin.GET("/rbac/roles", h.rbacPermissionMiddleware("rbac_role", "read"), h.listRoles)
		admin.GET("/rbac/roles/assignable", h.rbacPermissionMiddleware("rbac_role", "read"), h.listAssignableRoles)
		admin.POST("/rbac/roles", h.createRole)
		admin.GET("/rbac/roles/:id", h.rbacPermissionMiddleware("rbac_role", "read"), h.getRole)
		admin.GET("/rbac/roles/:id/policies", h.rbacPermissionMiddleware("rbac_role", "read"), h.listRolePolicies)
		admin.POST("/rbac/roles/:id/policies", h.rbacPermissionMiddleware("rbac_role", "update"), h.createRolePolicies)
		admin.DELETE("/rbac/roles/:id/policies/:policy_id", h.rbacPermissionMiddleware("rbac_role", "update"), h.deleteRolePolicy)
		admin.GET("/rbac/roles/:id/users", h.rbacPermissionMiddleware("user", "read"), h.listRoleUsers)
		admin.POST("/rbac/roles/:id/users", h.rbacPermissionMiddleware("user", "update", "rbac_role", "update"), h.assignRoleUsers)
		admin.DELETE("/rbac/roles/:id/users", h.rbacPermissionMiddleware("user", "update", "rbac_role", "update"), h.removeRoleUsers)
		admin.POST("/rbac/roles/:id", h.rbacPermissionMiddleware("rbac_role", "update"), h.updateRole)
		admin.DELETE("/rbac/roles/:id", h.rbacPermissionMiddleware("rbac_role", "delete"), h.deleteRole)
		admin.POST("/rbac/policies/:id", h.rbacPermissionMiddleware("rbac_policy", "update"), h.updatePolicy)
		admin.DELETE("/rbac/policies/:id", h.rbacPermissionMiddleware("rbac_policy", "delete"), h.deletePolicy)
		admin.GET("/rbac/me/permissions", h.getCurrentPermissions)
	}
	admin.GET("/api-keys", h.listAPIKeys)
	admin.POST("/api-keys", h.createAPIKey)
	admin.GET("/api-keys/:id", h.getAPIKey)
	admin.POST("/api-keys/:id", h.updateAPIKey)
	admin.DELETE("/api-keys/:id", h.deleteAPIKey)
	admin.POST("/api-keys/:id/revoke", h.revokeAPIKey)
	admin.POST("/api-keys/:id/sales-channels", h.linkAPIKeySalesChannels)
	admin.GET("/customers", h.listCustomers)
	admin.POST("/customers", h.createCustomer)
	admin.GET("/customers/:id", h.getCustomer)
	admin.POST("/customers/:id", h.updateCustomer)
	admin.DELETE("/customers/:id", h.deleteCustomer)
	admin.GET("/customers/:id/addresses", h.listAdminCustomerAddresses)
	admin.POST("/customers/:id/addresses", h.createAdminCustomerAddress)
	admin.GET("/customers/:id/addresses/:address_id", h.getAdminCustomerAddress)
	admin.POST("/customers/:id/addresses/:address_id", h.updateAdminCustomerAddress)
	admin.DELETE("/customers/:id/addresses/:address_id", h.deleteAdminCustomerAddress)
	admin.POST("/customers/:id/customer-groups", h.linkCustomerGroups)
	admin.GET("/customer-groups", h.listCustomerGroups)
	admin.POST("/customer-groups", h.createCustomerGroup)
	admin.GET("/customer-groups/:id", h.getCustomerGroup)
	admin.POST("/customer-groups/:id", h.updateCustomerGroup)
	admin.DELETE("/customer-groups/:id", h.deleteCustomerGroup)
	admin.POST("/customer-groups/:id/customers", h.linkGroupCustomers)
	admin.GET("/products", h.listProducts)
	admin.POST("/products", h.createProduct)
	admin.GET("/products/:id", h.getProduct)
	admin.GET("/products/:id/variants", h.listProductVariants)
	admin.GET("/product-variants", h.listAllProductVariants)
	admin.GET("/products/:id/options", h.listOptionsForProduct)
	admin.POST("/products/:id", h.updateProduct)
	admin.DELETE("/products/:id", h.deleteProduct)
	admin.GET("/product-types", h.listProductTypes)
	admin.GET("/product-types/:id", h.getProductType)
	admin.POST("/product-types", h.createProductType)
	admin.POST("/product-types/:id", h.updateProductType)
	admin.DELETE("/product-types/:id", h.deleteProductType)
	admin.GET("/product-tags", h.listProductTags)
	admin.GET("/product-tags/:id", h.getProductTag)
	admin.POST("/product-tags", h.createProductTag)
	admin.POST("/product-tags/:id", h.updateProductTag)
	admin.DELETE("/product-tags/:id", h.deleteProductTag)
	admin.GET("/product-options", h.listProductOptions)
	admin.POST("/product-options", h.createProductOption)
	admin.GET("/product-options/:id", h.getProductOption)
	admin.POST("/product-options/:id", h.updateProductOption)
	admin.DELETE("/product-options/:id", h.deleteProductOption)
	admin.GET("/product-categories", h.listProductCategories)
	admin.POST("/product-categories", h.createProductCategory)
	admin.GET("/product-categories/:id", h.getProductCategory)
	admin.POST("/product-categories/:id", h.updateProductCategory)
	admin.DELETE("/product-categories/:id", h.deleteProductCategory)
	admin.POST("/product-categories/:id/products", h.linkProductCategoryProducts)
	admin.GET("/layouts/configurations", h.listLayoutConfigurations)
	admin.GET("/layouts/:zone/configuration", h.getLayoutConfiguration)
	admin.POST("/layouts/:zone/configuration", h.setLayoutConfiguration)
	admin.DELETE("/layouts/:zone/configuration", h.clearLayoutConfiguration)
	admin.GET("/property-labels", h.listPropertyLabels)
	admin.POST("/property-labels", h.createPropertyLabel)
	admin.GET("/property-labels/:id", h.getPropertyLabel)
	admin.POST("/property-labels/:id", h.updatePropertyLabel)
	admin.DELETE("/property-labels/:id", h.deletePropertyLabel)
	admin.POST("/property-labels/batch", h.batchPropertyLabels)
	admin.GET("/notifications", h.listNotifications)
	admin.GET("/notifications/:id", h.getNotification)
	admin.GET("/regions", h.listRegions)
	admin.POST("/regions", h.createRegion)
	admin.GET("/regions/:id", h.getRegion)
	admin.GET("/regions/:id", h.getRegion)
	admin.POST("/regions/:id", h.updateRegion)
	admin.DELETE("/regions/:id", h.deleteRegion)
	admin.GET("/sales-channels", h.listSalesChannels)
	admin.POST("/sales-channels", h.createSalesChannel)
	admin.GET("/sales-channels/:id", h.getSalesChannel)
	admin.POST("/sales-channels/:id/products", permissionMiddleware(h.UserRolesWorkflow, h.RBACEnabled, "sales_channel", "update"), h.linkSalesChannelProducts)
	admin.POST("/sales-channels/:id", h.updateSalesChannel)
	admin.DELETE("/sales-channels/:id", h.deleteSalesChannel)
	admin.GET("/stores", h.listStores)
	admin.GET("/stores/:id", h.getStore)
	admin.POST("/stores/:id", h.updateStore)
	admin.GET("/currencies", h.listCurrencies)
	admin.GET("/currencies/:code", h.getCurrency)
	admin.GET("/users", h.listUsers)
	admin.GET("/users/me", h.getCurrentUser)
	admin.GET("/users/:id", h.getUser)
	admin.POST("/users/:id", h.updateUser)
	admin.DELETE("/users/:id", h.deleteUser)
	admin.GET("/users/:id/auth-providers", h.listUserAuthProviders)
	admin.POST("/users/:id/reset-password", h.resetUserPassword)
	if h.RBACEnabled {
		admin.GET("/users/:id/roles", h.rbacPermissionMiddleware("user", "read", "rbac_role", "read"), h.listUserRoles)
		admin.POST("/users/:id/roles", h.rbacPermissionMiddleware("user", "update", "rbac_role", "update"), h.assignUserRoles)
		admin.DELETE("/users/:id/roles", h.rbacPermissionMiddleware("user", "update", "rbac_role", "update"), h.removeUserRoles)
		admin.DELETE("/users/:id/roles/:role_id", h.rbacPermissionMiddleware("user", "update", "rbac_role", "update"), h.removeSingleUserRole)
	}
}

func (h *ModuleHTTP) rbacPermissionMiddleware(pairs ...string) gin.HandlerFunc {
	return permissionMiddleware(h.UserRolesWorkflow, true, pairs...)
}

func permissionMiddleware(workflow UserRolesWorkflow, enabled bool, pairs ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}
		actions := make([]rbacmodels.PermissionAction, 0, len(pairs)/2)
		for i := 0; i+1 < len(pairs); i += 2 {
			actions = append(actions, rbacmodels.PermissionAction{Resource: pairs[i], Operation: pairs[i+1]})
		}
		allowed, err := workflow.HasPermissions(c, contextValue(c.MustGet("actor_id")), contextValue(c.MustGet("actor_type")), actions)
		if err != nil {
			writeModuleError(c, err)
			c.Abort()
			return
		}
		if !allowed {
			writeModuleError(c, userroleflows.ErrUserRolePermission)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *ModuleHTTP) getCurrentPermissions(c *gin.Context) {
	actorID := contextValue(c.MustGet("actor_id"))
	actorType := contextValue(c.MustGet("actor_type"))
	if actorID == "" || actorType == "" {
		c.JSON(http.StatusOK, gin.H{"permissions": []string{}})
		return
	}
	roles, err := h.UserRolesWorkflow.ListUserRoles(c, actorID)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	policies, _, err := h.Policies.ListPolicies(c, 10000, 0)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	grants := make([]rbacmodels.Policy, 0)
	for _, role := range roles {
		grants = append(grants, role.Policies...)
	}
	granted := map[string]bool{}
	for _, policy := range policies {
		if policy.Resource == "*" || policy.Operation == "*" {
			continue
		}
		for _, grant := range grants {
			if (grant.Resource == policy.Resource || grant.Resource == "*") && (grant.Operation == policy.Operation || grant.Operation == "*") {
				granted[policy.Resource+":"+policy.Operation] = true
				break
			}
		}
	}
	permissions := make([]string, 0, len(granted))
	for permission := range granted {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

func (h *ModuleHTTP) listAssignableRoles(c *gin.Context) {
	actorID, actorType := contextValue(c.MustGet("actor_id")), contextValue(c.MustGet("actor_type"))
	limit, offset := paginationFor(c, 20, 1000)
	names := c.QueryArray("name[]")
	if len(names) == 0 {
		names = c.QueryArray("name")
	}
	rows, count, err := h.UserRolesWorkflow.ListAssignableRoles(c, actorID, actorType, c.Query("q"), queryIDs(c), names, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"roles": rows, "count": count})
}

func (h *ModuleHTTP) listAssignablePolicies(c *gin.Context) {
	actorID, actorType := contextValue(c.MustGet("actor_id")), contextValue(c.MustGet("actor_type"))
	limit, offset := paginationFor(c, 20, 1000)
	keys := c.QueryArray("key[]")
	if len(keys) == 0 {
		keys = c.QueryArray("key")
	}
	resources := c.QueryArray("resource[]")
	if len(resources) == 0 {
		resources = c.QueryArray("resource")
	}
	operations := c.QueryArray("operation[]")
	if len(operations) == 0 {
		operations = c.QueryArray("operation")
	}
	rows, count, err := h.UserRolesWorkflow.ListAssignablePolicies(c, actorID, actorType, c.Query("q"), queryIDs(c), keys, resources, operations, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": rows, "count": count})
}

func (h *ModuleHTTP) listPolicies(c *gin.Context) {
	limit, offset := pagination(c)
	rows, count, err := h.Policies.ListPolicies(c, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("policies", rows, count, offset, limit))
}
func (h *ModuleHTTP) getPolicy(c *gin.Context) {
	row, err := h.Policies.GetPolicy(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": row})
}
func (h *ModuleHTTP) listRoles(c *gin.Context) {
	limit, offset := pagination(c)
	rows, count, err := h.Policies.ListRoles(c, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("roles", rows, count, offset, limit))
}
func (h *ModuleHTTP) getRole(c *gin.Context) {
	row, err := h.Policies.GetRole(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"role": row})
}
func (h *ModuleHTTP) listRolePolicies(c *gin.Context) {
	limit, offset := paginationFor(c, 50, 1000)
	rows, count, err := h.Policies.ListRolePolicies(c, c.Param("id"), limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": rows, "count": count, "offset": offset, "limit": limit})
}

func (h *ModuleHTTP) createRolePolicies(c *gin.Context) {
	var body struct {
		Policies []string `json:"policies"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Policies) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "policies must be a non-empty array"})
		return
	}
	actorID, actorType := contextValue(c.MustGet("actor_id")), contextValue(c.MustGet("actor_type"))
	inputs := make([]rbacmodels.RolePolicyInput, len(body.Policies))
	for i, policyID := range body.Policies {
		inputs[i] = rbacmodels.RolePolicyInput{RoleID: c.Param("id"), PolicyID: policyID}
	}
	created, err := rbacflows.CreateRbacRolePolicies(c, h.Policies, h.UserRolesWorkflow, actorID, actorType, inputs)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	rows, _, err := h.Policies.ListRolePolicies(c, c.Param("id"), 10000, 0)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	wanted := map[string]bool{}
	for _, row := range created {
		wanted[row.ID] = true
	}
	result := make([]rbacmodels.RolePolicy, 0, len(created))
	for _, row := range rows {
		if wanted[row.ID] {
			result = append(result, row)
		}
	}
	c.JSON(http.StatusOK, gin.H{"policies": result})
}

func (h *ModuleHTTP) deleteRolePolicy(c *gin.Context) {
	roleID, policyID := c.Param("id"), c.Param("policy_id")
	rows, _, err := h.Policies.ListRolePolicies(c, roleID, 10000, 0)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	var linkID any
	for _, row := range rows {
		if row.PolicyID == policyID {
			linkID = row.ID
			if err := rbacflows.DeleteRbacRolePolicies(c, h.Policies, []string{row.ID}); err != nil {
				writeModuleError(c, err)
				return
			}
			break
		}
	}
	response := gin.H{"object": "rbac_role_policy", "deleted": true}
	if linkID != nil {
		response["id"] = linkID
	}
	c.JSON(http.StatusOK, response)
}

func (h *ModuleHTTP) listRoleUsers(c *gin.Context) {
	ids := c.QueryArray("user_id[]")
	if len(ids) == 0 {
		ids = c.QueryArray("user_id")
	}
	limit, offset := paginationFor(c, 50, 1000)
	users, count, err := h.UserRolesWorkflow.ListRoleUsers(c, c.Param("id"), ids, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users, "count": count, "offset": offset, "limit": limit})
}

func (h *ModuleHTTP) assignRoleUsers(c *gin.Context) {
	var input struct {
		Users []string `json:"users"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Users) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "users must be a non-empty array"})
		return
	}
	roleID := c.Param("id")
	if _, err := h.Policies.GetRole(c, roleID); err != nil {
		writeModuleError(c, err)
		return
	}
	actorID, actorType := contextValue(c.MustGet("actor_id")), contextValue(c.MustGet("actor_type"))
	if err := h.UserRolesWorkflow.AssignUsersToRole(c, actorID, actorType, roleID, input.Users); err != nil {
		writeModuleError(c, err)
		return
	}
	users, _, err := h.UserRolesWorkflow.ListRoleUsers(c, roleID, nil, 100000, 0)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *ModuleHTTP) removeRoleUsers(c *gin.Context) {
	var input struct {
		Users []string `json:"users"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Users) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "users must be a non-empty array"})
		return
	}
	roleID := c.Param("id")
	if _, err := h.Policies.GetRole(c, roleID); err != nil {
		writeModuleError(c, err)
		return
	}
	actorID, actorType := contextValue(c.MustGet("actor_id")), contextValue(c.MustGet("actor_type"))
	if err := h.UserRolesWorkflow.RemoveUsersFromRole(c, actorID, actorType, roleID, input.Users); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ids": input.Users, "object": "role_user", "deleted": true})
}

func (h *ModuleHTTP) createRole(c *gin.Context) {
	var input rbacmodels.RoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "name is required"})
		return
	}
	actor, _ := c.Get("actor_type")
	actorType, _ := actor.(string)
	actorIDValue, _ := c.Get("actor_id")
	actorID, _ := actorIDValue.(string)
	created, err := rbacflows.CreateRbacRoles(c, h.Policies, nil, actorID, actorType, []rbacflows.CreateRoleInput{{Name: input.Name, Description: input.Description, Metadata: input.Metadata}})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"role": created[0]})
}

func (h *ModuleHTTP) updateRole(c *gin.Context) {
	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	for key := range raw {
		if key != "name" && key != "description" && key != "metadata" {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "unsupported or invalid field: " + key})
			return
		}
	}
	var input rbacmodels.RoleUpdate
	if value, ok := raw["name"]; ok {
		var name string
		if err := json.Unmarshal(value, &name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "name must be a string"})
			return
		}
		input.Name = &name
	}
	if value, ok := raw["description"]; ok {
		input.DescriptionSet = true
		if string(value) != "null" {
			var description string
			if err := json.Unmarshal(value, &description); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "description must be a string or null"})
				return
			}
			input.Description = &description
		}
	}
	if value, ok := raw["metadata"]; ok {
		if !isJSONRecord(value) {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "metadata must be an object or null"})
			return
		}
		input.MetadataSet = true
		input.Metadata = value
	}
	row, err := rbacflows.UpdateRbacRole(c, h.Policies, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"role": row})
}

func (h *ModuleHTTP) deleteRole(c *gin.Context) {
	id := c.Param("id")
	if err := rbacflows.DeleteRbacRoles(c, h.Policies, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "rbac_role", "deleted": true})
}

func (h *ModuleHTTP) updatePolicy(c *gin.Context) {
	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	for key := range raw {
		if key != "key" && key != "resource" && key != "operation" && key != "name" && key != "description" && key != "metadata" {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "invalid field: " + key})
			return
		}
	}
	var input rbacmodels.PolicyUpdate
	for key, target := range map[string]**string{"key": &input.Key, "resource": &input.Resource, "operation": &input.Operation} {
		if value, ok := raw[key]; ok {
			var text string
			if err := json.Unmarshal(value, &text); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": key + " must be a string"})
				return
			}
			*target = &text
		}
	}
	for key, field := range map[string]struct {
		target **string
		set    *bool
	}{"name": {&input.Name, &input.NameSet}, "description": {&input.Description, &input.DescriptionSet}} {
		if value, ok := raw[key]; ok {
			*field.set = true
			if string(value) != "null" {
				var text string
				if err := json.Unmarshal(value, &text); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": key + " must be a string or null"})
					return
				}
				*field.target = &text
			}
		}
	}
	if value, ok := raw["metadata"]; ok {
		if !isJSONRecord(value) {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "metadata must be an object or null"})
			return
		}
		input.MetadataSet = true
		input.Metadata = value
	}
	row, err := rbacflows.UpdateRbacPolicy(c, h.Policies, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": row})
}

func isJSONRecord(raw json.RawMessage) bool {
	if strings.TrimSpace(string(raw)) == "null" {
		return true
	}
	var record map[string]json.RawMessage
	return json.Unmarshal(raw, &record) == nil && record != nil
}

func (h *ModuleHTTP) deletePolicy(c *gin.Context) {
	id := c.Param("id")
	if err := rbacflows.DeleteRbacPolicies(c, h.Policies, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "rbac_policy", "deleted": true})
}

func (h *ModuleHTTP) listAPIKeys(c *gin.Context) {
	limit, offset := pagination(c)
	filter := apikeymodels.Filter{IDs: queryValues(c, "id"), Titles: queryValues(c, "title"), Types: queryValues(c, "type"), Query: c.Query("q"), Limit: limit, Offset: offset}
	rows, count, err := h.APIKeys.List(c, filter)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectAPIKey(row, c.Query("fields")))
	}
	c.JSON(http.StatusOK, listResponse("api_keys", items, count, offset, limit))
}
func (h *ModuleHTTP) createAPIKey(c *gin.Context) {
	var input apikeymodels.CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	actor, _ := c.Get("actor_id")
	input.CreatedBy, _ = actor.(string)
	created, err := apikeyflows.CreateApiKeys(c, h.APIKeys, input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	created.APIKey.Token = created.RawToken
	c.JSON(http.StatusOK, gin.H{"api_key": projectAPIKey(created.APIKey, c.Query("fields"))})
}
func (h *ModuleHTTP) getAPIKey(c *gin.Context) {
	row, err := h.APIKeys.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"api_key": projectAPIKey(row, c.Query("fields"))})
}
func (h *ModuleHTTP) updateAPIKey(c *gin.Context) {
	var input apikeymodels.UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := apikeyflows.UpdateApiKeys(c, h.APIKeys, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"api_key": projectAPIKey(row, c.Query("fields"))})
}
func (h *ModuleHTTP) deleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	if err := apikeyflows.DeleteApiKeys(c, h.APIKeys, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "api_key", "deleted": true})
}
func (h *ModuleHTTP) revokeAPIKey(c *gin.Context) {
	var input apikeymodels.RevokeInput
	if err := c.ShouldBindJSON(&input); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	actor, _ := c.Get("actor_id")
	input.RevokedBy, _ = actor.(string)
	row, err := apikeyflows.RevokeApiKeys(c, h.APIKeys, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"api_key": projectAPIKey(row, c.Query("fields"))})
}
func (h *ModuleHTTP) linkAPIKeySalesChannels(c *gin.Context) {
	var input struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := apikeyflows.LinkSalesChannelsToApiKey(c, h.APIKeys, c.Param("id"), input.Add, input.Remove)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"api_key": projectAPIKey(row, c.Query("fields"))})
}
func projectAPIKey(row apikeymodels.APIKey, requested string) map[string]any {
	data, _ := json.Marshal(row)
	all := map[string]any{}
	_ = json.Unmarshal(data, &all)
	defaults := []string{"id", "title", "token", "redacted", "type", "last_used_at", "updated_at", "created_at", "created_by", "revoked_at", "revoked_by", "sales_channels"}
	fields := strings.Split(requested, ",")
	if requested == "" {
		fields = defaults
	}
	out := map[string]any{}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "*" {
			return all
		}
	}
	useDefaults := false
	for _, field := range fields {
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			useDefaults = true
		}
	}
	if useDefaults {
		for _, field := range defaults {
			out[field] = all[field]
		}
	}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if strings.HasPrefix(field, "-") {
			delete(out, strings.TrimPrefix(field, "-"))
			continue
		}
		if strings.HasPrefix(field, "sales_channels.") {
			out["sales_channels"] = all["sales_channels"]
			continue
		}
		field = strings.TrimPrefix(field, "+")
		if value, ok := all[field]; ok {
			out[field] = value
		}
	}
	return out
}

func (h *ModuleHTTP) createPolicy(c *gin.Context) {
	var input rbacmodels.PolicyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	created, err := rbacflows.CreateRbacPolicies(c, h.Policies, []rbacmodels.PolicyInput{input})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": created[0]})
}

func (h *ModuleHTTP) listStoreProducts(c *gin.Context) {
	rows, err := h.Products.ListProducts(c, queryIDs(c))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	published := rows[:0]
	for _, product := range rows {
		if product.Status == "published" {
			published = append(published, product)
		}
	}
	writePaged(c, "products", published)
}

func (h *ModuleHTTP) getStoreProduct(c *gin.Context) {
	row, err := h.Products.GetProduct(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if row.Status != "published" {
		c.JSON(http.StatusNotFound, gin.H{"type": "not_found", "message": "Product not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": row})
}

func (h *ModuleHTTP) listStoreRegions(c *gin.Context) {
	rows, err := h.Regions.List(c)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	writePaged(c, "regions", rows)
}

func (h *ModuleHTTP) getStoreRegion(c *gin.Context) {
	row, err := h.Regions.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"region": row})
}

func (h *ModuleHTTP) listCustomers(c *gin.Context) {
	limit, offset := pagination(c)
	total, err := h.Customers.Count(c)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	rows, err := h.Customers.List(c, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("customers", rows, int(total), offset, limit))
}

func (h *ModuleHTTP) getCustomer(c *gin.Context) {
	row, err := h.Customers.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer": row})
}

func (h *ModuleHTTP) createCustomer(c *gin.Context) {
	var input customermodels.Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if actorID, exists := c.Get("actor_id"); exists {
		createdBy := contextValue(actorID)
		input.CreatedBy = &createdBy
	}
	row, err := h.CustomerWorkflow.Create(c, input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer": row})
}

func (h *ModuleHTTP) updateCustomer(c *gin.Context) {
	var input customermodels.Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := h.CustomerWorkflow.Update(c, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer": row})
}

func (h *ModuleHTTP) deleteCustomer(c *gin.Context) {
	id := c.Param("id")
	if err := h.CustomerWorkflow.DeleteCustomer(c, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "customer", "deleted": true})
}

func (h *ModuleHTTP) listAdminCustomerAddresses(c *gin.Context) {
	limit, offset := paginationFor(c, 50, 1000)
	rows, total, err := h.CustomerWorkflow.ListAddresses(c, c.Param("id"), limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("addresses", rows, int(total), offset, limit))
}

func (h *ModuleHTTP) createAdminCustomerAddress(c *gin.Context) {
	var input customermodels.AddressInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if _, err := h.CustomerWorkflow.CreateAddress(c, c.Param("id"), input); err != nil {
		writeModuleError(c, err)
		return
	}
	row, err := h.Customers.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer": row})
}

func (h *ModuleHTTP) getAdminCustomerAddress(c *gin.Context) {
	row, err := h.CustomerWorkflow.GetAddress(c, c.Param("id"), c.Param("address_id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": row})
}

func (h *ModuleHTTP) updateAdminCustomerAddress(c *gin.Context) {
	var input customermodels.AddressUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := h.CustomerWorkflow.UpdateAddress(c, c.Param("id"), c.Param("address_id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer": row})
}

func (h *ModuleHTTP) deleteAdminCustomerAddress(c *gin.Context) {
	id := c.Param("address_id")
	if err := h.CustomerWorkflow.DeleteAddress(c, c.Param("id"), id); err != nil {
		writeModuleError(c, err)
		return
	}
	parent, err := h.Customers.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "customer_address", "deleted": true, "parent": parent})
}

func (h *ModuleHTTP) linkCustomerGroups(c *gin.Context) {
	var input struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	actorID, _ := c.Get("actor_id")
	createdBy := contextValue(actorID)
	if err := h.CustomerWorkflow.LinkCustomerGroups(c, c.Param("id"), input.Add, input.Remove, &createdBy); err != nil {
		writeModuleError(c, err)
		return
	}
	row, err := h.Customers.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer": row})
}

func (h *ModuleHTTP) listCustomerGroups(c *gin.Context) {
	rows, err := h.CustomerWorkflow.ListCustomerGroups(c)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	limit, offset := paginationFor(c, 50, 1000)
	total := len(rows)
	rows = page(rows, offset, limit)
	c.JSON(http.StatusOK, listResponse("customer_groups", rows, total, offset, limit))
}

func (h *ModuleHTTP) createCustomerGroup(c *gin.Context) {
	var input struct {
		Name     string          `json:"name"`
		Metadata json.RawMessage `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	actorID, _ := c.Get("actor_id")
	createdBy := contextValue(actorID)
	row, err := h.CustomerWorkflow.CreateCustomerGroup(c, input.Name, input.Metadata, &createdBy)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer_group": row})
}

func (h *ModuleHTTP) getCustomerGroup(c *gin.Context) {
	row, err := h.CustomerWorkflow.GetCustomerGroup(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer_group": row})
}

func (h *ModuleHTTP) updateCustomerGroup(c *gin.Context) {
	var input customermodels.CustomerGroupUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := h.CustomerWorkflow.UpdateCustomerGroup(c, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer_group": row})
}

func (h *ModuleHTTP) deleteCustomerGroup(c *gin.Context) {
	id := c.Param("id")
	if err := h.CustomerWorkflow.DeleteCustomerGroup(c, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "customer_group", "deleted": true})
}

func (h *ModuleHTTP) linkGroupCustomers(c *gin.Context) {
	var input struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	actorID, _ := c.Get("actor_id")
	createdBy := contextValue(actorID)
	if err := h.CustomerWorkflow.LinkGroupCustomers(c, c.Param("id"), input.Add, input.Remove, &createdBy); err != nil {
		writeModuleError(c, err)
		return
	}
	row, err := h.CustomerWorkflow.GetCustomerGroup(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer_group": row})
}

func (h *ModuleHTTP) listProducts(c *gin.Context) {
	var isGiftcard *bool
	if raw := c.Query("is_giftcard"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "is_giftcard must be a boolean"})
			return
		}
		isGiftcard = &value
	}
	limit, offset := paginationFor(c, 50, 1000)
	rows, total, err := h.Products.ListAdminProducts(c, queryIDs(c), isGiftcard, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("products", rows, total, offset, limit))
}

func (h *ModuleHTTP) createProduct(c *gin.Context) {
	if h.ProductCreator == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product creation workflow is not configured"})
		return
	}
	var request struct {
		productmodels.ProductInput
		AdditionalData map[string]any `json:"additional_data"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if strings.TrimSpace(request.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "title is required"})
		return
	}
	// Additional data is accepted by the source contract. The workflow currently
	// passes it through without the source subscriber integrations.
	created, err := h.ProductCreator.CreateProducts(c, productmodels.CreateProductsInput{Products: []productmodels.ProductInput{request.ProductInput}, AdditionalData: request.AdditionalData})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(created) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "unexpected_state", "message": "create-products returned no product"})
		return
	}
	product, err := h.Products.GetProduct(c, created[0].ID)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": product})
}

func (h *ModuleHTTP) listProductVariants(c *gin.Context) {
	h.listVariants(c, c.Param("id"))
}

func (h *ModuleHTTP) listAllProductVariants(c *gin.Context) {
	h.listVariants(c, "")
}

func (h *ModuleHTTP) listVariants(c *gin.Context, productID string) {
	filter := productmodels.ProductVariantFilter{
		IDs:      queryIDs(c),
		SKUs:     queryArray(c, "sku"),
		EANs:     queryArray(c, "ean"),
		UPCs:     queryArray(c, "upc"),
		Barcodes: queryArray(c, "barcode"),
		Query:    c.Query("q"),
	}
	var err error
	if filter.ManageInventory, err = optionalBoolQuery(c, "manage_inventory"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if filter.AllowBackorder, err = optionalBoolQuery(c, "allow_backorder"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	limit, offset := paginationFor(c, 50, 1000)
	rows, count, err := h.Products.ListProductVariants(c, productID, filter, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("variants", rows, count, offset, limit))
}

func (h *ModuleHTTP) listOptionsForProduct(c *gin.Context) {
	product, err := h.Products.GetProduct(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	ids := queryIDs(c)
	title := c.Query("title")
	query := strings.ToLower(c.Query("q"))
	options := make([]productmodels.ProductOption, 0, len(product.Options))
	for _, option := range product.Options {
		if len(ids) > 0 && !contains(ids, option.ID) {
			continue
		}
		if title != "" && option.Title != title {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(option.Title), query) {
			continue
		}
		options = append(options, option)
	}
	limit, offset := paginationFor(c, 50, 1000)
	count := len(options)
	options = page(options, offset, limit)
	c.JSON(http.StatusOK, listResponse("product_options", options, count, offset, limit))
}

func (h *ModuleHTTP) listProductOptions(c *gin.Context) {
	filter := productmodels.ProductOptionFilter{IDs: queryArray(c, "id"), Titles: queryArray(c, "title"), Query: c.Query("q")}
	var err error
	if filter.IsExclusive, err = optionalBoolQuery(c, "is_exclusive"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	limit, offset := paginationFor(c, 20, 1000)
	items, count, err := h.Products.ListProductOptions(c, filter, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("product_options", items, count, offset, limit))
}

func (h *ModuleHTTP) getProductOption(c *gin.Context) {
	item, err := h.Products.GetProductOption(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_option": item})
}

func (h *ModuleHTTP) createProductOption(c *gin.Context) {
	var input productmodels.CreateProductOptionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if h.ProductOptionCreator == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product option workflow is not configured"})
		return
	}
	items, err := h.ProductOptionCreator.CreateProductOptions(c, []productmodels.CreateProductOptionInput{input})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(items) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "unexpected_state", "message": "create-product-options returned no result"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_option": items[0]})
}

func (h *ModuleHTTP) updateProductOption(c *gin.Context) {
	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if h.ProductOptionCreator == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product option workflow is not configured"})
		return
	}
	item, err := h.ProductOptionCreator.UpdateProductOption(c, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_option": item})
}

func (h *ModuleHTTP) deleteProductOption(c *gin.Context) {
	id := c.Param("id")
	if h.ProductOptionCreator == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product option workflow is not configured"})
		return
	}
	if err := h.ProductOptionCreator.DeleteProductOptions(c, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "product_option", "deleted": true})
}

func (h *ModuleHTTP) listProductCategories(c *gin.Context) {
	limit, offset := paginationWithMax(c, 1000)
	rows, count, err := h.Products.ListProductCategories(c, queryIDs(c), limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("product_categories", rows, count, offset, limit))
}
func (h *ModuleHTTP) getProductCategory(c *gin.Context) {
	item, err := h.Products.GetProductCategory(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_category": item})
}
func (h *ModuleHTTP) createProductCategory(c *gin.Context) {
	var input productmodels.ProductCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if h.ProductCategories == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product category workflow is not configured"})
		return
	}
	items, err := categoryflows.CreateProductCategories(c, h.ProductCategories, []productmodels.ProductCategoryInput{input})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(items) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "unexpected_state", "message": "create-product-categories returned no result"})
		return
	}
	item, err := h.Products.GetProductCategory(c, items[0].ID)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_category": item})
}
func (h *ModuleHTTP) updateProductCategory(c *gin.Context) {
	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	item, err := categoryflows.UpdateProductCategory(c, h.ProductCategories, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_category": item})
}
func (h *ModuleHTTP) deleteProductCategory(c *gin.Context) {
	id := c.Param("id")
	if err := categoryflows.DeleteProductCategories(c, h.ProductCategories, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "product_category", "deleted": true})
}
func (h *ModuleHTTP) linkProductCategoryProducts(c *gin.Context) {
	var input struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	id := c.Param("id")
	if err := categoryflows.BatchLinkProductsToCategory(c, h.ProductCategories, id, input.Add, input.Remove); err != nil {
		writeModuleError(c, err)
		return
	}
	item, err := h.Products.GetProductCategory(c, id)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_category": item})
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func queryArray(c *gin.Context, field string) []string {
	values := c.QueryArray(field + "[]")
	if len(values) == 0 {
		values = c.QueryArray(field)
	}
	return values
}

func optionalBoolQuery(c *gin.Context, field string) (*bool, error) {
	raw := c.Query(field)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be a boolean", field)
	}
	return &value, nil
}

func (h *ModuleHTTP) getProduct(c *gin.Context) {
	row, err := h.Products.GetProduct(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": row})
}

func (h *ModuleHTTP) deleteProduct(c *gin.Context) {
	if h.ProductCreator == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product deletion workflow is not configured"})
		return
	}
	id := c.Param("id")
	if err := h.ProductCreator.DeleteProduct(c, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "product", "deleted": true})
}

func (h *ModuleHTTP) updateProduct(c *gin.Context) {
	if h.ProductCreator == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product update workflow is not configured"})
		return
	}
	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if _, err := h.Products.GetProduct(c, c.Param("id")); err != nil {
		writeModuleError(c, err)
		return
	}
	if err := h.ProductCreator.UpdateProduct(c, c.Param("id"), input); err != nil {
		writeModuleError(c, err)
		return
	}
	product, err := h.Products.GetProduct(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": product})
}

func (h *ModuleHTTP) listProductTypes(c *gin.Context) {
	limit, offset := paginationWithMax(c, 1000)
	rows, count, err := h.Products.ListProductTypes(c, queryIDs(c), limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("product_types", rows, count, offset, limit))
}

func (h *ModuleHTTP) getProductType(c *gin.Context) {
	item, err := h.Products.GetProductType(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_type": item})
}

func (h *ModuleHTTP) listProductTags(c *gin.Context) {
	limit, offset := paginationWithMax(c, 1000)
	rows, count, err := h.Products.ListProductTags(c, queryIDs(c), limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("product_tags", rows, count, offset, limit))
}

func (h *ModuleHTTP) getProductTag(c *gin.Context) {
	item, err := h.Products.GetProductTag(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_tag": item})
}

func (h *ModuleHTTP) createProductType(c *gin.Context) {
	var input productmodels.ProductTypeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if h.ProductCatalog == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product type workflow is not configured"})
		return
	}
	items, err := h.ProductCatalog.CreateProductTypes(c, []productmodels.ProductTypeInput{input})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(items) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "unexpected_state", "message": "create-product-types returned no result"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_type": items[0]})
}

func (h *ModuleHTTP) updateProductType(c *gin.Context) {
	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if h.ProductCatalog == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product type workflow is not configured"})
		return
	}
	item, err := h.ProductCatalog.UpdateProductType(c, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_type": item})
}

func (h *ModuleHTTP) deleteProductType(c *gin.Context) {
	id := c.Param("id")
	if h.ProductCatalog == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product type workflow is not configured"})
		return
	}
	if err := h.ProductCatalog.DeleteProductTypes(c, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "product_type", "deleted": true})
}

func (h *ModuleHTTP) createProductTag(c *gin.Context) {
	var input productmodels.ProductTagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if h.ProductCatalog == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product tag workflow is not configured"})
		return
	}
	items, err := h.ProductCatalog.CreateProductTags(c, []productmodels.ProductTagInput{input})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(items) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"type": "unexpected_state", "message": "create-product-tags returned no result"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_tag": items[0]})
}

func (h *ModuleHTTP) updateProductTag(c *gin.Context) {
	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if h.ProductCatalog == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product tag workflow is not configured"})
		return
	}
	item, err := h.ProductCatalog.UpdateProductTag(c, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_tag": item})
}

func (h *ModuleHTTP) deleteProductTag(c *gin.Context) {
	id := c.Param("id")
	if h.ProductCatalog == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "product tag workflow is not configured"})
		return
	}
	if err := h.ProductCatalog.DeleteProductTags(c, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "product_tag", "deleted": true})
}

func (h *ModuleHTTP) listLayoutConfigurations(c *gin.Context) {
	limit, offset := paginationFor(c, 20, 1000)
	userID, _ := c.Get("actor_id")
	filter := settingsmodels.LayoutConfigurationFilter{IDs: queryValues(c, "id"), Zones: queryValues(c, "zone"), UserIDs: queryValues(c, "user_id")}
	if raw := c.Query("is_system_default"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "is_system_default must be a boolean"})
			return
		}
		filter.IsSystemDefault = &value
	}
	rows, count, err := h.Layouts.ListLayoutConfigurations(c, userID.(string), filter, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	selected := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		selected = append(selected, projectLayout(row, c.Query("fields")))
	}
	c.JSON(http.StatusOK, listResponse("layout_configurations", selected, count, offset, limit))
}

func projectLayout(row settingsmodels.LayoutConfiguration, requested string) map[string]any {
	data, _ := json.Marshal(row)
	all := map[string]any{}
	_ = json.Unmarshal(data, &all)
	defaults := []string{"id", "zone", "user_id", "is_system_default", "configuration", "created_at", "updated_at"}
	if requested == "" {
		requested = strings.Join(defaults, ",")
	}
	fields := strings.Split(requested, ",")
	out := map[string]any{}
	includeAll, useDefaults := false, false
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "*" {
			includeAll = true
		}
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			useDefaults = true
		}
	}
	if includeAll {
		return all
	}
	if useDefaults {
		for _, field := range defaults {
			out[field] = all[field]
		}
	}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if strings.HasPrefix(field, "-") {
			delete(out, strings.TrimPrefix(field, "-"))
			continue
		}
		field = strings.TrimPrefix(field, "+")
		if value, ok := all[field]; ok {
			out[field] = value
		}
	}
	return out
}

func (h *ModuleHTTP) getLayoutConfiguration(c *gin.Context) {
	userID, _ := c.Get("actor_id")
	result, err := h.Layouts.GetLayoutConfiguration(c, c.Param("zone"), userID.(string))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ModuleHTTP) setLayoutConfiguration(c *gin.Context) {
	var body struct {
		IsDefault     *bool           `json:"is_default"`
		Configuration json.RawMessage `json:"configuration"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Configuration) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "configuration is required"})
		return
	}
	var cfg struct {
		Widgets map[string]struct {
			Hidden  *bool    `json:"hidden"`
			Section *string  `json:"section"`
			Order   *float64 `json:"order"`
		} `json:"widgets"`
	}
	if err := json.Unmarshal(body.Configuration, &cfg); err != nil || cfg.Widgets == nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "configuration.widgets must be an object"})
		return
	}
	actor, _ := c.Get("actor_id")
	actorID, _ := actor.(string)
	isDefault := body.IsDefault != nil && *body.IsDefault
	if err := settingsflows.SetLayoutConfiguration(c, h.Layouts, c.Param("zone"), actorID, isDefault, body.Configuration); err != nil {
		writeModuleError(c, err)
		return
	}
	result, err := h.Layouts.GetLayoutConfiguration(c, c.Param("zone"), actorID)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ModuleHTTP) clearLayoutConfiguration(c *gin.Context) {
	actor, _ := c.Get("actor_id")
	actorID, _ := actor.(string)
	if err := settingsflows.ClearLayoutConfiguration(c, h.Layouts, c.Param("zone"), actorID); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ModuleHTTP) propertyLabelsNotFound(c *gin.Context) bool {
	if !h.PropertyLabelsEnabled {
		c.JSON(http.StatusNotFound, gin.H{"type": "not_found", "message": "Route not found"})
		return true
	}
	if h.PropertyLabels == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"type": "not_implemented", "message": "property label module is not configured"})
		return true
	}
	return false
}
func (h *ModuleHTTP) listPropertyLabels(c *gin.Context) {
	if h.propertyLabelsNotFound(c) {
		return
	}
	limit, offset := paginationFor(c, 20, 1000)
	rows, count, err := h.PropertyLabels.ListPropertyLabels(c, settingsmodels.PropertyLabelFilter{Entity: c.Query("entity"), Property: c.Query("property"), Query: c.Query("q")}, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listResponse("property_labels", rows, count, offset, limit))
}
func (h *ModuleHTTP) getPropertyLabel(c *gin.Context) {
	if h.propertyLabelsNotFound(c) {
		return
	}
	row, err := h.PropertyLabels.GetPropertyLabel(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"property_label": row})
}
func (h *ModuleHTTP) createPropertyLabel(c *gin.Context) {
	if h.propertyLabelsNotFound(c) {
		return
	}
	var in settingsmodels.PropertyLabelInput
	if err := c.ShouldBindJSON(&in); err != nil || in.Entity == "" || in.Property == "" || in.Label == "" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "entity, property, and label are required"})
		return
	}
	rows, err := settingspropertyflows.CreatePropertyLabels(c, h.PropertyLabels, []settingsmodels.PropertyLabelInput{in})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	if len(rows) == 0 {
		c.JSON(500, gin.H{"type": "unexpected_state", "message": "create-property-labels returned no result"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"property_label": rows[0]})
}
func (h *ModuleHTTP) updatePropertyLabel(c *gin.Context) {
	if h.propertyLabelsNotFound(c) {
		return
	}
	var in settingspropertyflows.PropertyLabelUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	in.ID = c.Param("id")
	row, err := settingspropertyflows.UpdatePropertyLabel(c, h.PropertyLabels, in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"property_label": row})
}
func (h *ModuleHTTP) deletePropertyLabel(c *gin.Context) {
	if h.propertyLabelsNotFound(c) {
		return
	}
	id := c.Param("id")
	if err := settingspropertyflows.DeletePropertyLabels(c, h.PropertyLabels, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "property_label", "deleted": true})
}
func (h *ModuleHTTP) batchPropertyLabels(c *gin.Context) {
	if h.propertyLabelsNotFound(c) {
		return
	}
	var in settingspropertyflows.PropertyLabelBatch
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	out, err := settingspropertyflows.BatchPropertyLabels(c, h.PropertyLabels, in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	deleted := make([]gin.H, 0, len(out.Deleted))
	for _, v := range out.Deleted {
		deleted = append(deleted, gin.H{"id": v.ID, "object": "property_label", "deleted": true})
	}
	c.JSON(200, gin.H{"created": out.Created, "updated": out.Updated, "deleted": deleted})
}

func (h *ModuleHTTP) listNotifications(c *gin.Context) {
	limit, offset := paginationFor(c, 50, 1000)
	filter := notificationmodels.NotificationFilter{IDs: queryValues(c, "id"), To: queryValues(c, "to"), Channel: queryValues(c, "channel"), Query: c.Query("q")}
	order := c.DefaultQuery("order", "-created_at")
	if order != "-created_at" && order != "created_at" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "order must be created_at or -created_at"})
		return
	}
	rows, count, err := h.Notifications.List(c, filter, limit, offset, order)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	selected := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		selected = append(selected, projectNotification(row, c.Query("fields")))
	}
	c.JSON(http.StatusOK, listResponse("notifications", selected, count, offset, limit))
}

func (h *ModuleHTTP) getNotification(c *gin.Context) {
	item, err := h.Notifications.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"notification": projectNotification(item, c.Query("fields"))})
}

func queryValues(c *gin.Context, key string) []string {
	values := c.QueryArray(key + "[]")
	if len(values) == 0 {
		values = c.QueryArray(key)
	}
	out := []string{}
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			if strings.TrimSpace(part) != "" {
				out = append(out, strings.TrimSpace(part))
			}
		}
	}
	return out
}

func projectNotification(row notificationmodels.Notification, requested string) map[string]any {
	data, _ := json.Marshal(row)
	all := map[string]any{}
	_ = json.Unmarshal(data, &all)
	fields := strings.Split(requested, ",")
	defaultFields := []string{"id", "to", "channel", "template", "data", "trigger_type", "resource_id", "resource_type", "receiver_id", "created_at", "updated_at"}
	if requested == "" {
		fields = defaultFields
	}
	out := map[string]any{}
	includeAll, useDefaults := false, false
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "*" {
			includeAll = true
		}
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			useDefaults = true
		}
	}
	if includeAll {
		for key, value := range all {
			out[key] = value
		}
	}
	if useDefaults {
		for _, field := range defaultFields {
			out[field] = all[field]
		}
	}
	if includeAll {
		return out
	}
	if requested == "" {
		return out
	}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "*" {
			continue
		}
		if strings.HasPrefix(field, "-") {
			delete(out, strings.TrimPrefix(field, "-"))
			continue
		}
		field = strings.TrimPrefix(field, "+")
		if value, ok := all[field]; ok {
			out[field] = value
		}
	}
	return out
}

func (h *ModuleHTTP) listRegions(c *gin.Context) {
	rows, err := h.Regions.List(c)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	writePaged(c, "regions", rows)
}

func (h *ModuleHTTP) getRegion(c *gin.Context) {
	row, err := h.Regions.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"region": row})
}

func (h *ModuleHTTP) createRegion(c *gin.Context) {
	var input regionmodels.RegionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if input.Name == nil || input.CurrencyCode == nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "name and currency_code are required"})
		return
	}
	row, err := h.RegionWorkflow.CreateRegion(c, input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"region": row})
}

func (h *ModuleHTTP) updateRegion(c *gin.Context) {
	var input regionmodels.RegionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := h.RegionWorkflow.UpdateRegion(c, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"region": row})
}

func (h *ModuleHTTP) deleteRegion(c *gin.Context) {
	id := c.Param("id")
	if err := h.RegionWorkflow.DeleteRegion(c, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "region", "deleted": true})
}

func (h *ModuleHTTP) listSalesChannels(c *gin.Context) {
	rows, err := h.SalesChannels.List(c)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	writePaged(c, "sales_channels", rows)
}

func (h *ModuleHTTP) getSalesChannel(c *gin.Context) {
	row, err := h.SalesChannels.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sales_channel": row})
}

func (h *ModuleHTTP) linkSalesChannelProducts(c *gin.Context) {
	var input struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	id := c.Param("id")
	if err := h.SalesChannelWorkflow.LinkProducts(c, id, input.Add, input.Remove); err != nil {
		writeModuleError(c, err)
		return
	}
	row, err := h.SalesChannels.Get(c, id)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sales_channel": row})
}

func (h *ModuleHTTP) createSalesChannel(c *gin.Context) {
	input, err := decodeSalesChannelInput(c, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := h.SalesChannelWorkflow.CreateSalesChannel(c, input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sales_channel": row})
}

func (h *ModuleHTTP) updateSalesChannel(c *gin.Context) {
	input, err := decodeSalesChannelInput(c, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := h.SalesChannelWorkflow.UpdateSalesChannel(c, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sales_channel": row})
}

func (h *ModuleHTTP) deleteSalesChannel(c *gin.Context) {
	id := c.Param("id")
	if err := h.SalesChannelWorkflow.DeleteSalesChannel(c, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "sales-channel", "deleted": true})
}

func decodeSalesChannelInput(c *gin.Context, create bool) (salesmodels.Input, error) {
	var fields map[string]json.RawMessage
	if err := c.ShouldBindJSON(&fields); err != nil {
		return salesmodels.Input{}, err
	}
	for key := range fields {
		if key != "name" && key != "description" && key != "is_disabled" && key != "metadata" {
			return salesmodels.Input{}, fmt.Errorf("unknown field %q", key)
		}
	}
	var input salesmodels.Input
	if raw, ok := fields["name"]; ok {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
			return input, errors.New("name must be a non-empty string")
		}
		input.Name = &value
	} else if create {
		return input, errors.New("name is required")
	}
	if raw, ok := fields["description"]; ok {
		input.DescriptionSet = true
		if string(raw) != "null" {
			var value string
			if err := json.Unmarshal(raw, &value); err != nil {
				return input, errors.New("description must be a string or null")
			}
			input.Description = &value
		}
	}
	if raw, ok := fields["is_disabled"]; ok {
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return input, errors.New("is_disabled must be a boolean")
		}
		input.IsDisabled = &value
	}
	if raw, ok := fields["metadata"]; ok {
		input.MetadataSet = true
		input.Metadata = raw
	}
	return input, nil
}

func (h *ModuleHTTP) listStores(c *gin.Context) {
	rows, err := h.Stores.List(c)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	writePaged(c, "stores", rows)
}

func (h *ModuleHTTP) getStore(c *gin.Context) {
	row, err := h.Stores.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"store": row})
}

func (h *ModuleHTTP) updateStore(c *gin.Context) {
	var input storemodels.StoreInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	previous, err := h.Stores.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	updated, err := storeflows.UpdateStores(c, h.Stores, nil, previous.ID, input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"store": updated[0]})
}

func (h *ModuleHTTP) listCurrencies(c *gin.Context) {
	rows, err := h.Currencies.ListCurrencies(c, queryIDs(c))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	writePaged(c, "currencies", rows)
}

func (h *ModuleHTTP) getCurrency(c *gin.Context) {
	row, err := h.Currencies.RetrieveCurrency(c, c.Param("code"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"currency": row})
}

func (h *ModuleHTTP) listUsers(c *gin.Context) {
	rows, err := h.Users.ListUsers(c, queryIDs(c))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	writePaged(c, "users", rows)
}

func (h *ModuleHTTP) getUser(c *gin.Context) {
	row, err := h.Users.GetUser(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": row})
}

func (h *ModuleHTTP) updateUser(c *gin.Context) {
	var fields map[string]json.RawMessage
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	input := usermodels.UserInput{}
	for key, raw := range fields {
		switch key {
		case "first_name":
			input.FirstNameSet = true
			if string(raw) != "null" {
				var value string
				if err := json.Unmarshal(raw, &value); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "first_name must be a string or null"})
					return
				}
				input.FirstName = &value
			}
		case "last_name":
			input.LastNameSet = true
			if string(raw) != "null" {
				var value string
				if err := json.Unmarshal(raw, &value); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "last_name must be a string or null"})
					return
				}
				input.LastName = &value
			}
		case "avatar_url":
			input.AvatarURLSet = true
			if string(raw) != "null" {
				var value string
				if err := json.Unmarshal(raw, &value); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "avatar_url must be a string or null"})
					return
				}
				input.AvatarURL = &value
			}
		case "metadata":
			input.MetadataSet = true
			input.Metadata = raw
		default:
			c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": fmt.Sprintf("unknown field %q", key)})
			return
		}
	}
	row, err := h.UserWorkflow.UpdateUser(c, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": row})
}

func (h *ModuleHTTP) deleteUser(c *gin.Context) {
	id := c.Param("id")
	actorID, _ := c.Get("actor_id")
	if contextValue(actorID) == id {
		c.JSON(http.StatusForbidden, gin.H{"type": "not_allowed", "message": "A user cannot delete itself"})
		return
	}
	if err := h.UserAccountWorkflow.RemoveUserAccount(c, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "user", "deleted": true})
}

func (h *ModuleHTTP) listUserAuthProviders(c *gin.Context) {
	providers, err := h.UserAuthProviders.ListProviderNamesByActor(c, "user", c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

func (h *ModuleHTTP) resetUserPassword(c *gin.Context) {
	user, err := h.Users.GetUser(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	token, err := h.UserResetToken.GenerateUserResetToken(c, user.Email)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *ModuleHTTP) listUserRoles(c *gin.Context) {
	if _, err := h.Users.GetUser(c, c.Param("id")); err != nil {
		writeModuleError(c, err)
		return
	}
	rows, err := h.UserRolesWorkflow.ListUserRoles(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	roleIDs := c.QueryArray("role_id[]")
	if len(roleIDs) == 0 {
		roleIDs = c.QueryArray("role_id")
	}
	if len(roleIDs) > 0 {
		allowed := map[string]bool{}
		for _, id := range roleIDs {
			allowed[id] = true
		}
		filtered := rows[:0]
		for _, row := range rows {
			if allowed[row.ID] {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	limit, offset := paginationFor(c, 50, 100)
	all := rows
	start := offset
	if start > len(all) {
		start = len(all)
	}
	end := start + limit
	if end > len(all) {
		end = len(all)
	}
	c.JSON(http.StatusOK, gin.H{"roles": all[start:end], "count": len(all), "offset": offset, "limit": limit})
}

func (h *ModuleHTTP) assignUserRoles(c *gin.Context) {
	var input struct {
		Roles []string `json:"roles"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Roles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "roles must contain at least one role id"})
		return
	}
	if _, err := h.Users.GetUser(c, c.Param("id")); err != nil {
		writeModuleError(c, err)
		return
	}
	actorID, _ := c.Get("actor_id")
	actorType, _ := c.Get("actor_type")
	if err := h.UserRolesWorkflow.AssignUserRoles(c, contextValue(actorID), contextValue(actorType), c.Param("id"), input.Roles); err != nil {
		writeModuleError(c, err)
		return
	}
	rows, err := h.UserRolesWorkflow.ListUserRoles(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"roles": rows})
}

func (h *ModuleHTTP) removeUserRoles(c *gin.Context) {
	var input struct {
		Roles []string `json:"roles"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Roles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "roles must contain at least one role id"})
		return
	}
	if _, err := h.Users.GetUser(c, c.Param("id")); err != nil {
		writeModuleError(c, err)
		return
	}
	actorID, _ := c.Get("actor_id")
	actorType, _ := c.Get("actor_type")
	if err := h.UserRolesWorkflow.RemoveUserRoles(c, contextValue(actorID), contextValue(actorType), c.Param("id"), input.Roles); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ids": input.Roles, "object": "user_role", "deleted": true})
}

func (h *ModuleHTTP) removeSingleUserRole(c *gin.Context) {
	actorID, _ := c.Get("actor_id")
	actorType, _ := c.Get("actor_type")
	roleID := c.Param("role_id")
	if err := h.UserRolesWorkflow.RemoveUserRoles(c, contextValue(actorID), contextValue(actorType), c.Param("id"), []string{roleID}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": roleID, "object": "user_role", "deleted": true})
}

func (h *ModuleHTTP) getCurrentUser(c *gin.Context) {
	id, _ := c.Get("actor_id")
	actorID, _ := id.(string)
	if actorID == "" {
		c.JSON(http.StatusNotFound, gin.H{"type": "not_found", "message": "User ID not found"})
		return
	}
	row, err := h.Users.GetUser(c, actorID)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": row})
}

func pagination(c *gin.Context) (int, int) {
	return paginationFor(c, 20, 100)
}

func paginationWithMax(c *gin.Context, maxLimit int) (int, int) {
	return paginationFor(c, 20, maxLimit)
}

func paginationFor(c *gin.Context, defaultLimit, maxLimit int) (int, int) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}
	return limit, offset
}

func queryIDs(c *gin.Context) []string {
	ids := c.QueryArray("id[]")
	if len(ids) == 0 {
		ids = c.QueryArray("id")
	}
	return ids
}

func writePaged[T any](c *gin.Context, key string, rows []T) {
	limit, offset := pagination(c)
	total := len(rows)
	rows = page(rows, offset, limit)
	c.JSON(http.StatusOK, listResponse(key, rows, total, offset, limit))
}

func page[T any](rows []T, offset, limit int) []T {
	if offset >= len(rows) {
		return []T{}
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end]
}

func listResponse[T any](key string, rows []T, total, offset, limit int) gin.H {
	return gin.H{key: rows, "count": total, "offset": offset, "limit": limit}
}

func writeModuleError(c *gin.Context, err error) {
	if errors.Is(err, fulfillmentdomain.ErrNotAllowed) {
		c.JSON(http.StatusForbidden, gin.H{"type": "not_allowed", "message": err.Error()})
		return
	}
	if errors.Is(err, fulfillmentdomain.ErrInvalidData) {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if errors.Is(err, userroleflows.ErrUserRolePermission) {
		c.JSON(http.StatusForbidden, gin.H{"type": "not_allowed", "message": err.Error()})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"type": "not_found", "message": "Record not found"})
		return
	}
	if errors.Is(err, productdomain.ErrInvalidData) {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if errors.Is(err, salesdomain.ErrInvalidData) {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"type": "database_error", "message": err.Error()})
}

func contextValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
