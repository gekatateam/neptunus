# neptunus
Event-driven data processing pipelines engine  
  
The engine supports six types of plugins:
 - [inputs](plugins/inputs/) - event consumers
 - [processors](plugins/processors/) - event transformers
 - [outputs](plugins/outputs/) - event producers
 - [filters](plugins/filters/) - event routers by conditions
 - [parsers](plugins/parsers/) - converters of raw data into events
 - [serializers](plugins/serializers/) - converters of event into raw data

Neptunus engine supports multiple pipelines at once. Typical pipeline consists of at least one input, at least one output and, not necessarily, processors.  
Processors can be scaled to multiple `lines` - parallel streams - for cases when events are consumed and produced faster than they are transformed in one stream.  
> **Important!** Experimentally founded that scaling can reduce performance if processors cumulatively process events faster than outputs send them (because of filling channels buffers). Use it after testing it first.  

# Configuration
Neptunus configuration consists of two parts:
 - application configuration for daemon mode
 - pipelines that are run by demon

Each of them can be written in `json`, `yaml` or `toml` format. Pipelines reads at startup from configured storage.

## Daemon config
Here is a full daemon configuration example:
```toml
[common]
  log_level = "info"
  log_format = "logfmt"
  http_port = ":9600"
  [common.log_fields]
    runner = "local"

[pipeline]
  storage = "fs"
  [pipeline.fs]
    directory = ".pipelines"
    extention = "toml"
```

Common section:
 - `log_level` - logging level, accepts `trace`, `debug`, `info`, `warn`, `error` and `fatal`
 - `log_format` - in which format to write logs, accepts `logfmt` and `json`
 - `http_port` - `address:port` binding for internal http server
 - `log_fields` - a map with fields that will be added to every log entry

Pipelines engine section:

Конфигурация делится на две части - на конфигурацию приложения и на пайплайны.  
Пайплайн состоит из хотя бы одного входа, хотя бы одного выхода и, опционально процессоров. Каждому плагину можно назначить несколько фильтров, но только по одному каждого типа.  
Фильтр имеет один вход и два выхода - для прошедших фильтрацию и для отклоненных событий. В случае процессоров, отклоненные события отправляются дальше по пайплайну, не доходя до плагина. В случае входов и выходов, отклоненные события удаляются.  
Входные и выходные плагины могут требовать (но не обязательно требуют, зависит от плагина) указания парсера/сериализатора.  
  
Опции командной строки:
 - `neptunus run --config config.toml` - запуск демона с параметрами из конфигурации
 - `neptunus test --config config.toml` - запуск тестов сконфигурированных пайплайнов
 - `neptunus pipeline [--server-adderss ADDRESS] SUBCOMMAND` - команды для управления пайплайнами на сервере

Подробная информация об опциях - `neptunus [COMMAND [SUBCOMMAND]] -h`.

# TODO
## Common
 - [x] реализовать пайплайн менеджер для управления пайплайнами (старт/стоп/обновление по сигналу)
 - [ ] написать структуру, реализующую батчинг для плагинов ИЛИ плагин с примером реализации
 - [ ] продумать контроль доставки, трассировку событий и необходимость DLQ
 - [x] добавить плагины-парсеры для входов
 - [x] добавить плагины-сериализаторы для выходов (тут можно обойтись гошными темплейтами?)
 - [ ] исследовать работу с бинарными данными (энкодинг, декодинг, потребности в обработке)
 - [ ] написать юниты строгой консистентности

## Inputs
 - [ ] кафка, чтение из партиции, чтение в группе консьюмеров
 - [ ] рэббит

## Outputs
Должны поддерживать буферизацию и параметризируемый батчинг
 - [ ] кафка
 - [ ] рэббит
 - [ ] эластик
 - [ ] постгрес

## Filters
 - [x] глобы для роутинг кея, полей и лейблов для строк
 - [ ] больше (или равно), меньше (или равно), равно для чисел

## Processors
 - [ ] регулярки
 - [ ] конвертация между типами (строки, числа, время, байты, лейблы, теги, ключ маршрутизации, поля)
 - [ ] установка значений полей по умолчанию
 - [ ] математические операции
 - [ ] старларк

## Менеджмент пайплайнов
 - [ ] Реализовать возможность хранить пайпланйы в разных хранилищах:
   - [x] файлы (уже реализовано сейчас, но описание пайплайна и его конфигурация оторваны друг от друга, поэтому рефакторинг)
   - [ ] база данных
   - [ ] КВ сторадж, например, консула

 - [x] Реализовать работающий с стораджем менеджер пайплайнов. Менеджер пайплайнов должен уметь:
   - запускать пайплайны, вычитывая их из стораджа
   - останавливать конкретный пайплайн
   - запускать конкретный пайплайн
   - сохранять в сторадж новый пайплайн
   - обновлять/удалять конкретный пайплайн
  
 - [x] Реализовать REST апи над менеджером пайплайнов. Апи должен иметь ручки для остановки, запуска, обновления (с опциональным перезапуском), удаления с остановкой, сохранения нового (с опциональным запуском).

 - [ ] Реализовать gRPC апи над менеджером пайплайнов аналогично REST
  
 - [ ] Реализовать фронты для управления пайплайнами, которые будут ходить в апи и делать указанное.
  
 - [ ] **(?)** Реализовать межсервисное взаимодействие (одноранговый кластер) через gRPC.
