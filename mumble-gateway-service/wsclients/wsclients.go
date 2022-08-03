package wsclients

// Client list placed inside a redundant package to escape circular dep between rbmq and websocket pkg

import "mumble-gateway-service/types"

var WsClients *types.WsClients
