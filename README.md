# Emerald AI

<p align="center">
  <img width="500" src="assets/Emerald AI Logo_A Solid Multi.png">
</p>

> Data wrangling made easy
> _- AI driven data cleaning and attribution_

## Getting Started

1. Install project dependencies:

   - Install [**Docker >= 20.10.8**](https://docs.docker.com/install/)
   - Install [**AWS CLI >= 2.2.31**](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)
   - Install [**Go**](https://golang.org/doc/install) (optional if not developing)
   - Install [**GoSwag**](https://goswagger.io/install.html) (optional if not developing)
   - Install [**staticcheck**](https://staticcheck.io/docs/getting-started/) (optional if not developing)

2. Build the project: `make build`
3. Create a new config based on `config/.env.example` e.g. `config/.env.<user>`.
   - Replace each instance of `example` in the example config to match the name of your environment
   - Specify a random `JWT_SECRET` (min length 64 characters)
   - Update the AWS credential settings
   - For local deployments, set `MONGO_URI=mongodb://mongo:27017`
4. Run the services: `MONGO_PROFILE=localdb ENV=config/.env.<env> make up`

> _Note: If needing to deploy to ECS and on a Linux OS, the Docker Compose plugin will need to be installed as it is not native to Docker yet on Linux. \
> Instructions to install the plugin can be found [**here**](https://docs.docker.com/compose/cli-command/)_

## Directives

### Environment

This project makes use of an environmental file for establishing necessary deployment variables. There is an example environment file provided with descriptions of the various settings at `<project-root>/config/.env.example`. This file **should not** be modified. Create a new file in the same directory e.g. `.env.bob` and provide the settings appropriate for your environment.

This file should be passed using the environmental variable, `ENV`. As an example:

```bash
ENV=config/.env make up
```

### Database

This project is configured to use either local or cloud (Atlas) Mongo instances. For local Mongo instances, configure the `MONGO_URI` setting in `.env.<env>` to `mongodb://mongo:27017` and prefix make directives with `MONGO_PROFILE=localdb`. For cloud (Atlas) instances, configure the `MONGO_URI` to be the full Atlas connection URI e.g. `MONGO_URI=mongodb+srv://root:emld_mongo01@stage-serverless.cqcqn.mongodb.net/?retryWrites=true&w=majority`. The `MONGO_PROFILE` prefix is not required for cloud instances.

#### Manual Migration

The script located `<project-root>/scripts/mongodb_export.sh` can be used to manually import / export collections in the database.

**Example Export:**

```sh
export DEV=mongodb://admin:<mongo-password>@<mongodb-url>:27017
./scripts/mongodb_export.sh --export $DEV emld01 "user tag project content dataset upload"
./scripts/mongodb_export.sh --export $DEV casbin "casbin_rule"
```

**Example Import:**

```sh
export DEV=mongodb://admin:<mongo-password>@<mongodb-url>:27017
./scripts/mongodb_export.sh --import $DEV emld01 "user tag project content dataset upload"
./scripts/mongodb_export.sh --export $DEV casbin "casbin_rule"
```

### Build-Local

> **This directive is recommended primarily for local developing & testing**

This directive will compile and install all all package binaries to the `project-root/build/bin/<arch>` folder.

```bash
make build-local
```

An individual service can then be run via:

```bash
CONFIG_PATH="s3://emld-configuration-store/config.portal.yml" go run cmd/emld-portal-svc/main.go
```

### Build-Docker

This directive uses multi-stage builds to build each service in Docker.

```bash
make build or make build-docker
```

### Build-Docker Arm64

This directive uses multi-stage builds to build each service in Docker for arm64.

```bash
make build-arm
```

### Push

Push Docker images to the Elastic Container Registry. This is only required for ECS deployments.

```bash
make push
```

### Up

Runs the project services locally.

```bash
make up
```

### Down

Tears down the project services locally.

```bash
make down
```

### ECS-Generate

Generates the Cloudformation template used for provisioning infrastructure in AWS (ECS cluster, S3, projects, SQS queues, etc.). The output of this directive is a Cloudformation template at `/tmp/emld_cf.yml`.

```bash
make ecs-generate
```

### ECS-Up

```bash
make ecs-up
```

Deploy Cloudformation stack to AWS. This directive will wait until the cluster is deployed.

### ECS-Update (untested)

Update the Cloudformation stack. This directive will update the cluster in the background.

```bash
make ecs-update
```

### ECS-Down

Delete the Cloudformation stack and all associated resources. Note that EFS stores will not be deleted automatically and must be torn down manually before deploying to the same environment.

```bash
make ecs-down
```

### ECS-Describe

Describe the Cloudformation stack associated with the deployment. This is useful for determining if an update or stack delete command had completed.

```bash
make ecs-describe
```

### Deploy

This is a convenience directive for deploying Emerald to an ECS cluster. It will:

- Build all docker images
- Push to the Elastic Container Registry
- Generate and upload configuration files
- Generate Cloudformation stack
- Deploy Cloudformation stack

#### Deploy from AMD64

```bash
MONGO_PROFILE=localdb ENV=config/.env.<env> make deploy
```

#### Deploy from ARM64

```bash
export DOCKER_DEFAULT_PLATFORM=linux/amd64
MONGO_PROFILE=localdb ENV=config/.env.<env> make deploy-arm
```

### Update (deprecated and untested)

This is a convenience directive for updating an Emerald ECS cluster. It will:

- Build all docker images
- Push to the Elastic Container Registry
- Generate and upload configuration files
- Generate Cloudformation stack
- Update Cloudformation stack

```bash
make update
```

### Update (new manual process)

- Build and Push: `make build && MONGO_PROFILE=localdb ENV=config/.env.<env> make push`
- Navigate to the cluster you want to update:

    <img width="500" src="assets/README/ecs.png">

- Navigate to the service and click on 'tasks':

    <img width="500" src="assets/README/tasks.png">

- Click on the task definition for the service and click 'Create new revision'

    <img width="500" src="assets/README/task_definition.png">

- In the 'Create new revision of Task Definition ' window, ensure the 'Task definition name' and 'Task role' match the environment (e.g. dev) and service (e.g. model) you are updating.

    <img width="500" src="assets/README/new_definition-1.png">

- Under 'Container definitions', click the container definition for the service you are updating (e.g. dev)

    <img width="500" src="assets/README/new_definition-2.png">

- In a new window, navigate to Amazon ECR. Locate the latest image for the service you are updating (e.g. model)

    <img width="500" src="assets/README/ecr.png">

- Back in the 'Edit container' window in ECS, update the image uri and digest (`<uri>@<digest>`) with the latest version/hash found in the previously opened ECR window. Update the `VERSION` environmental variable as well. Click 'Update'

    <img width="500" src="assets/README/new_definition-3.png">
    <img width="500" src="assets/README/new_definition-4.png">

- Once the container difinition is updated, click 'Update' in the 'Create new revision of Task Definition' window to complete the task definition update.
- Next, click 'Actions' > 'Update Service'

    <img width="500" src="assets/README/update-service-1.png">

- In the 'Configure service' window, update the following:
  - Operating system family: Linux
  - Force new deployment: Checked
  - Cluster: ensure this points to the correct environment (e.g. emerald-dev for dev cluster)
  - Service name: ensure this points to the correct environment and service

    <img width="500" src="assets/README/update-service-2.png">

- Proceed clicking 'Next step' until the 'Review' window appears. Click 'Update Service'.
- In the service deinfition page, click 'Tasks'. Click on the currently running task and then stop the task. This will cause a new task to be spun up using th new task definition we created earlier.

    <img width="500" src="assets/README/bounce-1.png">
    <img width="500" src="assets/README/bounce-2.png">

## Test

Run project unit tests.

```bash
make test
```

## Services

---

### Portal

The Portal Service is responsible for all user, project, and content coordination. Portal hosts the public facing API for Emerald. Casbin is used for RBAC policies. Currently, there are admin and user roles. Admin accounts have access to user CRUD routes while User accounts do not. More information on the Casbin Middleware can be found in the [documentation](https://casbin.org/docs/en/management-api)

#### Swagger API Documentation

> _Note: If running in Docker, Swagger docs are automatically generated._

To generate Swagger docs, run `make swag`

This will compile swagger documentation and serve them via a static route in the Emerald API. To view the docs, start the service and navigate to `http://<emerald-ip>:8080/`

#### DB Migration

> _Note: If running in Docker, migration is done automatically._

If starting Emerald for the first time, insert an admin user by running the Portal DB Migration application: `make migrate`. The Portal Service and Mongo will need to be running to migrate the database. If wanting to change the default username/password (admin/admin), you can run the migrate command manually via `docker exec -it portal ./migrate -help` See the output of this help command for setting specific user defaults.

### Model

This service interacts with the EmeraldAI model Engine. The service is used internally by the Portal service for model training and inference. Users interact with this service through requests that are proxied through the Portal.

### Exporter

The Exporter service is a lightweight Lmabda task that asyncronously packages up an export of any combination of the project (data), dataset (annotations and statistics), and model (inference & training data).

## Upgrading SwaggerUI

SwaggerUI can be downloaded from their [GitHub Repo Releases page](https://github.com/swagger-api/swagger-ui/releases/latest). Once downloaded, place the contents of dist folder in the root of this Go project and rename to `swaggerui`.

## Scaling to Zero

If needing to stop the tasks in a cluster while still preserving the data, stop each individual task by going to the console, looking up the name for each service in the cluster, and run:

```sh
 aws ecs update-service --cluster emerald-dev --service <name-of-service> --desired-count 0
 ```
