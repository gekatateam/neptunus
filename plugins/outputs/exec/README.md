# Exec Output Plugin

The `exec` output executes configured command on each event.

## Configuration
```toml
[[outputs]]
  [outputs.exec]
    # the command that will be executed
    command = "bash"

    # list of fields that values will be used as command args
    # if field is []any, plugin adds each entry to args list
    # if field is map[string]any, plugin adds each key
    # and it's value to args list
    # otherwise, field will be used as is
    args = ["."]

    # exec timeout
    timeout = "10s"

    # a "env name <- label" map
    # each configured will be added to the environment 
    # of the process in the "env_name=label_value" form
    [outputs.exec.envlabels]
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
