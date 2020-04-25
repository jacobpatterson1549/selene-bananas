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
            this._websocket.onopen = event =>{
                console.log("websocket opened: ", event)
                resolve(true);
            };
            this._websocket.onerror = event =>{
                console.log("websocket error: ", event)
                resolve(false);
            };
        }) ;
    }
};