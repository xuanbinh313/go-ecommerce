package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	adminapi "github.com/example/medusa-go-clone/internal/api/admin"
	authhttp "github.com/example/medusa-go-clone/internal/api/auth"
	storeapi "github.com/example/medusa-go-clone/internal/api/store"
	authflows "github.com/example/medusa-go-clone/internal/coreflows/auth/workflows"
	customerflows "github.com/example/medusa-go-clone/internal/coreflows/customer/workflows"
	productflows "github.com/example/medusa-go-clone/internal/coreflows/product/workflows"
	regionflows "github.com/example/medusa-go-clone/internal/coreflows/region/workflows"
	saleschannelflows "github.com/example/medusa-go-clone/internal/coreflows/sales-channel/workflows"
	userflows "github.com/example/medusa-go-clone/internal/coreflows/user/workflows"
	"github.com/example/medusa-go-clone/internal/database"
	analyticsapp "github.com/example/medusa-go-clone/internal/modules/analytics/application"
	analyticsdomain "github.com/example/medusa-go-clone/internal/modules/analytics/domain"
	analyticslocal "github.com/example/medusa-go-clone/internal/modules/analytics/infrastructure/local"
	analyticsposthog "github.com/example/medusa-go-clone/internal/modules/analytics/infrastructure/posthog"
	apikeyapp "github.com/example/medusa-go-clone/internal/modules/api-key/application"
	apikeypostgres "github.com/example/medusa-go-clone/internal/modules/api-key/infrastructure/postgres"
	authapp "github.com/example/medusa-go-clone/internal/modules/auth/application"
	authmodels "github.com/example/medusa-go-clone/internal/modules/auth/domain/models"
	authgithub "github.com/example/medusa-go-clone/internal/modules/auth/infrastructure/github"
	authgoogle "github.com/example/medusa-go-clone/internal/modules/auth/infrastructure/google"
	authoidc "github.com/example/medusa-go-clone/internal/modules/auth/infrastructure/oidc"
	authpostgres "github.com/example/medusa-go-clone/internal/modules/auth/infrastructure/postgres"
	authsecurity "github.com/example/medusa-go-clone/internal/modules/auth/infrastructure/security"
	authtotp "github.com/example/medusa-go-clone/internal/modules/auth/infrastructure/totp"
	cachememoryapp "github.com/example/medusa-go-clone/internal/modules/cache-inmemory/application"
	cachememory "github.com/example/medusa-go-clone/internal/modules/cache-inmemory/infrastructure/memory"
	cartapp "github.com/example/medusa-go-clone/internal/modules/cart/application"
	cartpostgres "github.com/example/medusa-go-clone/internal/modules/cart/infrastructure/postgres"
	currencyapp "github.com/example/medusa-go-clone/internal/modules/currency/application"
	currencypostgres "github.com/example/medusa-go-clone/internal/modules/currency/infrastructure/postgres"
	customerapp "github.com/example/medusa-go-clone/internal/modules/customer/application"
	customermodels "github.com/example/medusa-go-clone/internal/modules/customer/domain/models"
	customerpostgres "github.com/example/medusa-go-clone/internal/modules/customer/infrastructure/postgres"
	eventbusapp "github.com/example/medusa-go-clone/internal/modules/event-bus-local/application"
	fileapp "github.com/example/medusa-go-clone/internal/modules/file/application"
	filedomain "github.com/example/medusa-go-clone/internal/modules/file/domain"
	filelocal "github.com/example/medusa-go-clone/internal/modules/file/infrastructure/local"
	files3 "github.com/example/medusa-go-clone/internal/modules/file/infrastructure/s3"
	fulfillmentapp "github.com/example/medusa-go-clone/internal/modules/fulfillment/application"
	manualfulfillment "github.com/example/medusa-go-clone/internal/modules/fulfillment/infrastructure/manual"
	fulfillmentpostgres "github.com/example/medusa-go-clone/internal/modules/fulfillment/infrastructure/postgres"
	inventoryapp "github.com/example/medusa-go-clone/internal/modules/inventory/application"
	inventorypostgres "github.com/example/medusa-go-clone/internal/modules/inventory/infrastructure/postgres"
	linksapp "github.com/example/medusa-go-clone/internal/modules/links/application"
	linkmodels "github.com/example/medusa-go-clone/internal/modules/links/domain/models"
	linkspostgres "github.com/example/medusa-go-clone/internal/modules/links/infrastructure/postgres"
	lockingmemory "github.com/example/medusa-go-clone/internal/modules/locking/infrastructure/memory"
	lockingpostgres "github.com/example/medusa-go-clone/internal/modules/locking/infrastructure/postgres"
	lockingredis "github.com/example/medusa-go-clone/internal/modules/locking/infrastructure/redis"
	notificationapp "github.com/example/medusa-go-clone/internal/modules/notification/application"
	notificationmodels "github.com/example/medusa-go-clone/internal/modules/notification/domain/models"
	notificationlocal "github.com/example/medusa-go-clone/internal/modules/notification/infrastructure/local"
	notificationpostgres "github.com/example/medusa-go-clone/internal/modules/notification/infrastructure/postgres"
	notificationsendgrid "github.com/example/medusa-go-clone/internal/modules/notification/infrastructure/sendgrid"
	orderapp "github.com/example/medusa-go-clone/internal/modules/order/application"
	orderpostgres "github.com/example/medusa-go-clone/internal/modules/order/infrastructure/postgres"
	paymentapp "github.com/example/medusa-go-clone/internal/modules/payment/application"
	paymentmodels "github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
	paymentpostgres "github.com/example/medusa-go-clone/internal/modules/payment/infrastructure/postgres"
	stripepayment "github.com/example/medusa-go-clone/internal/modules/payment/infrastructure/stripe"
	pricingapp "github.com/example/medusa-go-clone/internal/modules/pricing/application"
	pricingmodels "github.com/example/medusa-go-clone/internal/modules/pricing/domain/models"
	pricingpostgres "github.com/example/medusa-go-clone/internal/modules/pricing/infrastructure/postgres"
	productapp "github.com/example/medusa-go-clone/internal/modules/product/application"
	productmodels "github.com/example/medusa-go-clone/internal/modules/product/domain/models"
	productpostgres "github.com/example/medusa-go-clone/internal/modules/product/infrastructure/postgres"
	promotionapp "github.com/example/medusa-go-clone/internal/modules/promotion/application"
	promotionpostgres "github.com/example/medusa-go-clone/internal/modules/promotion/infrastructure/postgres"
	rbacapp "github.com/example/medusa-go-clone/internal/modules/rbac/application"
	rbacmodels "github.com/example/medusa-go-clone/internal/modules/rbac/domain/models"
	rbacpostgres "github.com/example/medusa-go-clone/internal/modules/rbac/infrastructure/postgres"
	regionapp "github.com/example/medusa-go-clone/internal/modules/region/application"
	regionmodels "github.com/example/medusa-go-clone/internal/modules/region/domain/models"
	regionpostgres "github.com/example/medusa-go-clone/internal/modules/region/infrastructure/postgres"
	saleschannelapp "github.com/example/medusa-go-clone/internal/modules/sales-channel/application"
	saleschannelmodels "github.com/example/medusa-go-clone/internal/modules/sales-channel/domain/models"
	saleschannelpostgres "github.com/example/medusa-go-clone/internal/modules/sales-channel/infrastructure/postgres"
	searchapp "github.com/example/medusa-go-clone/internal/modules/search/application"
	searchdomain "github.com/example/medusa-go-clone/internal/modules/search/domain"
	searchpostgres "github.com/example/medusa-go-clone/internal/modules/search/infrastructure/postgres"
	settingsapp "github.com/example/medusa-go-clone/internal/modules/settings/application"
	settingspostgres "github.com/example/medusa-go-clone/internal/modules/settings/infrastructure/postgres"
	stocklocationapp "github.com/example/medusa-go-clone/internal/modules/stock-location/application"
	stocklocationpostgres "github.com/example/medusa-go-clone/internal/modules/stock-location/infrastructure/postgres"
	storeapp "github.com/example/medusa-go-clone/internal/modules/store/application"
	storepostgres "github.com/example/medusa-go-clone/internal/modules/store/infrastructure/postgres"
	taxapp "github.com/example/medusa-go-clone/internal/modules/tax/application"
	taxpostgres "github.com/example/medusa-go-clone/internal/modules/tax/infrastructure/postgres"
	translationapp "github.com/example/medusa-go-clone/internal/modules/translation/application"
	translationpostgres "github.com/example/medusa-go-clone/internal/modules/translation/infrastructure/postgres"
	userapp "github.com/example/medusa-go-clone/internal/modules/user/application"
	usermodels "github.com/example/medusa-go-clone/internal/modules/user/domain/models"
	userpostgres "github.com/example/medusa-go-clone/internal/modules/user/infrastructure/postgres"
	usersecurity "github.com/example/medusa-go-clone/internal/modules/user/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	dbURL := env("DATABASE_URL", "postgres://root:root@localhost/my-db?sslmode=disable")
	db, err := database.Connect(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	events := eventbusapp.NewService(slog.Default())
	var analyticsProvider analyticsdomain.Provider = analyticslocal.New(slog.Default())
	analyticsProviderID := analyticslocal.Identifier
	switch strings.ToLower(env("ANALYTICS_PROVIDER", "local")) {
	case "local":
	case "posthog":
		analyticsProvider, err = analyticsposthog.New(env("POSTHOG_EVENTS_KEY", ""), env("POSTHOG_HOST", ""), nil, slog.Default())
		if err != nil {
			log.Fatal(err)
		}
		analyticsProviderID = analyticsposthog.Identifier
	default:
		log.Fatalf("unsupported ANALYTICS_PROVIDER %q (expected local or posthog)", env("ANALYTICS_PROVIDER", "local"))
	}
	analytics, err := analyticsapp.New(map[string]analyticsdomain.Provider{analyticsProviderID: analyticsProvider})
	if err != nil {
		log.Fatal(err)
	}
	defer analytics.Shutdown(context.Background())
	customerRepository := customerpostgres.NewRepository(db)
	customers := customerapp.NewService(customerRepository)
	apiKeys := apikeyapp.NewService(apikeypostgres.NewRepository(db))
	stockLocations := stocklocationapp.NewService(stocklocationpostgres.NewRepository(db))
	orders := orderapp.New(orderpostgres.NewRepository(db))
	paymentRepository := paymentpostgres.NewRepository(db)
	var paymentProviders *paymentapp.ProviderService
	if stripeAPIKey := env("STRIPE_API_KEY", ""); stripeAPIKey != "" {
		asyncMethods := []string{}
		for _, method := range strings.Split(env("STRIPE_ASYNC_PAYMENT_METHOD_TYPES", ""), ",") {
			if value := strings.TrimSpace(method); value != "" {
				asyncMethods = append(asyncMethods, value)
			}
		}
		registry, providerErr := stripepayment.NewRegistry(stripepayment.Options{APIKey: stripeAPIKey, WebhookSecret: env("STRIPE_WEBHOOK_SECRET", ""), Capture: strings.EqualFold(env("STRIPE_CAPTURE", "false"), "true"), AutomaticPaymentMethods: strings.EqualFold(env("STRIPE_AUTOMATIC_PAYMENT_METHODS", "false"), "true"), PaymentMethodConfiguration: env("STRIPE_PAYMENT_METHOD_CONFIGURATION", ""), PaymentDescription: env("STRIPE_PAYMENT_DESCRIPTION", ""), OxxoExpiresDays: positiveIntEnv("STRIPE_OXXO_EXPIRES_DAYS", 3), AsyncPaymentMethodTypes: asyncMethods})
		if providerErr != nil {
			log.Fatal(providerErr)
		}
		paymentProviders = paymentapp.NewProviderService(registry, slog.Default())
	}
	payments := paymentapp.NewWithProviderService(paymentRepository, paymentProviders)
	manualFulfillmentProviders, err := fulfillmentapp.NewProviderService(manualfulfillment.New())
	if err != nil {
		log.Fatal(err)
	}
	fulfillments := fulfillmentapp.NewWithProviders(fulfillmentpostgres.NewRepository(db), manualFulfillmentProviders)
	links := linksapp.NewService(linkspostgres.NewRepository(db))
	products := productapp.NewService(productpostgres.NewRepository(db))
	inventory := inventoryapp.NewService(inventorypostgres.NewRepository(db))
	var locks interface {
		Execute(context.Context, []string, time.Duration, func(context.Context) error) error
	} = lockingmemory.NewProvider()
	switch strings.ToLower(env("LOCKING_PROVIDER", "memory")) {
	case "memory":
	case "postgres":
		locks = lockingpostgres.NewProvider(db)
	case "redis":
		provider, providerErr := lockingredis.NewProvider(env("LOCKING_REDIS_URL", ""), env("LOCKING_REDIS_NAMESPACE", ""), lockingredis.Options{
			WaitLockingTimeout:   time.Duration(positiveIntEnv("LOCKING_WAIT_TIMEOUT_SECONDS", 5)) * time.Second,
			DefaultRetryInterval: time.Duration(positiveIntEnv("LOCKING_DEFAULT_RETRY_INTERVAL_MS", 20)) * time.Millisecond,
			MaximumRetryInterval: time.Duration(positiveIntEnv("LOCKING_MAXIMUM_RETRY_INTERVAL_MS", 1000)) * time.Millisecond,
			BackoffFactor:        positiveFloatEnv("LOCKING_BACKOFF_FACTOR", 2),
		}, slog.Default())
		if providerErr != nil {
			log.Fatal(providerErr)
		}
		locks = provider
		defer provider.Close()
	default:
		log.Fatalf("unsupported LOCKING_PROVIDER %q (expected memory, postgres, or redis)", env("LOCKING_PROVIDER", "memory"))
	}
	carts := cartapp.NewService(cartpostgres.NewRepository(db), regionpostgres.NewRepository(db), storepostgres.NewRepository(db), saleschannelpostgres.NewRepository(db))
	taxes := taxapp.NewService(taxpostgres.NewRepository(db))
	translations := translationapp.NewService(translationpostgres.NewRepository(db))
	searchService := configuredSearchService(db)
	storeSearchAllowlist := map[string]storeapi.StoreSearchIndexConfig{}
	if raw := strings.TrimSpace(os.Getenv("SEARCH_STORE_ALLOWLIST")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &storeSearchAllowlist); err != nil {
			log.Fatalf("invalid SEARCH_STORE_ALLOWLIST: %v", err)
		}
	}
	fileProviderID := strings.ToLower(env("FILE_PROVIDER", "local"))
	var fileProvider filedomain.Provider
	switch fileProviderID {
	case "local":
		fileProvider = filelocal.New(env("FILE_UPLOAD_DIR", "static"), env("FILE_PRIVATE_UPLOAD_DIR", env("FILE_UPLOAD_DIR", "static")), env("FILE_BACKEND_URL", "http://localhost:8080/static"))
	case "s3":
		config := files3.Config{FileURL: env("S3_FILE_URL", ""), AccessKeyID: env("S3_ACCESS_KEY_ID", ""), SecretAccessKey: env("S3_SECRET_ACCESS_KEY", ""), SessionToken: env("S3_SESSION_TOKEN", ""), AuthenticationMethod: env("S3_AUTHENTICATION_METHOD", "access-key"), Region: env("S3_REGION", ""), Bucket: env("S3_BUCKET", ""), Prefix: env("S3_PREFIX", ""), Endpoint: env("S3_ENDPOINT", ""), CacheControl: env("S3_CACHE_CONTROL", "public, max-age=31536000"), UsePathStyle: strings.EqualFold(env("S3_USE_PATH_STYLE", "false"), "true")}
		if rawDuration := os.Getenv("S3_DOWNLOAD_FILE_DURATION"); rawDuration != "" {
			seconds, parseErr := strconv.Atoi(rawDuration)
			if parseErr != nil || seconds <= 0 {
				log.Fatalf("S3_DOWNLOAD_FILE_DURATION must be a positive number of seconds")
			}
			config.DownloadFileDuration = time.Duration(seconds) * time.Second
		}
		if acl := os.Getenv("S3_ACL"); acl == "false" {
			config.DisableACL = true
		} else if acl != "" {
			config.ACL = &acl
		}
		fileProvider, err = files3.New(ctx, config, slog.Default())
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unsupported FILE_PROVIDER %q (expected local or s3)", fileProviderID)
	}
	fileService, err := fileapp.New(map[string]filedomain.Provider{fileProviderID: fileProvider})
	if err != nil {
		log.Fatal(err)
	}
	promotions := promotionapp.New(promotionpostgres.NewRepository(db))
	pricing := pricingapp.NewService(pricingpostgres.NewRepository(db))
	rbac := rbacapp.NewService(rbacpostgres.NewRepository(db))
	regions := regionapp.NewService(regionpostgres.NewRepository(db))
	salesChannels := saleschannelapp.NewService(saleschannelpostgres.NewRepository(db))
	stores := storeapp.NewService(storepostgres.NewRepository(db))
	currencies := currencyapp.NewService(currencypostgres.NewRepository(db))
	userRepository := userpostgres.NewRepository(db)
	users := userapp.NewService(userRepository, usersecurity.NewInviteTokenManager(env("JWT_SECRET", "development-secret-change-before-deploy")), 0)
	layouts := settingsapp.NewService(settingspostgres.NewRepository(db))
	var activeNotificationProvider interface {
		Identifier() string
		Send(context.Context, *notificationmodels.ProviderSendNotificationInput) (map[string]any, error)
	} = notificationlocal.New(slog.Default())
	switch strings.ToLower(env("NOTIFICATION_PROVIDER", "local")) {
	case "local":
	case "sendgrid":
		activeNotificationProvider = notificationsendgrid.New(env("SENDGRID_API_KEY", ""), env("SENDGRID_FROM", ""), nil)
	default:
		log.Fatalf("unsupported NOTIFICATION_PROVIDER %q (expected local or sendgrid)", env("NOTIFICATION_PROVIDER", "local"))
	}
	notificationProviders, err := notificationapp.NewProviderService(activeNotificationProvider)
	if err != nil {
		log.Fatal(err)
	}
	notifications := notificationapp.NewWithProviderService(notificationpostgres.NewRepository(db), notificationProviders)
	authRepository := authpostgres.NewRepository(db)
	authService := authapp.NewService(authRepository, authsecurity.NewPasswordHasher(), authsecurity.NewTokenManager(env("JWT_SECRET", "development-secret-change-before-deploy")))
	authService.ConfigureMFA(authtotp.New(env("AUTH_MFA_ENCRYPTION_KEY", ""), env("AUTH_MFA_ISSUER", "Medusa"), 6, 30, 1), env("AUTH_MFA_DISABLE_POLICY", "session"))
	authCache := cachememoryapp.New(cachememory.New(30))
	authService.ConfigureMFAChallenges(authCache, positiveIntEnv("AUTH_MFA_CHALLENGE_TTL_SECONDS", 300), positiveIntEnv("AUTH_MFA_CHALLENGE_MAX_ATTEMPTS", 5))
	githubClientID, githubClientSecret, githubCallbackURL := env("AUTH_GITHUB_CLIENT_ID", ""), env("AUTH_GITHUB_CLIENT_SECRET", ""), env("AUTH_GITHUB_CALLBACK_URL", "")
	if githubClientID != "" && githubClientSecret != "" && githubCallbackURL != "" {
		authService.ConfigureGitHubOAuth(authCache, authgithub.New(githubClientID, githubClientSecret), githubClientID, githubCallbackURL)
	}
	googleClientID, googleClientSecret, googleCallbackURL := env("AUTH_GOOGLE_CLIENT_ID", ""), env("AUTH_GOOGLE_CLIENT_SECRET", ""), env("AUTH_GOOGLE_CALLBACK_URL", "")
	if googleClientID != "" && googleClientSecret != "" && googleCallbackURL != "" {
		authService.ConfigureOAuthProvider(authCache, "google", authgoogle.New(googleClientID, googleClientSecret), googleClientID, googleCallbackURL)
	}
	if oidcConfig := strings.TrimSpace(env("AUTH_OIDC_CONFIG", "")); oidcConfig != "" {
		var options authoidc.Options
		if err := json.Unmarshal([]byte(oidcConfig), &options); err != nil {
			log.Fatalf("invalid AUTH_OIDC_CONFIG: %v", err)
		}
		provider, err := authoidc.New(options, strings.EqualFold(env("NODE_ENV", "development"), "production"))
		if err != nil {
			log.Fatalf("invalid AUTH_OIDC_CONFIG: %v", err)
		}
		authService.ConfigureOAuthProvider(authCache, "oidc", provider, options.ClientID, options.CallbackURL)
	}
	accountWorkflow := userAccountWorkflow{users: users, repository: userRepository, auth: authService, links: links, events: events, rbac: rbac}
	authHTTP := &authhttp.HTTP{Service: authService, PasswordReset: passwordResetWorkflow{auth: authService, events: events}, Verification: authVerificationWorkflow{auth: authService, events: events}, Users: users, UserAccount: accountWorkflow, Events: events}
	gin.SetMode(env("GIN_MODE", gin.ReleaseMode))
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), cors(env("ADMIN_CORS", "http://localhost:7001")+","+env("STORE_CORS", "http://localhost:8000")))
	router.Static("/static", env("FILE_UPLOAD_DIR", "static"))
	rbacEnabled := strings.EqualFold(env("MEDUSA_FF_RBAC", "false"), "true")
	propertyLabelsEnabled := strings.EqualFold(env("MEDUSA_FF_VIEW_CONFIGURATIONS", "false"), "true")
	registerAPI := func(api *gin.RouterGroup) {
		authHTTP.Register(api)
		translationEnabled := strings.EqualFold(env("MEDUSA_FF_TRANSLATION", "false"), "true")
		adminapi.RegisterStartupRoutes(api, map[string]bool{"translation": translationEnabled, "rbac": rbacEnabled, "view_configurations": propertyLabelsEnabled})
		(&adminapi.ModuleHTTP{Customers: customers, CustomerWorkflow: customerWorkflow{service: customers, auth: authService, events: events}, Products: products, ProductCreator: productCreator{products: products, links: links, events: events}, ProductCatalog: productCreator{products: products, events: events}, ProductOptionCreator: productCreator{products: products, events: events}, ProductCategories: products, PropertyLabels: layouts, PropertyLabelsEnabled: propertyLabelsEnabled, Regions: regions, RegionWorkflow: regionWorkflow{service: regions, links: links, payments: payments, prices: pricing, events: events}, SalesChannels: salesChannels, SalesChannelWorkflow: salesChannelWorkflow{service: salesChannels, stores: stores, links: links, events: events}, Stores: stores, Currencies: currencies, Users: users, UserWorkflow: userWorkflow{service: users, events: events}, UserRolesWorkflow: userRolesWorkflow{rbac: rbac, links: links, users: users}, UserAccountWorkflow: accountWorkflow, UserAuthProviders: authService, UserResetToken: userResetTokenWorkflow{auth: authService, events: events}, Policies: rbac, RBACEnabled: rbacEnabled, Layouts: layouts, Notifications: notifications, APIKeys: apiKeys}).Register(api, authHTTP.AdminMiddleware())
		inviteHTTP := &adminapi.InviteHTTP{Service: users, Workflow: inviteWorkflow{users: users, repository: userRepository, rbac: rbac, auth: authService, links: links, events: events}}
		inviteHTTP.Register(api, authHTTP.AdminMiddleware())
		inviteHTTP.RegisterAccept(api, authHTTP.AuthMiddleware(true))
		(&adminapi.StockLocationHTTP{Service: stockLocations}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.OrderHTTP{Service: orders, Customers: customers, Products: products, Translations: translations, TranslationEnabled: translationEnabled, Events: events, Permissions: userRolesWorkflow{rbac: rbac, links: links, users: users}, RBACEnabled: rbacEnabled}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.CollectionHTTP{Service: products, Events: events}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.PaymentCollectionHTTP{Orders: orders, Payments: payments, Links: links}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.PaymentHTTP{Service: payments}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.SearchIndexHTTP{Service: searchService}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.FulfillmentHTTP{Service: fulfillments, Locations: stockLocations, Links: links, Permissions: userRolesWorkflow{rbac: rbac, links: links, users: users}, RBACEnabled: rbacEnabled, Events: events}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.InventoryHTTP{Service: inventory, Links: productLinkAdapter{service: links}, Products: products, Events: events, Locks: locks, Permissions: userRolesWorkflow{rbac: rbac, links: links, users: users}, RBACEnabled: rbacEnabled}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.PricingHTTP{Service: pricing}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.TaxHTTP{Service: taxes, Permissions: userRolesWorkflow{rbac: rbac, links: links, users: users}, RBACEnabled: rbacEnabled}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.TranslationHTTP{Service: translations, Enabled: translationEnabled, Permissions: userRolesWorkflow{rbac: rbac, links: links, users: users}, RBACEnabled: rbacEnabled}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.FileHTTP{Service: fileService}).Register(api, authHTTP.AdminMiddleware())
		(&adminapi.PromotionHTTP{Service: promotions, RuleOptions: promotionRuleOptions{regions: regions, currencies: currencies, customers: customers, salesChannels: salesChannels, products: products}, Permissions: userRolesWorkflow{rbac: rbac, links: links, users: users}, RBACEnabled: rbacEnabled}).Register(api, authHTTP.AdminMiddleware())
		(&storeapi.CustomerHTTP{Customers: customers, Auth: authService, Authenticate: authHTTP.AuthMiddleware(true), RequireActor: authHTTP.AuthMiddleware(false)}).Register(api)
		(&storeapi.CartHTTP{Service: carts, Customers: customers, Events: events, Authenticate: authHTTP.AuthMiddleware(true)}).Register(api)
		(&storeapi.CollectionHTTP{Service: products}).Register(api)
		(&storeapi.ProductCatalogHTTP{Service: products}).Register(api)
		(&storeapi.StoreProductCategoryHTTP{Service: products}).Register(api)
		(&storeapi.StoreDiscoveryHTTP{Currencies: currencies, Stores: stores, Locales: translations}).Register(api, translationEnabled)
		(&storeapi.RegionHTTP{Service: regions, Links: links, Payments: payments}).Register(api)
		(&storeapi.StoreProductOptionHTTP{Service: products}).Register(api)
		(&storeapi.StoreProductVariantHTTP{Service: products}).Register(api)
		(&storeapi.PaymentProviderHTTP{Links: links, Payments: payments}).Register(api)
		(&storeapi.OrderHTTP{Service: orders, RequireActor: authHTTP.AuthMiddleware(false)}).Register(api)
		(&storeapi.ReturnReasonHTTP{Service: orders}).Register(api)
		(&storeapi.SearchHTTP{Service: searchService, APIKeys: apiKeys, Allowlist: storeSearchAllowlist}).Register(api, authHTTP.AuthMiddleware(true))
	}
	registerAPI(router.Group(""))
	registerAPI(router.Group("/api/v1"))
	addr := env("HTTP_ADDR", ":8080")
	if err := router.Run(addr); err != nil {
		log.Fatal(err)
	}
}

// configuredSearchService reads the JSON form of Medusa's Search module
// options. Empty configuration leaves the optional module disabled.
func configuredSearchService(db *pgxpool.Pool) *searchapp.Service {
	raw := strings.TrimSpace(os.Getenv("SEARCH_MODULE_OPTIONS"))
	if raw == "" {
		return nil
	}
	var options searchapp.ModuleOptions
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		log.Fatalf("invalid SEARCH_MODULE_OPTIONS: %v", err)
	}
	var language string
	var searchProviders []searchdomain.SearchProvider
	for _, configured := range options.Providers {
		if !strings.Contains(configured.Resolve, "search-postgres") {
			log.Fatalf("unsupported Search provider %q (only search-postgres is implemented)", configured.Resolve)
		}
		if value, ok := configured.Options["language"].(string); ok {
			language = value
		}
		if engine, ok := configured.Options["engine"].(string); ok && engine != "native" {
			log.Fatalf("unsupported search-postgres engine %q", engine)
		}
		provider, err := searchpostgres.NewProvider(db, language)
		if err != nil {
			log.Fatal(err)
		}
		searchProviders = append(searchProviders, provider)
	}
	providers, err := searchapp.NewProviderRegistry(searchProviders...)
	if err != nil {
		log.Fatal(err)
	}
	definitions, err := searchapp.ResolveIndexDefinitions(options, providers)
	if err != nil {
		log.Fatalf("invalid SEARCH_MODULE_OPTIONS indexes: %v", err)
	}
	repository := searchpostgres.NewRepository(db)
	resolver := searchapp.NewConfiguredIndexResolver(repository, definitions)
	return searchapp.NewService(providers, resolver, repository)
}

type promotionRuleOptions struct {
	regions       *regionapp.Service
	currencies    *currencyapp.Service
	customers     *customerapp.Service
	salesChannels *saleschannelapp.Service
	products      *productapp.Service
}

func (r promotionRuleOptions) ListPromotionRuleOptionValues(ctx context.Context, attribute string) ([]adminapi.PromotionRuleOption, error) {
	values := []adminapi.PromotionRuleOption{}
	switch attribute {
	case "region":
		rows, err := r.regions.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, x := range rows {
			values = append(values, adminapi.PromotionRuleOption{Label: x.Name, Value: x.ID})
		}
	case "country":
		rows, err := r.regions.List(ctx)
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, x := range rows {
			for _, c := range x.Countries {
				if !seen[c.ISO2] {
					seen[c.ISO2] = true
					values = append(values, adminapi.PromotionRuleOption{Label: c.DisplayName, Value: c.ISO2})
				}
			}
		}
	case "currency_code":
		rows, err := r.currencies.ListCurrencies(ctx, nil)
		if err != nil {
			return nil, err
		}
		for _, x := range rows {
			values = append(values, adminapi.PromotionRuleOption{Label: x.Name, Value: x.Code})
		}
	case "customer_group":
		rows, err := r.customers.ListCustomerGroups(ctx)
		if err != nil {
			return nil, err
		}
		for _, x := range rows {
			values = append(values, adminapi.PromotionRuleOption{Label: x.Name, Value: x.ID})
		}
	case "sales_channel":
		rows, err := r.salesChannels.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, x := range rows {
			values = append(values, adminapi.PromotionRuleOption{Label: x.Name, Value: x.ID})
		}
	case "product_type":
		rows, _, err := r.products.ListProductTypes(ctx, nil, 10000, 0)
		if err != nil {
			return nil, err
		}
		for _, x := range rows {
			values = append(values, adminapi.PromotionRuleOption{Label: x.Value, Value: x.ID})
		}
	case "product_tag":
		rows, _, err := r.products.ListProductTags(ctx, nil, 10000, 0)
		if err != nil {
			return nil, err
		}
		for _, x := range rows {
			values = append(values, adminapi.PromotionRuleOption{Label: x.Value, Value: x.ID})
		}
	case "product":
		rows, _, err := r.products.ListAdminProducts(ctx, nil, nil, 10000, 0)
		if err != nil {
			return nil, err
		}
		for _, x := range rows {
			values = append(values, adminapi.PromotionRuleOption{Label: x.Title, Value: x.ID})
		}
	default:
		return nil, fmt.Errorf("promotion rule value options for %q require an unported Medusa query entry point", attribute)
	}
	return values, nil
}

type productCreator struct {
	products *productapp.Service
	links    *linksapp.Service
	events   *eventbusapp.Service
}

type salesChannelWorkflow struct {
	service *saleschannelapp.Service
	stores  *storeapp.Service
	links   *linksapp.Service
	events  *eventbusapp.Service
}

type regionWorkflow struct {
	service  *regionapp.Service
	links    *linksapp.Service
	payments *paymentapp.Service
	prices   *pricingapp.Service
	events   *eventbusapp.Service
}

type userWorkflow struct {
	service *userapp.Service
	events  *eventbusapp.Service
}

type customerWorkflow struct {
	service *customerapp.Service
	auth    *authapp.Service
	events  *eventbusapp.Service
}

func (w customerWorkflow) Create(ctx context.Context, input customermodels.Input) (customermodels.Customer, error) {
	return customerflows.CreateCustomers(ctx, w.service, w.events, input)
}

func (w customerWorkflow) Update(ctx context.Context, id string, input customermodels.Input) (customermodels.Customer, error) {
	return customerflows.UpdateCustomers(ctx, w.service, w.events, id, input)
}

func (w customerWorkflow) DeleteCustomer(ctx context.Context, id string) error {
	return customerflows.RemoveCustomerAccount(ctx, w.service, w.auth, w.events, id)
}

func (w customerWorkflow) CreateAddress(ctx context.Context, id string, input customermodels.AddressInput) (customermodels.Address, error) {
	return w.service.CreateAddress(ctx, id, input)
}

func (w customerWorkflow) ListAddresses(ctx context.Context, id string, limit, offset int) ([]customermodels.Address, int64, error) {
	return w.service.ListAddresses(ctx, id, limit, offset)
}

func (w customerWorkflow) GetAddress(ctx context.Context, id, addressID string) (customermodels.Address, error) {
	return w.service.GetAddress(ctx, id, addressID)
}

func (w customerWorkflow) UpdateAddress(ctx context.Context, id, addressID string, input customermodels.AddressUpdate) (customermodels.Customer, error) {
	return w.service.UpdateAddress(ctx, id, addressID, input)
}

func (w customerWorkflow) DeleteAddress(ctx context.Context, id, addressID string) error {
	return w.service.DeleteAddress(ctx, id, addressID)
}

func (w customerWorkflow) LinkCustomerGroups(ctx context.Context, customerID string, add, remove []string, createdBy *string) error {
	return customerflows.LinkCustomerGroups(ctx, w.service, customerID, add, remove, createdBy)
}

func (w customerWorkflow) ListCustomerGroups(ctx context.Context) ([]customermodels.CustomerGroup, error) {
	return w.service.ListCustomerGroups(ctx)
}

func (w customerWorkflow) GetCustomerGroup(ctx context.Context, id string) (customermodels.CustomerGroup, error) {
	return w.service.GetCustomerGroup(ctx, id)
}

func (w customerWorkflow) CreateCustomerGroup(ctx context.Context, name string, metadata json.RawMessage, createdBy *string) (customermodels.CustomerGroup, error) {
	return customerflows.CreateCustomerGroup(ctx, w.service, name, metadata, createdBy)
}

func (w customerWorkflow) UpdateCustomerGroup(ctx context.Context, id string, input customermodels.CustomerGroupUpdate) (customermodels.CustomerGroup, error) {
	return customerflows.UpdateCustomerGroup(ctx, w.service, id, input)
}

func (w customerWorkflow) DeleteCustomerGroup(ctx context.Context, id string) error {
	return customerflows.DeleteCustomerGroup(ctx, w.service, id)
}

func (w customerWorkflow) LinkGroupCustomers(ctx context.Context, id string, add, remove []string, createdBy *string) error {
	return customerflows.LinkGroupCustomers(ctx, w.service, id, add, remove, createdBy)
}

type userRolesWorkflow struct {
	rbac  *rbacapp.Service
	links *linksapp.Service
	users *userapp.Service
}

type userAccountWorkflow struct {
	users      *userapp.Service
	repository *userpostgres.Repository
	auth       *authapp.Service
	links      *linksapp.Service
	events     *eventbusapp.Service
	rbac       *rbacapp.Service
}

type userResetTokenWorkflow struct {
	auth   *authapp.Service
	events *eventbusapp.Service
}

type passwordResetWorkflow struct {
	auth   *authapp.Service
	events *eventbusapp.Service
}

type authVerificationWorkflow struct {
	auth   *authapp.Service
	events *eventbusapp.Service
}

func (w passwordResetWorkflow) Generate(ctx context.Context, identifier, actorType, provider string, metadata map[string]any) error {
	_, err := authflows.GenerateResetPasswordToken(ctx, w.auth, w.events, identifier, actorType, provider, metadata)
	return err
}

func (w authVerificationWorkflow) Request(ctx context.Context, input authmodels.RequestAuthVerificationInput, ttl time.Duration) (authmodels.AuthVerification, error) {
	return authflows.RequestAuthVerification(ctx, w.auth, w.events, input, ttl)
}

type inviteWorkflow struct {
	users      *userapp.Service
	repository *userpostgres.Repository
	rbac       *rbacapp.Service
	auth       *authapp.Service
	links      *linksapp.Service
	events     *eventbusapp.Service
}

func (w inviteWorkflow) CreateInvite(ctx context.Context, input userflows.CreateInviteInput) (usermodels.Invite, error) {
	rows, err := userflows.CreateInvites(ctx, w.users, w.rbac, productLinkAdapter{service: w.links}, w.events, []userflows.CreateInviteInput{input})
	if err != nil {
		return usermodels.Invite{}, err
	}
	return rows[0], nil
}
func (w inviteWorkflow) DeleteInvite(ctx context.Context, id string) error {
	return userflows.DeleteInvite(ctx, w.users, w.events, id)
}
func (w inviteWorkflow) RefreshInviteToken(ctx context.Context, id string) (usermodels.Invite, error) {
	return userflows.RefreshInviteTokens(ctx, w.users, w.events, id)
}
func (w inviteWorkflow) AcceptInvite(ctx context.Context, token, authIdentityID string, email, firstName, lastName *string) (usermodels.User, error) {
	return userflows.AcceptInvite(ctx, w.users, w.repository, w.rbac, w.auth, productLinkAdapter{service: w.links}, w.events, userflows.AcceptInviteInput{InviteToken: token, AuthIdentityID: authIdentityID, Email: email, FirstName: firstName, LastName: lastName})
}

func (w userResetTokenWorkflow) GenerateUserResetToken(ctx context.Context, email string) (string, error) {
	return authflows.GenerateResetPasswordToken(ctx, w.auth, w.events, email, "user", "emailpass", map[string]any{})
}

func (w userAccountWorkflow) RemoveUserAccount(ctx context.Context, id string) error {
	return userflows.RemoveUserAccount(ctx, w.users, w.auth, productLinkAdapter{service: w.links}, w.events, id)
}

func (w userAccountWorkflow) CreateUserAccount(ctx context.Context, identityID string, input usermodels.UserInput) (usermodels.User, error) {
	return userflows.CreateUserAccount(ctx, w.repository, w.rbac, productLinkAdapter{service: w.links}, w.auth, w.events, userflows.CreateUserAccountInput{AuthIdentityID: identityID, UserData: input})
}

func (w userAccountWorkflow) SetAuthAppMetadata(ctx context.Context, identityID, actorType, actorID string) error {
	return authflows.SetAuthAppMetadataStep(ctx, w.auth, identityID, actorType, actorID)
}

func (w userRolesWorkflow) ListUserRoles(ctx context.Context, userID string) ([]rbacmodels.Role, error) {
	return userflows.UserRoles(ctx, w.rbac, productLinkAdapter{service: w.links}, userID)
}
func (w userRolesWorkflow) ListAssignableRoles(ctx context.Context, actorID, actorType, query string, ids, names []string, limit, offset int) ([]rbacmodels.Role, int, error) {
	return userflows.ListAssignableRoles(ctx, w.rbac, productLinkAdapter{service: w.links}, actorID, actorType, query, ids, names, limit, offset)
}
func (w userRolesWorkflow) ListAssignablePolicies(ctx context.Context, actorID, actorType, query string, ids, keys, resources, operations []string, limit, offset int) ([]rbacmodels.Policy, int, error) {
	return userflows.ListAssignablePolicies(ctx, w.rbac, productLinkAdapter{service: w.links}, actorID, actorType, query, ids, keys, resources, operations, limit, offset)
}
func (w userRolesWorkflow) AssignUserRoles(ctx context.Context, actorID, actorType, userID string, roleIDs []string) error {
	return userflows.AssignUserRoles(ctx, w.rbac, productLinkAdapter{service: w.links}, actorID, actorType, userID, roleIDs)
}
func (w userRolesWorkflow) RemoveUserRoles(ctx context.Context, actorID, actorType, userID string, roleIDs []string) error {
	return userflows.RemoveUserRoles(ctx, w.rbac, productLinkAdapter{service: w.links}, actorID, actorType, userID, roleIDs)
}
func (w userRolesWorkflow) ValidateUserPermissions(ctx context.Context, actorID, actorType string, policyIDs []string) error {
	return userflows.ValidateUserPermissions(ctx, w.rbac, productLinkAdapter{service: w.links}, actorID, actorType, policyIDs)
}
func (w userRolesWorkflow) HasPermissions(ctx context.Context, actorID, actorType string, actions []rbacmodels.PermissionAction) (bool, error) {
	return userflows.HasUserPermissions(ctx, w.rbac, productLinkAdapter{service: w.links}, actorID, actorType, actions)
}
func (w userRolesWorkflow) ListRoleUsers(ctx context.Context, roleID string, userIDs []string, limit, offset int) ([]usermodels.User, int, error) {
	return userflows.UsersForRole(ctx, w.users, productLinkAdapter{service: w.links}, roleID, userIDs, limit, offset)
}
func (w userRolesWorkflow) AssignUsersToRole(ctx context.Context, actorID, actorType, roleID string, userIDs []string) error {
	return userflows.AssignUsersToRole(ctx, w.rbac, productLinkAdapter{service: w.links}, actorID, actorType, roleID, userIDs)
}
func (w userRolesWorkflow) RemoveUsersFromRole(ctx context.Context, actorID, actorType, roleID string, userIDs []string) error {
	return userflows.RemoveUsersFromRole(ctx, w.rbac, productLinkAdapter{service: w.links}, actorID, actorType, roleID, userIDs)
}

func (w userWorkflow) UpdateUser(ctx context.Context, id string, input usermodels.UserInput) (usermodels.User, error) {
	return userflows.UpdateUser(ctx, w.service, w.events, id, input)
}

func (w regionWorkflow) ValidatePaymentProviders(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	enabled := true
	rows, _, err := w.payments.ListPaymentProviders(ctx, paymentmodels.PaymentProviderFilter{IDs: ids, IsEnabled: &enabled, Limit: len(ids)})
	if err != nil {
		return err
	}
	if len(rows) != len(ids) {
		return fmt.Errorf("one or more payment providers do not exist")
	}
	return nil
}
func (w regionWorkflow) CreateRegionPricePreferences(ctx context.Context, rows []regionflows.RegionPricePreference) error {
	for _, row := range rows {
		if _, err := w.prices.CreatePreference(ctx, pricingmodels.PreferenceInput{Attribute: row.Attribute, Value: row.Value, IsTaxInclusive: row.IsTaxInclusive}); err != nil {
			return err
		}
	}
	return nil
}
func (w regionWorkflow) DeleteRegionPricePreferences(ctx context.Context, ids []string) error {
	rows, _, err := w.prices.ListPreferences(ctx, []string{"region_id"}, ids, 1000, 0)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err = w.prices.DeletePreference(ctx, row.ID); err != nil {
			return err
		}
	}
	return nil
}
func (w regionWorkflow) CreateRegion(ctx context.Context, input regionmodels.RegionInput) (regionmodels.Region, error) {
	rows, err := regionflows.CreateRegions(ctx, w.service, productLinkAdapter{service: w.links}, w, w, w.events, []regionmodels.RegionInput{input})
	if err != nil {
		return regionmodels.Region{}, err
	}
	return rows[0], nil
}
func (w regionWorkflow) SetRegionTaxInclusive(ctx context.Context, id string, value bool) error {
	rows, _, err := w.prices.ListPreferences(ctx, []string{"region_id"}, []string{id}, 100, 0)
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err = w.prices.UpdatePreference(ctx, row.ID, pricingmodels.PreferenceUpdate{IsTaxInclusive: &value})
		if err != nil {
			return err
		}
	}
	return nil
}
func (w regionWorkflow) UpdateRegion(ctx context.Context, id string, input regionmodels.RegionInput) (regionmodels.Region, error) {
	return regionflows.UpdateRegion(ctx, w.service, productLinkAdapter{service: w.links}, w, w, w.events, id, input)
}
func (w regionWorkflow) DeleteRegion(ctx context.Context, id string) error {
	return regionflows.DeleteRegion(ctx, w.service, productLinkAdapter{service: w.links}, w.events, id)
}

func (w salesChannelWorkflow) CreateSalesChannel(ctx context.Context, input saleschannelmodels.Input) (saleschannelmodels.SalesChannel, error) {
	rows, err := saleschannelflows.CreateSalesChannels(ctx, w.service, w.events, []saleschannelmodels.Input{input})
	if err != nil {
		return saleschannelmodels.SalesChannel{}, err
	}
	return rows[0], nil
}

func (w salesChannelWorkflow) UpdateSalesChannel(ctx context.Context, id string, input saleschannelmodels.Input) (saleschannelmodels.SalesChannel, error) {
	return saleschannelflows.UpdateSalesChannel(ctx, w.service, w.events, id, input)
}

func (w salesChannelWorkflow) DeleteSalesChannel(ctx context.Context, id string) error {
	return saleschannelflows.DeleteSalesChannels(ctx, w.service, w.stores, productLinkAdapter{service: w.links}, w.events, []string{id})
}

func (w salesChannelWorkflow) LinkProducts(ctx context.Context, id string, add, remove []string) error {
	return saleschannelflows.LinkProductsToSalesChannel(ctx, productLinkAdapter{service: w.links}, w.events, id, add, remove)
}

func (c productCreator) CreateProducts(ctx context.Context, input productmodels.CreateProductsInput) ([]productmodels.Product, error) {
	return productflows.CreateProducts(ctx, c.products, productLinkAdapter{service: c.links}, productVariantCreator{products: c.products}, c.events, input)
}

func (c productCreator) DeleteProduct(ctx context.Context, id string) error {
	return productflows.DeleteProducts(ctx, c.products, productLinkAdapter{service: c.links}, c.events, []string{id})
}

func (c productCreator) UpdateProduct(ctx context.Context, id string, input map[string]any) error {
	return productflows.UpdateProduct(ctx, c.products, productLinkAdapter{service: c.links}, c.events, id, input)
}

func (c productCreator) CreateProductTypes(ctx context.Context, input []productmodels.ProductTypeInput) ([]productmodels.ProductType, error) {
	return productflows.CreateProductTypes(ctx, c.products, c.events, input)
}
func (c productCreator) UpdateProductType(ctx context.Context, id string, input map[string]any) (productmodels.ProductType, error) {
	return productflows.UpdateProductType(ctx, c.products, c.events, id, input)
}
func (c productCreator) DeleteProductTypes(ctx context.Context, ids []string) error {
	return productflows.DeleteProductTypes(ctx, c.products, c.events, ids)
}
func (c productCreator) CreateProductTags(ctx context.Context, input []productmodels.ProductTagInput) ([]productmodels.ProductTag, error) {
	return productflows.CreateProductTags(ctx, c.products, c.events, input)
}

func (c productCreator) CreateProductOptions(ctx context.Context, input []productmodels.CreateProductOptionInput) ([]productmodels.ProductOption, error) {
	return productflows.CreateProductOptions(ctx, c.products, c.events, input)
}

func (c productCreator) UpdateProductOption(ctx context.Context, id string, input map[string]any) (productmodels.ProductOption, error) {
	return productflows.UpdateProductOption(ctx, c.products, c.events, id, input)
}

func (c productCreator) DeleteProductOptions(ctx context.Context, ids []string) error {
	return productflows.DeleteProductOptions(ctx, c.products, c.events, ids)
}
func (c productCreator) UpdateProductTag(ctx context.Context, id string, input map[string]any) (productmodels.ProductTag, error) {
	return productflows.UpdateProductTag(ctx, c.products, c.events, id, input)
}
func (c productCreator) DeleteProductTags(ctx context.Context, ids []string) error {
	return productflows.DeleteProductTags(ctx, c.products, c.events, ids)
}

type productLinkAdapter struct{ service *linksapp.Service }

func (l productLinkAdapter) Create(ctx context.Context, definition linkmodels.Definition, rows []map[string]any) ([]linkmodels.Link, error) {
	return l.service.Create(ctx, definition.Name, rows)
}

func (l productLinkAdapter) Update(ctx context.Context, definition linkmodels.Definition, filters, data map[string]any) ([]linkmodels.Link, error) {
	return l.service.Update(ctx, definition.Name, filters, data)
}

func (l productLinkAdapter) Dismiss(ctx context.Context, definition linkmodels.Definition, rows []map[string]any) ([]linkmodels.Link, error) {
	return l.service.Dismiss(ctx, definition.Name, rows)
}

func (l productLinkAdapter) List(ctx context.Context, definition linkmodels.Definition, filters map[string]any) ([]linkmodels.Link, error) {
	return l.service.List(ctx, definition.Name, filters)
}

func (l productLinkAdapter) Delete(ctx context.Context, definition linkmodels.Definition, rows []map[string]any) error {
	return l.service.Delete(ctx, definition.Name, rows)
}

func (l productLinkAdapter) Restore(ctx context.Context, definition linkmodels.Definition, rows []map[string]any) ([]linkmodels.Link, error) {
	return l.service.Restore(ctx, definition.Name, rows)
}

type productVariantCreator struct{ products *productapp.Service }

func (v productVariantCreator) CreateProductVariants(ctx context.Context, input []productmodels.ProductVariantInput) ([]productmodels.ProductVariant, error) {
	return v.products.CreateVariants(ctx, input)
}

func cors(allowedOrigins string) gin.HandlerFunc {
	origins := make(map[string]struct{})
	for _, origin := range strings.Split(allowedOrigins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins[origin] = struct{}{}
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := origins[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			requestedHeaders := c.GetHeader("Access-Control-Request-Headers")
			if requestedHeaders != "" {
				c.Header("Access-Control-Allow-Headers", requestedHeaders)
			} else {
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Publishable-Api-Key, X-Medusa-Locale, X-Requested-With")
			}
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func positiveIntEnv(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		log.Fatalf("%s must be a positive integer", name)
	}
	return value
}

func positiveFloatEnv(name string, fallback float64) float64 {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 {
		log.Fatalf("%s must be a positive number", name)
	}
	return value
}
