# Exec Lookup Plugin

The `exec` lookup executes configured command with args and stores the result as lookup data. This plugin requires parser and only first parsed event will be used.

## Configuration
```toml
[[lookups]]
  [lookups.exec]
    alias = "token.renew"

    # command to execute
    command = "kubectl"

    # command args, strings list
    args = []

    # command stdin
    stdin = ""

    # exec timeout
    timeout = "10s"

    # lookup update interval
    # if zero, plugin executes command only on pipeline startup
    interval = "30s"

    # command envs (os.Environ() already included)
    # each configured will be added to the environment 
    # of the process in the "env_name=env_value" form
    [lookups.exec.envs]
      CATCH_PHRASE = "phrase"

    [lookups.exec.parser]
      type = "json"
```
