# Exec Processor Plugin

The `exec` processor executes configured command on each event.

## Configuration
```toml
[[processors]]
  [processors.exec]
    # the command that will be executed
    command = "bash"

    # list of fields that values will be used as command args
    # if field is []any, plugin adds each entry to args list
    # if field is map[string]any, plugin adds each key
    # and it's value to args list
    # otherwise, field will be used as is
    args = []

    # event field, which value will be passed to process stdin
    stdin_field = ""

    # command working directory
    dir = ""

    # path to set process exit code
    exec_code_to = "exec.code"

    # path to set process combined output
    exec_output_to = "exec.output"

    # exec timeout
    timeout = "10s"

    # command envs (os.Environ() already included)
    # each configured will be added to the environment 
    # of the process in the "env_name=label_value" form
    [processors.exec.envs]
      EXEC_CONTEXT = "neptunus-processor"

    # a "env name <- label" map
    # each configured will be added to the "envs"
    # in the "env_name=label_value" form
    [processors.exec.envlabels]
      CATCH_PHRASE = "phrase"
```

## Example
```json
# simple echo
{
  "routing_key": "exec.bash.echo",
  "labels": {
    "phrase": "Hi, I'm Elfo"
  },
  "data": [
    "-c",
    "echo $CATCH_PHRASE"
  ]
}
```
