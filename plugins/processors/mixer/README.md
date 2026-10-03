# Mixer Processor Plugin

The `mixer` processor consumes events from the previous processors set in the pipeline and writes them all to one output channel, which is the input for the next processors set.

It is a special plugin for cases where one of your processors in a multiline configuration generates many more events than the others, and you want to spread them out evenly across the next processors in the pipeline:

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
This plugin has no specific configuration. The mixer also does not accept filters, but an alias and a custom log level can be assigned.
