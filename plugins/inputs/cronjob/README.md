# Cronjob Input Plugin

The `cronjob` input plugin generates events using the [robfig/cron](https://github.com/robfig/cron) package. Each job produces one event with `cronjob.%job name%` as its routing key.

A job's `schedule` parameter accepts an expression in the [format used by the library](https://pkg.go.dev/github.com/robfig/cron/v3#hdr-CRON_Expression_Format), with seconds in the first field.

## Configuration
```toml
[[inputs]]
  [inputs.cronjob]
    # cron location in IANA format
    # see more in https://pkg.go.dev/time#LoadLocation
    location = "UTC"

    # a list of scheduled jobs
    [[inputs.cronjob.jobs]]
      # job name, used in routing key of generated event
      name = "partition.create"
      # job schedule
      schedule = "0 0 0 * * *"
      # if true, job is forced at startup
      force = false

    [[inputs.cronjob.jobs]]
      name = "healthcheck.notify"
      schedule = "@every 10s"
      force = true
```
