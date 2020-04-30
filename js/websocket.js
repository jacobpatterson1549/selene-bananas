var websocket = {

    _websocket: null,

    connect: function (event) {
        if (this._websocket != null) {
            return new Promise(resolve => { resolve(true); });
        }
        var form = event.target;
        var url = form.action;
        url = url.replace(/^http/, "ws");
        var jwt = content.getJWT();
        url += "?access_token=" + jwt
        return new Promise(resolve => {
            this._websocket = new WebSocket(url);
            var websocketElement = document.getElementById("has-websocket");
            websocketElement.checked = true;
            this._websocket.onopen = event => {
                resolve(true);
            };
            this._websocket.onerror = event => {
                log.error("websocket error: ", event);
                resolve(false);
            };
            this._websocket.onmessage = this.onMessage;
        });
    },

    close: function () {
        var websocketElement = document.getElementById("has-websocket");
        websocketElement.checked = false;
        if (this._websocket != null) {
            this._websocket.close();
            this._websocket = null;
        }
    },

    send: function (message) {
        var messageJSON = JSON.stringify(message);
        if (this._websocket != null && this._websocket.readyState == 1) { // OPEN
            this._websocket.send(messageJSON);
        } else {
            log.error("websocket not open, closing");
            this.close();
        }
    },

    onMessage: function (event) {
        var message = JSON.parse(event.data);
        switch (message.type) {
            case 11: // gameInfos
                lobby.setGameInfos(message.gameInfos);
                break;
            case 14: // socketInfo
                if (message.gameState != null) {
                    game.setState(message.gameState);
                }
                if (message.tilesLeft != null) { // keep after game.setState()
                    game.setTilesLeft(message.tilesLeft);
                }
                if (message.gamePlayers != null) {
                    game.setPlayers(message.gamePlayers);
                }
                if (message.tilePositions != null) {
                    game.replaceGameTiles(message.tiles, message.tilePositions)
                    break;
                }
                log.info(message.info);
                if (message.tiles != null) {
                    game.addUnusedTiles(message.tiles);
                    if (message.tilesLeft == null) { // the server will not send a tilesLeft = 0 because that is the empty value
                        game.setTilesLeft(0);
                    }
                }
                break;
            case 15: // socketError
                log.error(message.info);
                break;
            case 16: // socketClosed
                lobby.leave();
                break;
            case 17: // socketHTTPPing
                var pingFormElement = document.getElementById("ping-form");
                var event = {
                    preventDefault: () => {},
                    target: pingFormElement,
                }
                pingFormElement.onsubmit(event);
                break;
            default:
                log.error('unknown message type received - message:', event.data);
                break;
        }
    },
}