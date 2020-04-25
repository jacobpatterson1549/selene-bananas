var websocket = {
    
    _websocket: null,
    
    connect: function(event) {
        if (this._websocket != null) {
            return new Promise(true);
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
            this._websocket.onopen = event =>{
                console.log("websocket opened: ", event)
                resolve(true);
            };
            this._websocket.onerror = event =>{
                console.log("websocket error: ", event)
                resolve(false);
            };
        }) ;
    },

    close: function() {
        var websocketElement = document.getElementById("has-websocket");
        websocketElement.checked = false;
        if (this._websocket != null) {
            this._websocket.close();
        }
    },
};