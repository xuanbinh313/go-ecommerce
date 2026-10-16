package authhttp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	authapp "github.com/example/medusa-go-clone/internal/modules/auth/application"
	authdomain "github.com/example/medusa-go-clone/internal/modules/auth/domain"
	"github.com/example/medusa-go-clone/internal/modules/auth/domain/models"
	usermodels "github.com/example/medusa-go-clone/internal/modules/user/domain/models"
	"github.com/gin-gonic/gin"
)

type AuthService interface {
	ParseToken(string) (*authdomain.TokenClaims, error)
	RefreshActorToken(context.Context, *authdomain.TokenClaims) (authapp.LoginResult, error)
	ListProviderNamesByIdentity(context.Context, string) ([]string, error)
	SetActorMetadata(context.Context, string, string, string) (string, error)
	ConfirmAuthVerification(context.Context, models.ConfirmAuthVerificationInput, time.Duration) (models.AuthVerification, error)
	ListAuthMFA(context.Context, string) ([]models.AuthMFAFactor, error)
	GetAuthMFA(context.Context, string, string) (models.AuthMFAFactor, error)
	StartAuthMFA(context.Context, models.AuthMFAStartInput) (models.AuthMFAEnrollment, error)
	VerifyAuthMFASetup(context.Context, string, string, string) (models.AuthMFAFactor, error)
	DisableAuthMFA(context.Context, string, string, string, string) (models.AuthMFAFactor, error)
	GenerateAuthMFARecoveryCodes(context.Context, string, int) ([]string, error)
	VerifyAuthMFAChallenge(context.Context, string, string, string, string) (models.AuthMFAChallenge, error)
	IssueMFAChallengeToken(context.Context, string, string, string, models.AuthMFAChallenge) (authapp.LoginResult, error)
	CreateIdentityWithProvider(context.Context, string, string, string) (string, error)
	RegistrationToken(string, string, string) (authapp.LoginResult, error)
	DeleteIdentity(context.Context, string) error
	AuthenticateProvider(context.Context, string, string, string, string) (authapp.LoginResult, error)
	StartOAuth(context.Context, string, string) (string, error)
	CompleteOAuth(context.Context, string, string, string, string, map[string]string) (authapp.LoginResult, error)
	OAuthProviderConfigured(string) bool
	OAuthProviderDisplayName(string) string
	ConsumePasswordResetToken(context.Context, string, string, string) error
	UpdateProviderPasswordForEntity(context.Context, string, string, string, string) error
}

type PasswordResetWorkflow interface {
	Generate(context.Context, string, string, string, map[string]any) error
}
type AuthVerificationWorkflow interface {
	Request(context.Context, models.RequestAuthVerificationInput, time.Duration) (models.AuthVerification, error)
}
type UserAccountReader interface {
	FindUsersByEmail(context.Context, []string) ([]usermodels.User, error)
	GetUser(context.Context, string) (usermodels.User, error)
}
type UserAccountWorkflow interface {
	CreateUserAccount(context.Context, string, usermodels.UserInput) (usermodels.User, error)
	SetAuthAppMetadata(context.Context, string, string, string) error
}
type AuthEventPublisher interface {
	Publish(context.Context, string, []map[string]any) error
}
type HTTP struct {
	Service       AuthService
	PasswordReset PasswordResetWorkflow
	Verification  AuthVerificationWorkflow
	Users         UserAccountReader
	UserAccount   UserAccountWorkflow
	Events        AuthEventPublisher
}

func (h *HTTP) Register(router *gin.RouterGroup) {
	group := router.Group("/auth")
	// Medusa exposes provider discovery as /auth/:actor_type/providers.
	// Register the static segment explicitly so it is not treated as the
	// provider identifier on the generic authenticate route.
	group.GET("/:actor_type/providers", h.providers)
	group.POST("/token/refresh", h.refreshToken)
	group.POST("/verification/request", h.requestVerification)
	group.POST("/verification/confirm", h.confirmVerification)
	group.POST("/:actor_type/user", h.createUserFromIdentity)
	group.GET("/mfa/factors", h.listMFAFactors)
	group.POST("/mfa/factors", h.startMFAFactor)
	group.POST("/mfa/factors/:id/verify", h.verifyMFAFactor)
	group.DELETE("/mfa/factors/:id", h.disableMFAFactor)
	group.POST("/mfa/recovery-codes", h.generateMFARecoveryCodes)
	group.POST("/mfa/challenges/:id/verify", h.verifyMFAChallenge)
	group.POST("/:actor_type/:auth_provider/register", h.register)
	group.POST("/:actor_type/:auth_provider", h.authenticate)
	group.GET("/:actor_type/:auth_provider", h.authenticate)
	group.GET("/:actor_type/:auth_provider/callback", h.oauthCallback)
	group.POST("/:actor_type/:auth_provider/callback", h.oauthCallback)
	group.POST("/:actor_type/:auth_provider/reset-password", h.resetPassword)
	group.POST("/:actor_type/:auth_provider/update", h.updatePassword)
}

func (h *HTTP) verifyMFAChallenge(c *gin.Context) {
	claims, ok := h.authIdentityClaims(c)
	if !ok {
		return
	}
	var input struct {
		Method string `json:"method"`
		Code   string `json:"code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Method) == "" || strings.TrimSpace(input.Code) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "method and code are required"})
		return
	}
	challenge, err := h.Service.VerifyAuthMFAChallenge(c, c.Param("id"), claims.AuthIdentityID, input.Method, input.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	result, err := h.Service.IssueMFAChallengeToken(c, claims.AuthIdentityID, claims.ActorType, claims.Provider, challenge)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": result.Token})
}

func (h *HTTP) listMFAFactors(c *gin.Context) {
	claims, ok := h.authIdentityClaims(c)
	if !ok {
		return
	}
	factors, err := h.Service.ListAuthMFA(c, claims.AuthIdentityID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mfa_factors": factors})
}

func (h *HTTP) generateMFARecoveryCodes(c *gin.Context) {
	claims, ok := h.authIdentityClaims(c)
	if !ok {
		return
	}
	var input struct {
		Count *int `json:"count"`
	}
	if err := c.ShouldBindJSON(&input); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	count := 0
	if input.Count != nil {
		count = *input.Count
	}
	codes, err := h.Service.GenerateAuthMFARecoveryCodes(c, claims.AuthIdentityID, count)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if err = h.publishAuthEvent(c, "auth.mfa_recovery_codes_generated", map[string]any{"auth_identity_id": claims.AuthIdentityID, "count": len(codes)}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"recovery_codes": codes})
}

func (h *HTTP) startMFAFactor(c *gin.Context) {
	claims, ok := h.authIdentityClaims(c)
	if !ok {
		return
	}
	var input struct {
		Provider string          `json:"provider"`
		Label    *string         `json:"label"`
		Issuer   string          `json:"issuer"`
		Metadata json.RawMessage `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "provider is required"})
		return
	}
	enrollment, err := h.Service.StartAuthMFA(c, models.AuthMFAStartInput{AuthIdentityID: claims.AuthIdentityID, Provider: input.Provider, Label: input.Label, Issuer: input.Issuer, Metadata: input.Metadata})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mfa_factor": enrollment.MFA, "secret": enrollment.Secret, "otpauth_url": enrollment.OTPAuthURL})
}

func (h *HTTP) verifyMFAFactor(c *gin.Context) {
	claims, ok := h.authIdentityClaims(c)
	if !ok {
		return
	}
	var input struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Code) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "code is required"})
		return
	}
	previous, err := h.Service.GetAuthMFA(c, c.Param("id"), claims.AuthIdentityID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "MFA factor was not found"})
		return
	}
	factor, err := h.Service.VerifyAuthMFASetup(c, c.Param("id"), claims.AuthIdentityID, input.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if previous.Status != "enabled" && factor.Status == "enabled" {
		if err = h.publishMFAEvent(c, "auth.mfa_enabled", factor); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"mfa_factor": factor})
}

func (h *HTTP) disableMFAFactor(c *gin.Context) {
	claims, ok := h.authIdentityClaims(c)
	if !ok {
		return
	}
	var input struct {
		Method string `json:"method"`
		Code   string `json:"code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	previous, err := h.Service.GetAuthMFA(c, c.Param("id"), claims.AuthIdentityID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "MFA factor was not found"})
		return
	}
	factor, err := h.Service.DisableAuthMFA(c, c.Param("id"), claims.AuthIdentityID, input.Method, input.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if previous.Status != "disabled" && factor.Status == "disabled" {
		if err = h.publishMFAEvent(c, "auth.mfa_disabled", factor); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"mfa_factor": factor})
}

func (h *HTTP) authIdentityClaims(c *gin.Context) (*authdomain.TokenClaims, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return nil, false
	}
	claims, err := h.Service.ParseToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
	if err != nil || claims == nil || claims.AuthIdentityID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return nil, false
	}
	return claims, true
}

func (h *HTTP) publishMFAEvent(c *gin.Context, name string, factor models.AuthMFAFactor) error {
	return h.publishAuthEvent(c, name, map[string]any{"auth_identity_id": factor.AuthIdentityID, "mfa_id": factor.ID, "provider": factor.Provider})
}

func (h *HTTP) publishAuthEvent(c *gin.Context, name string, data map[string]any) error {
	if h.Events == nil {
		return nil
	}
	return h.Events.Publish(c, name, []map[string]any{data})
}

func (h *HTTP) createUserFromIdentity(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	claims, err := h.Service.ParseToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
	if err != nil || claims == nil || claims.ActorType != "user" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	if claims.ActorID != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "The user is already registered and cannot create a new account."})
		return
	}
	email, _ := claims.UserMetadata["email"].(string)
	if strings.TrimSpace(email) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email is required to create a user account."})
		return
	}
	providerNames, err := h.Service.ListProviderNamesByIdentity(c, claims.AuthIdentityID)
	if err != nil || len(providerNames) != 1 || providerNames[0] != c.Param("actor_type") {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Only the route auth provider can create a user account for this authentication context"})
		return
	}
	if h.Users == nil || h.UserAccount == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "user account workflow is not configured"})
		return
	}
	users, err := h.Users.FindUsersByEmail(c, []string{email})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if len(users) > 0 {
		if err = h.UserAccount.SetAuthAppMetadata(c, claims.AuthIdentityID, "user", users[0].ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		user, getErr := h.Users.GetUser(c, users[0].ID)
		if getErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": getErr.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": user})
		return
	}
	userInput := usermodels.UserInput{Email: &email}
	if value, ok := claims.UserMetadata["given_name"].(string); ok {
		userInput.FirstName = &value
	}
	if value, ok := claims.UserMetadata["family_name"].(string); ok {
		userInput.LastName = &value
	}
	user, err := h.UserAccount.CreateUserAccount(c, claims.AuthIdentityID, userInput)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *HTTP) requestVerification(c *gin.Context) {
	claims, ok := h.parseClaims(c)
	if !ok {
		return
	}
	var input struct {
		EntityID     string          `json:"entity_id"`
		EntityType   string          `json:"entity_type"`
		CodeProvider string          `json:"code_provider"`
		Metadata     json.RawMessage `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.EntityID) == "" || strings.TrimSpace(input.EntityType) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "entity_id and entity_type are required"})
		return
	}
	if input.CodeProvider == "" {
		input.CodeProvider = "token"
	}
	if input.Metadata == nil {
		input.Metadata = json.RawMessage("null")
	}
	if h.Verification == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "verification workflow is not configured"})
		return
	}
	verification, err := h.Verification.Request(c, models.RequestAuthVerificationInput{AuthIdentityID: claims.AuthIdentityID, EntityID: input.EntityID, EntityType: input.EntityType, CodeProvider: input.CodeProvider, Metadata: input.Metadata}, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"verification": verification})
}

func (h *HTTP) confirmVerification(c *gin.Context) {
	var input struct {
		Code         string `json:"code"`
		CodeProvider string `json:"code_provider"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Code) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "code is required"})
		return
	}
	authIdentityID := ""
	if header := c.GetHeader("Authorization"); strings.HasPrefix(header, "Bearer ") {
		claims, err := h.Service.ParseToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil || claims == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			return
		}
		authIdentityID = claims.AuthIdentityID
	}
	if input.CodeProvider == "" {
		input.CodeProvider = "token"
	}
	verification, err := h.Service.ConfirmAuthVerification(c, models.ConfirmAuthVerificationInput{AuthIdentityID: authIdentityID, Code: input.Code, CodeProvider: input.CodeProvider}, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entity_id": verification.EntityID, "entity_type": verification.EntityType, "code_provider": verification.CodeProvider, "verified_at": verification.VerifiedAt})
}

func (h *HTTP) refreshToken(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	claims, err := h.Service.ParseToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
	if err != nil || claims == nil || claims.Provider == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	result, err := h.Service.RefreshActorToken(c, claims)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": result.Token})
}

func (h *HTTP) resetPassword(c *gin.Context) {
	if c.Param("auth_provider") != "emailpass" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "unknown authentication provider"})
		return
	}
	var input struct {
		Identifier string         `json:"identifier"`
		Metadata   map[string]any `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Identifier) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "identifier is required"})
		return
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	if h.PasswordReset != nil {
		_ = h.PasswordReset.Generate(c, input.Identifier, c.Param("actor_type"), c.Param("auth_provider"), input.Metadata)
	}
	c.Status(http.StatusCreated)
}

func (h *HTTP) updatePassword(c *gin.Context) {
	if c.Param("auth_provider") != "emailpass" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	claims, err := h.Service.ParseToken(token)
	if err != nil || claims == nil || claims.EntityID == "" || claims.Purpose != "reset" || claims.TokenID == "" || claims.ActorType != c.Param("actor_type") || claims.Provider != c.Param("auth_provider") {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if err = c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	if err = h.Service.ConsumePasswordResetToken(c, claims.TokenID, claims.Provider, claims.EntityID); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	if input.Password != "" {
		if err = h.Service.UpdateProviderPasswordForEntity(c, claims.Provider, claims.ActorType, claims.EntityID, input.Password); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
func (h *HTTP) providers(c *gin.Context) {
	providers := []gin.H{{
		"id":           "emailpass",
		"identifier":   "emailpass",
		"display_name": "Email/Password Authentication",
		"flow":         "credentials",
	}}
	if h.Service.OAuthProviderConfigured("github") {
		providers = append(providers, gin.H{"id": "github", "identifier": "github", "display_name": "Github Authentication", "flow": "oauth"})
	}
	if h.Service.OAuthProviderConfigured("google") {
		providers = append(providers, gin.H{"id": "google", "identifier": "google", "display_name": "Google Authentication", "flow": "oauth"})
	}
	if h.Service.OAuthProviderConfigured("oidc") {
		providers = append(providers, gin.H{"id": "oidc", "identifier": "oidc", "display_name": h.Service.OAuthProviderDisplayName("oidc"), "flow": "oauth"})
	}
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}
func (h *HTTP) AuthMiddleware(allowUnregistered bool) gin.HandlerFunc {
	return h.AuthenticateMiddleware(allowUnregistered)
}
func (h *HTTP) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := h.parseClaims(c)
		if !ok {
			return
		}
		if claims.ActorID == "" || claims.ActorType != "user" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin user token required"})
			return
		}
		h.setClaims(c, claims)
		c.Next()
	}
}
func (h *HTTP) AuthenticateMiddleware(allowUnregistered bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := h.parseClaims(c)
		if !ok {
			return
		}
		if !allowUnregistered && claims.ActorID == "" {
			c.AbortWithStatusJSON(403, gin.H{"error": "token is not linked to an actor"})
			return
		}
		h.setClaims(c, claims)
		c.Next()
	}
}
func (h *HTTP) parseClaims(c *gin.Context) (*authdomain.TokenClaims, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		c.AbortWithStatusJSON(401, gin.H{"error": "bearer token required"})
		return nil, false
	}
	claims, err := h.Service.ParseToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired token"})
		return nil, false
	}
	return claims, true
}
func (h *HTTP) setClaims(c *gin.Context, claims *authdomain.TokenClaims) {
	c.Set("auth_identity_id", claims.AuthIdentityID)
	c.Set("actor_id", claims.ActorID)
	c.Set("actor_type", claims.ActorType)
}
func (h *HTTP) register(c *gin.Context) {
	if c.Param("auth_provider") != "emailpass" {
		c.JSON(400, gin.H{"error": "unknown authentication provider"})
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if c.ShouldBindJSON(&input) != nil || strings.TrimSpace(input.Email) == "" || input.Password == "" {
		c.JSON(400, gin.H{"error": "email and password are required"})
		return
	}
	identityID, err := h.Service.CreateIdentityWithProvider(c, c.Param("auth_provider"), input.Email, input.Password)
	if err != nil {
		writeError(c, err, 400)
		return
	}
	result, err := h.Service.RegistrationToken(identityID, c.Param("actor_type"), c.Param("auth_provider"))
	if err != nil {
		_ = h.Service.DeleteIdentity(c, identityID)
		writeError(c, err, 500)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": result.Token})
}
func (h *HTTP) authenticate(c *gin.Context) {
	if h.Service.OAuthProviderConfigured(c.Param("auth_provider")) {
		if providerError := c.Query("error"); providerError != "" {
			if c.Param("auth_provider") == "oidc" {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication failed"})
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"message": c.Query("error_description") + ", read more at: " + c.Query("error_uri")})
			return
		}
		var input struct {
			CallbackURL string `json:"callback_url"`
		}
		if c.Request.Method != http.MethodGet {
			if err := c.ShouldBindJSON(&input); err != nil && err != io.EOF {
				c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
				return
			}
		}
		location, err := h.Service.StartOAuth(c, c.Param("auth_provider"), input.CallbackURL)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"location": location})
		return
	}
	if c.Param("auth_provider") != "emailpass" {
		c.JSON(400, gin.H{"error": "unknown authentication provider"})
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if c.Request.Method == http.MethodGet {
		input.Email = c.Query("email")
		input.Password = c.Query("password")
	} else if c.ShouldBindJSON(&input) != nil {
		c.JSON(400, gin.H{"error": "email and password are required"})
		return
	}
	result, err := h.Service.AuthenticateProvider(c, c.Param("auth_provider"), input.Email, input.Password, c.Param("actor_type"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return
	}
	if result.MFARequired && result.MFAChallenge != nil {
		c.JSON(http.StatusOK, gin.H{"mfa_required": true, "mfa_challenge": result.MFAChallenge, "token": result.Token})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": result.Token})
}

func (h *HTTP) oauthCallback(c *gin.Context) {
	if !h.Service.OAuthProviderConfigured(c.Param("auth_provider")) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "unknown authentication provider"})
		return
	}
	if providerError := c.Query("error"); providerError != "" {
		if c.Param("auth_provider") == "oidc" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication failed"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"message": c.Query("error_description") + ", read more at: " + c.Query("error_uri")})
		return
	}
	code, state := c.Query("code"), c.Query("state")
	params := map[string]string{}
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	if c.Request.Method == http.MethodPost {
		mediaType := c.ContentType()
		if strings.HasPrefix(mediaType, "application/x-www-form-urlencoded") || strings.HasPrefix(mediaType, "multipart/form-data") {
			if err := c.Request.ParseForm(); err == nil {
				for key, values := range c.Request.PostForm {
					if len(values) > 0 {
						params[key] = values[0]
					}
				}
			}
		} else {
			var body map[string]any
			if c.ShouldBindJSON(&body) == nil {
				for key, value := range body {
					if stringValue, ok := value.(string); ok {
						params[key] = stringValue
					}
				}
			}
		}
	}
	if code == "" {
		code = params["code"]
	}
	if state == "" {
		state = params["state"]
	}
	result, err := h.Service.CompleteOAuth(c, c.Param("auth_provider"), state, code, c.Param("actor_type"), params)
	if err != nil {
		message := err.Error()
		if c.Param("auth_provider") == "oidc" && message != "No code provided" && message != "No state provided, or session expired" {
			message = "Authentication failed"
		}
		c.JSON(http.StatusUnauthorized, gin.H{"message": message})
		return
	}
	if result.MFARequired && result.MFAChallenge != nil {
		c.JSON(http.StatusOK, gin.H{"mfa_required": true, "mfa_challenge": result.MFAChallenge, "token": result.Token})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": result.Token})
}
func writeError(c *gin.Context, err error, status int) { c.JSON(status, gin.H{"error": err.Error()}) }
