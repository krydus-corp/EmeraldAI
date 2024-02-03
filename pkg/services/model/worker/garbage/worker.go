package garbage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	cloudwatch "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	sqs "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/aws/sqs"

	blob "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/blob"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	modelConfig "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/config"
	garbage "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage/garbage"
)

type WorkerPool struct {
	Platform *platform.Platform
	DB       *db.DB
	Blob     *blob.Blob
	Config   modelConfig.Configuration

	sagemakerClient  *sagemaker.Client
	cloudwatchClient *cloudwatch.Client
	consumer         sqs.Consumer

	garbage garbage.Garbage
}

func New(
	queueName string,
	blob *blob.Blob,
	db *db.DB,
	platform *platform.Platform,
	cfg modelConfig.Configuration,
) (*WorkerPool, error) {
	if queueName == "" {
		return nil, fmt.Errorf("queue name required")
	}

	consumer, err := sqs.NewConsumer(&sqs.Config{
		WorkerPool:        cfg.ModelService.GarbageWorkerConfig.Concurrency,
		VisibilityTimeout: 30,
	}, queueName)
	if err != nil {
		return nil, err
	}

	awsConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	sagemakerClient := sagemaker.NewFromConfig(awsConfig)
	cloudwatchClient := cloudwatch.NewFromConfig(awsConfig)

	garbage := garbage.New(&cfg.ModelService.GarbageConfig, sagemakerClient, cloudwatchClient, db, platform)

	return &WorkerPool{
		Blob:     blob,
		Platform: platform,
		DB:       db,
		Config:   cfg,

		sagemakerClient:  sagemakerClient,
		cloudwatchClient: cloudwatch.NewFromConfig(awsConfig),
		consumer:         consumer,

		garbage: *garbage,
	}, nil
}

func (w *WorkerPool) Start() {
	log.Infof("starting garbage collection service..")
	if w.Config.ModelService.GarbageConfig.EnableGarbageCollection {
		go w.garbageCollector()
	}
	go w.consumer.Consume(w.callback)
}

func (w *WorkerPool) garbageCollector() {
	ctx := context.Background()
	timeAwait := time.Now().Add(time.Minute * time.Duration(w.Config.ModelService.GarbageConfig.CollectionWaitTimeMinutes))

	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Infof("panic occured: ", r)
					}
					timeAwait = w.addWaitTime(timeAwait)
				}()

				waitUntil(ctx, timeAwait)

				log.Infof("garbage collecting endpoints..")
				if err := w.garbage.CleanEndpoints(); err != nil {
					log.Errorf("clean up endpoints error=%s", err.Error())
				}

				log.Infof("garbage collecting models..")
				if err := w.garbage.CleanModels(); err != nil {
					log.Errorf("clean up models error=%s", err.Error())
				}

				timeAwait = w.addWaitTime(timeAwait)
				log.Infof("done with garbage collection cycle. executing next at %s", timeAwait)
			}()
		}
	}()
}

// Add wait time if necessary.
func (w *WorkerPool) addWaitTime(awaitTime time.Time) time.Time {
	t := time.Now()
	if t.After(awaitTime) {
		return t.Add(time.Minute * time.Duration(w.Config.ModelService.GarbageConfig.CollectionWaitTimeMinutes))
	}
	return awaitTime
}

// Wait until specified execution time.
func waitUntil(ctx context.Context, until time.Time) {
	timer := time.NewTimer(time.Until(until))
	defer timer.Stop()

	select {
	case <-timer.C:
		return
	case <-ctx.Done():
		return
	}
}

func (w *WorkerPool) callback(msg *sqs.Message) error {
	var event Event
	if err := msg.Decode(&event); err != nil {
		log.Errorf("unable to decode train event; msg=%s error=%s", string(msg.Body()), err.Error())
		return nil
	}
	log.Debugf("Received message=%s", event.ToJSON())

	// -------------- DELETE EVENT -------------- //
	if event.Action == string(ActionDelete) {
		// Delete sagemaker model resources
		if err := w.garbage.Delete(event.ModelName, event.EndpointName); err != nil {
			log.Errorf("error deleting model resources; model=%s user=%s error=%s", event.ModelName, event.UserID, err)
			return nil
		}

		// Delete model predictions
		if err := w.Platform.PredictionDB.DeleteMany(w.DB, event.UserID, event.ModelID); err != nil {
			log.Errorf("error deleting model predictions; modelid=%s user=%s error=%s", event.ModelID, event.UserID, err)
		}
	}
	return nil
}
