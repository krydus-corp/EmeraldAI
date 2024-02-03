package config

import (
	cache "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/cache"
	logger "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// Configuration holds data necessary for configuring application
type Configuration struct {
	Server    *Server                 `yaml:"server,omitempty"`
	DB        *platform.Configuration `yaml:"database,omitempty"`
	BlobStore *blob.Configuration     `yaml:"blob_store,omitempty"`
	JWT       *JWT                    `yaml:"jwt,omitempty"`
	App       *Application            `yaml:"application,omitempty"`
	Logger    *logger.Configuration   `yaml:"log,omitempty"`
	Cache     *cache.Config           `yaml:"cache,omitempty"`
}

// Server holds data necessary for server configuration
type Server struct {
	Port              string `yaml:"port,omitempty"`
	Debug             bool   `yaml:"debug,omitempty"`
	Timeout           int    `yaml:"timeout_seconds,omitempty"`
	RBACModelFile     string `yaml:"rbac_model_file,omitempty"`
	RBACPolicyFile    string `yaml:"rbac_policy_file,omitempty"`
	Domain            string `yaml:"domain,omitempty"`
	CertCacheS3Bucket string `yaml:"cert_cache_s3_bucket,omitempty"`
	CrtFile           string `yaml:"crt_file"`
	KeyFile           string `yaml:"key_file"`
}

// JWT holds data necessary for JWT configuration
type JWT struct {
	MinSecretLength           int    `yaml:"min_secret_length,omitempty"`
	AccessDurationMinutes     int    `yaml:"access_duration_minutes,omitempty"`
	RefreshDurationMinutes    int    `yaml:"refresh_duration_minutes,omitempty"`
	AutoLogOffDurationMinutes int    `yaml:"auto_logoff_duration_minutes,omitempty"`
	SigningAlgorithm          string `yaml:"signing_algorithm,omitempty"`
}

// Application holds application configuration details
type Application struct {
	MinPasswordStr                  int    `yaml:"min_password_strength,omitempty"`
	SendGridPasswordResetTemplateID string `yaml:"sendgrid_password_reset_template_id,omitempty"`
	SendGridEmailConfirmTemplateID  string `yaml:"sendgrid_email_confirm_template_id,omitempty"`
	PassResetCodeExpiration         int    `yaml:"password_reset_expiration,omitempty"`
	PassResetEmail                  string `yaml:"password_reset_email,omitempty"`
	PassResetSubject                string `yaml:"password_reset_subject,omitempty"`

	// SQS Queues
	ExportJobQueueName   string `yaml:"export_job_queue_name,omitempty"`
	UploadJobQueueName   string `yaml:"upload_job_queue_name,omitempty"`
	TrainJobQueueName    string `yaml:"train_job_queue_name,omitempty"`
	EndpointJobQueueName string `yaml:"endpoint_job_queue_name,omitempty"`
	BatchJobQueueName    string `yaml:"batch_job_queue_name,omitempty"`
	GarbageJobQueueName  string `yaml:"garbage_job_queue_name,omitempty"`
}

// Cache holds cacheing layer configuration details
type Cache struct {
	Host string `yaml:"host,omitempty"`
	Port int    `yaml:"port,omitempty"`
}
