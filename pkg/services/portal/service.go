package portal

import (
	"crypto/sha1"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	cache "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/cache"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	runtime "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/runtime"
	server "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/server"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"

	mail "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/mail"
	config "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/config"
	jwt "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/jwt"
	apiKey "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/key"
	authMw "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/middleware/auth"
	secure "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/secure"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/annotation"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/auth"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/content"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/dataset"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/experimental"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/export"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/model"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/password"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/prediction"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/project"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/tag"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/api/user"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/swaggerui"
)

// Start starts the Portal service
func Start(cfg *config.Configuration) error {
	// Initialize DB
	db, err := db.New(cfg.DB.URL, cfg.DB.Timeout)
	if err != nil {
		return err
	}
	defer db.Shutdown()

	// Initialize Cache
	cache, err := cache.Initialize(cfg.Cache)
	if err != nil {
		return err
	}
	defer cache.Shutdown()

	// Initialize SendGrid Mail
	sgAPIKey, err := runtime.GetEnv("SENDGRID_API_KEY")
	if err != nil {
		return err
	}
	mail := mail.NewPortalSGMailService(
		sgAPIKey,
		cfg.App.SendGridEmailConfirmTemplateID,
		cfg.App.SendGridPasswordResetTemplateID,
		cfg.App.PassResetCodeExpiration,
		cfg.App.PassResetEmail,
		cfg.App.PassResetSubject,
	)

	// Initialize Platform layer
	platform := platform.NewPlatform()

	// Initialize Blob store
	blob, err := blob.NewBlob(cfg.BlobStore)
	if err != nil {
		return err
	}
	// Kick off blob GC
	go blob.GC()

	// Initialize Securer Service
	sec := secure.New(cfg.App.MinPasswordStr, sha1.New())
	tokenSvc, err := jwt.New(
		cfg.JWT.SigningAlgorithm,
		os.Getenv("JWT_SECRET"),
		cfg.JWT.AccessDurationMinutes,
		cfg.JWT.RefreshDurationMinutes,
		cfg.JWT.AutoLogOffDurationMinutes,
		cfg.JWT.MinSecretLength,
		cache, true, platform, db)
	if err != nil {
		return err
	}

	// Initialize API Key Service
	// TODO - update with config settings
	apiKeySvc, err := apiKey.New(db, platform, []string{"inference/realtime"})
	if err != nil {
		return err
	}

	// Initialize Logger
	err = log.New(*cfg.Logger, log.InstanceZapLogger)
	if err != nil {
		return errors.Wrap(err, "could not instantiate logger")
	}

	// Initialize HTTP Server
	echoServer := server.New(server.Config{
		Port:              cfg.Server.Port,
		TimeoutSeconds:    cfg.Server.Timeout,
		Domain:            cfg.Server.Domain,
		CertCacheS3Bucket: cfg.Server.CertCacheS3Bucket,
		CrtFile:           cfg.Server.CrtFile,
		KeyFile:           cfg.Server.KeyFile,
		Debug:             cfg.Server.Debug,
	})

	enforcer, err := authMw.NewEnforcer(cfg.Server.RBACModelFile, cfg.Server.RBACPolicyFile, cfg.DB.URL)
	if err != nil {
		return errors.Wrap(err, "could not instantiate casbin enforcer")
	}

	jwtMW := authMw.JWT(os.Getenv("JWT_SECRET"), apiKeySvc)
	authMW := authMw.Middleware(os.Getenv("JWT_SECRET"), tokenSvc, apiKeySvc, enforcer)
	auth.NewHTTP(auth.Initialize(db, cache, platform, tokenSvc, mail, sec), echoServer.Echo, jwtMW, authMW)

	v1 := echoServer.Group("/v1")
	v1.Use(jwtMW, authMW)

	userSvc := user.Initialize(db, platform, sec, enforcer, mail, cache)
	passwordSvc := password.Initialize(db, platform, sec)
	annotationSvc := annotation.Initialize(db, platform, blob)
	predictionSvc := prediction.Initialize(db, platform, blob)
	experimentalSvc := experimental.Initialize(db, platform)

	projectSvc, err := project.Initialize(db, platform, blob, *cfg.App)
	if err != nil {
		return errors.Wrap(err, "could not instantiate project service")
	}

	contentSvc, err := content.Initialize(db, platform, blob)
	if err != nil {
		return errors.Wrap(err, "could not instantiate content service")
	}

	tagSvc, err := tag.Initialize(db, platform)
	if err != nil {
		return errors.Wrap(err, "could not instantiate tag service")
	}

	modelSvc, err := model.Initialize(db, platform, blob, *cfg.App)
	if err != nil {
		return errors.Wrap(err, "could not instantiate model service")
	}

	exportSvc, err := export.Initialize(db, platform, blob, cfg.App.ExportJobQueueName)
	if err != nil {
		return errors.Wrap(err, "could not instantiate export service")
	}
	datasetSvc, err := dataset.Initialize(db, platform)
	if err != nil {
		return errors.Wrap(err, "could not instantiate dataset service")
	}

	user.NewHTTP(userSvc, v1)
	annotation.NewHTTP(annotationSvc, v1)
	password.NewHTTP(passwordSvc, v1)
	project.NewHTTP(projectSvc, v1)
	content.NewHTTP(contentSvc, v1)
	tag.NewHTTP(tagSvc, v1)
	export.NewHTTP(exportSvc, v1)
	dataset.NewHTTP(datasetSvc, v1)
	model.NewHTTP(modelSvc, v1)
	prediction.NewHTTP(predictionSvc, v1)
	experimental.NewHTTP(experimentalSvc, v1)

	// API Docs
	echoServer.GET("/*", echo.WrapHandler(swaggerui.Handler()))

	// Start Service
	log.Infof("Starting Emerald Portal Service Version: %s", BuildVersion)

	echoServer.Start()

	// Exit blob service and wait for GC jobs to complete
	blob.Exit()

	return nil
}
