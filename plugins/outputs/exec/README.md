# Exec Output Plugin

The `exec` output executes configured command on each event.

## Configuration
```toml
[[outputs]]
  [outputs.exec]
    # the command that will be executed
    command = "bash"

    # list of labels, each will be added to the environment 
    # of the process in the "label_key=label_value" form
    envs = []

    # list of fields that values will be used as command args
    # if field is []any, plugin adds each entry to args list
    # if field is map[string]any, plugin adds each key
    # and it's value to args list
    # otherwise, field will be used as is
    args = ["."]
```

## Example
```json
# simple echo
{
  "routing_key": "exec.bash.echo",
  "labels": {
    "CATCH_PHRASE": "Hi, I'm Elfo"
  },
  "data": [
    "-c",
    "echo $CATCH_PHRASE"
  ]
}
```
