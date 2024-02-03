/*
 * File: train.go
 * Project: train
 * File Created: Tuesday, 16th August 2022 6:21:54 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package train

import (
	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	train "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/sage/train"
)

func (w *WorkerPool) train(config *train.Config, algorithm models.ProjectAnnotationType) (trainingJobName string, metrics map[string]interface{}, err error) {
	// Create job spec
	trainer, err := train.New(config, w.sagemakerClient, algorithm)
	if err != nil {
		return "", nil, err
	}

	if err := trainer.Train(config.NumClasses, config.NumTrainingSamples, config.NumValidationSamples, config.ForcePaddingLabelWidth); err != nil {
		return "", nil, err
	}
	log.Debugf("created hyperparameter tuning job: tunningJobName=%s", *trainer.Input.HyperParameterTuningJobName)

	hyperParameterTrainingJobSummary, err := trainer.PollForStatus()
	if err != nil {
		return "", nil, err
	}

	log.Debugf("hyperparameter tuning job complete: selectedTrainingJobName=%s", *hyperParameterTrainingJobSummary.TrainingJobName)

	metrics, err = trainer.Metrics(*hyperParameterTrainingJobSummary.TrainingJobName)
	if err != nil {
		return "", nil, err
	}
	return *hyperParameterTrainingJobSummary.TrainingJobName, metrics, nil
}
