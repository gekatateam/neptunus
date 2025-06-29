# Mixer Processor Plugin

The `mixer` processor consumes events from all previous processors in pipeline and writes it's all to one output channel that is an input for all processors after.

It is a special plugin for cases when one of your processors in multiline configuration generates a lot of events than others and you want to spread them out evenly between next processors in pipeline:

```
            ┌────┐                         ┌────┐ 
1 event   >─┤proc├─┐         ┌> 22 events >┤proc│ 
            └────┘ |         │             └────┘ 
            ┌────┐ |┌───────┐│             ┌────┐ 
1 event   >─┤proc├─┼┤ Mixer ├┼> 22 events >┤proc│ 
            └────┘ |└───────┘│             └────┘ 
            ┌────┐ |         │             ┌────┐ 
64 events >─┤proc├─┘         └> 22 events >┤proc│ 
            └────┘                         └────┘ 
```

## Configuration
```toml
[[processors]]
  [processors.mixer]
```
This plugin has no any specific configuration. Mixer also doesn't accept filters, but alias can be assigned.
