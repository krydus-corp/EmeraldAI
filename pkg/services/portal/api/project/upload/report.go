/*
 * File: report.go
 * Project: upload
 * File Created: Wednesday, 4th January 2023 4:31:19 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package upload

type UploadErr struct {
	Error         string
	InternalError string
	Filename      string
	Message       string
}

const (
	UploadErrUnknown         = "UploadErrUnknown"
	UploadErrMaxSizeExceeded = "UploadErrMaxSizeExceeded"
	UploadErrIO              = "UploadErrIO"
	UploadErrStats           = "UploadErrStats"
	UploadErrInvalidFormat   = "UploadErrInvalidFormat"
	UploadErrProcessing      = "UploadErrProcessing"
	UploadErrAnnotation      = "UploadErrAnnotation"
)

// Report represents report job
//
// swagger:model Report
type Report struct {
	// Labels file associated with Upload
	//
	LabelsFile string `json:"labels_file"`
	// Files that failed to upload
	//
	Errors []UploadErr `json:"errors"`
	// Total bytes
	//
	TotalBytes int64 `json:"total_bytes"`
	// Total files
	//
	TotalFiles int64 `json:"total_files"`
	// Total files that failed
	//
	TotalFilesFailed int `json:"total_files_failed"`
	// Total files that succeeded
	//
	TotalImagesSucceeded int `json:"total_files_succeeded"`
	// Total duplicate images
	//
	TotalFilesDuplicate int `json:"total_files_duplicate"`
}

func NewReport(labelsFile string, totalFiles int) *Report {
	return &Report{
		LabelsFile: labelsFile,
		TotalFiles: int64(totalFiles),
		Errors:     []UploadErr{},
	}
}

func (r *Report) AddErr(internalErr error, err, filename, message string) {
	internalErrString := ""
	if internalErr != nil {
		internalErrString = internalErr.Error()
	}
	r.Errors = append(r.Errors, UploadErr{Error: err, Filename: filename, Message: message, InternalError: internalErrString})
	r.TotalFilesFailed++
}
