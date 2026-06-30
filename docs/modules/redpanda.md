# Redpanda

`redpanda` дает общую обертку над `kafka-go` для сервисов Endge.

## Что входит

- `Client` с единым логированием
- `NewReader` и `NewWriter`
- `NewMessage` с автоматической инъекцией trace headers
- `ExtractContext` для восстановления trace context при consume

## Ограничение

Пакет не прячет consumer loop и не превращается в framework. Он покрывает только повторяемую транспортную инфраструктуру.
