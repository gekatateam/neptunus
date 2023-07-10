# Neptunus HTTP Api

## Pipelines Controller

#### GET `/api/v1/pipelines/`
Get pipelines from configured storage

Request body: **No**.

Response:
 - **200** - Ok;
 - **500** - Any error.

#### POST `/api/v1/pipelines/`
Create new pipeline

Request body: pipeline in json format.

Response:
 - **201** - Pipeline created;
 - **400** - Json unmarshal failed or invalid pipeline provided;
 - **500** - Other errors.

#### GET `/api/v1/pipelines/{pipelineId}`
Get pipeline configuration

Request body: **No**.

Response:
 - **200** - Ok;
 - **500** - Any error.

#### POST `/api/v1/pipelines/{pipelineId}`
Update pipeline configuration by id

> **Note:** Only stopped pipeline can be updated

Request body: pipeline in json format.

Response:
 - **200** - Pipeline updated;
 - **400** - Json unmarshal failed or invalid pipeline provided;
 - **404** - Pipeline with provided `pipelineId` was not found;
 - **409** - Pipeline in illegal state;
 - **500** - Other errors.

#### DELETE `/api/v1/pipelines/{pipelineId}`
Delete pipeline by id

> **Note:** Only stopped pipeline can be deleted

Request body: **No**.

Response:
 - **200** - Pipeline deleted;
 - **404** - Pipeline with provided `pipelineId` was not found;
 - **409** - Pipeline in illegal state;
 - **500** - Other errors.

#### GET `/api/v1/pipelines/{pipelineId}/state`
Get pipeline state

Request body: **No**.

Response:
 - **200** - Ok;
 - **404** - Pipeline with provided `pipelineId` was not found;
 - **500** - Other errors.

#### POST `/api/v1/pipelines/{pipelineId}/start`
Start pipeline by id

> **Note:** Only stopped pipeline can be started

Request body: **No**.

Response:
 - **200** - Ok;
 - **404** - Pipeline with provided `pipelineId` was not found;
 - **409** - Pipeline in illegal state;
 - **500** - Other errors.

#### POST `/api/v1/pipelines/{pipelineId}/stop`
Stop pipeline by id

> **Note:** Only running pipeline can be stoppped

Request body: **No**.

Response:
 - **200** - Ok;
 - **404** - Pipeline with provided `pipelineId` was not found;
 - **409** - Pipeline in illegal state;
 - **500** - Other errors.

## Debug

#### GET `/metrics`
Neptunus metrics in Prometheus format.

#### GET `/debug/pprof/*`
[Runtime profiling](https://pkg.go.dev/net/http/pprof).
