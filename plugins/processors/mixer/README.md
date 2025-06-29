# Mixer Processor Plugin

The `mixer` processor consumes events from previous processors set in pipeline and writes it's all to one output channel that is an input for the next processors set.

It is a special plugin for cases when one of your processors in multiline configuration generates a lot of events than others and you want to spread them out evenly between next processors in pipeline:

```
       processors set one             processors set two
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
This plugin has no any specific configuration. Mixer also doesn't accept filters, but it can be assigned an alias and personal log level.
