package dynamicgrpc
// as CLIENT:
// - if enary - send every event
// - if client stream - send all events in batch
//
// as SERVER
// - if server stream - send every event to all connected clients