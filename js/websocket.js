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
        return new Promise((resolve, reject) => {
            this._websocket = new WebSocket(url);
            var websocketElement = document.getElementById("has-websocket");
            websocketElement.checked = true;
            this._websocket.onopen = event => {
                resolve();
            };
            this._websocket.onerror = event => {
                log.error("websocket error - check browser console");
                reject();
            };
            this._websocket.onclose = event => {
                this._close(false);
            };
            this._websocket.onmessage = this.onMessage;
        });
    },

    close: function () {
        this._close(true)
    },

    _close: function (expected) {
        if (this._websocket == null) {
            return;
        }
        var websocketElement = document.getElementById("has-websocket");
        websocketElement.checked = false;
        var hasGameElement = document.getElementById("has-game");
        hasGameElement.checked = false;
        var lobbyTab = document.getElementById("tab-4");
        lobbyTab.checked = true;
        this._websocket.onclose = null;
        this._websocket.close();
        this._websocket = null;
        if (!expected) {
            log.error("lobby closed");
        }
    },

    send: function (message) {
        if (this._websocket != null && this._websocket.readyState == 1) { // OPEN
            var messageJSON = JSON.stringify(message);
            this._websocket.send(messageJSON);
        }
    },

    onMessage: function (event) {
        var message = JSON.parse(event.data);
        switch (message.type) {
            case 3: // gameLeave
                game.leave();
                if (message.info) {
                    log.info(message.info);
                }
                break;
            case 4: // gameDelete
                lobby.leave();
                if (message.info) {
                    log.info(message.info);
                }
                break;
            case 10: // BoardRefresh
                game.replaceGameTiles(message.tiles, message.tilePositions);
                break;
            case 11: // gameInfos
                lobby.setGameInfos(message.gameInfos);
                break;
            case 13: // playerDelete
                lobby.leave();
                if (message.info) {
                    log.error(message.info);
                }
                break;
            case 14: // socketInfo
                if (message.gameStatus != null) {
                    game.setStatus(message.gameStatus);
                }
                if (message.tilesLeft != null) { // keep after game.setState()
                    game.setTilesLeft(message.tilesLeft);
                }
                if (message.gamePlayers != null) {
                    game.setPlayers(message.gamePlayers);
                }
                if (message.tilePositions != null) {
                    game.replaceGameTiles(message.tiles, message.tilePositions);
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
            case 17: // socketHTTPPing
                var pingFormElement = document.getElementById("ping-form");
                var pingEvent = {
                    preventDefault: () => { },
                    target: pingFormElement,
                }
                pingFormElement.onsubmit(pingEvent);
                break;
            case 19: // gameChatSend
                log.chat(message.info);
                break;
            default:
                log.error('unknown message type received - message:' + event.data);
                break;
        }
    },
}