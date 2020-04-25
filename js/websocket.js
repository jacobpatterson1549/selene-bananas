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
                console.log("websocket opened");
                resolve(true);
            };
            this._websocket.onerror = event => {
                console.log("websocket error: ", event);
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
        console.log("sending message json:", messageJSON);
        this._websocket.send(messageJSON);
    },

    onMessage: function (event) {
        console.log("received message: " + event.data);
        var message = JSON.parse(event.data);

        // handling
        if (!message.type) {
            console.log('unknown message received:', event.data);
        }

        switch (message.type) {
            case 11: // gameInfos
                lobby.setGameInfos(message.gameInfos);
                break;
            case 14: // socketInfo
                console.log("info:", message.info); // TODO: DELETEME (handle all infos correctly)
                // TODO: put in output window
                break;
            case 15: // socketError
                console.log("error:", message.info);
                // TODO: put in output window
                break;
            default:
                console.log('unknown message type received:', event.data);
                break;
        }
    },
}