/*
 * File: config.go
 * Project: endpoint
 * File Created: Tuesday, 16th August 2022 2:28:52 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package endpoint

type Config struct {
	ExecutionRoleArn string `yaml:"execution_role_arn,omitempty"`
	MemorySizeInMB   int32  `yaml:"memory_size_mb,omitempty"`
	MaxConcurrency   int32  `yaml:"max_concurrency,omitempty"`
	ResourceEnv      string `yaml:"resource_env"`
}
